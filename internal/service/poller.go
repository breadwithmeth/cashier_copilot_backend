package service

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"cashier_copilot_backend/internal/model"
	"cashier_copilot_backend/internal/repository"
)

// Poller runs background goroutines that periodically poll the database for new events.
// It replaces the role of NATS subscriptions — Python workers write directly to PostgreSQL,
// and these pollers read new records for processing.
type Poller struct {
	cvRepo        *repository.CvEventRepo
	speechRepo    *repository.SpeechRepo
	taskRepo      *repository.TaskRepo
	cameraRepo    *repository.CameraRepo
	violationRepo *repository.ViolationRepo

	ruleEngine *RuleEngine
	coPilot    *CoPilot
	fsm        *FSMManager
	hub        *Hub

	pollIntervalCv    time.Duration
	pollIntervalTasks time.Duration

	// Track the last processed IDs to avoid reprocessing
	mu                    sync.Mutex
	lastProcessedCvID     int64
	lastProcessedSpeechID int64

	logger *slog.Logger
}

// NewPoller creates a new Poller.
func NewPoller(
	cvRepo *repository.CvEventRepo,
	speechRepo *repository.SpeechRepo,
	taskRepo *repository.TaskRepo,
	cameraRepo *repository.CameraRepo,
	violationRepo *repository.ViolationRepo,
	ruleEngine *RuleEngine,
	coPilot *CoPilot,
	fsm *FSMManager,
	hub *Hub,
	pollIntervalCvMs int,
	pollIntervalTasksMs int,
	logger *slog.Logger,
) *Poller {
	return &Poller{
		cvRepo:            cvRepo,
		speechRepo:        speechRepo,
		taskRepo:          taskRepo,
		cameraRepo:        cameraRepo,
		violationRepo:     violationRepo,
		ruleEngine:        ruleEngine,
		coPilot:           coPilot,
		fsm:               fsm,
		hub:               hub,
		pollIntervalCv:    time.Duration(pollIntervalCvMs) * time.Millisecond,
		pollIntervalTasks: time.Duration(pollIntervalTasksMs) * time.Millisecond,
		logger:            logger,
	}
}

// StartAll launches all background pollers. Call with a cancellable context for graceful shutdown.
func (p *Poller) StartAll(ctx context.Context) {
	go p.pollCvEvents(ctx)
	go p.pollSpeechTranscripts(ctx)
	go p.pollCompletedTasks(ctx)

	p.logger.Info("all background pollers started",
		"cv_interval", p.pollIntervalCv,
		"tasks_interval", p.pollIntervalTasks,
	)
}

// pollCvEvents periodically reads new CV events from the database,
// updates the FSM, and triggers applicable rules.
func (p *Poller) pollCvEvents(ctx context.Context) {
	ticker := time.NewTicker(p.pollIntervalCv)
	defer ticker.Stop()

	p.logger.Info("cv_events poller started", "interval", p.pollIntervalCv)

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("cv_events poller stopped")
			return
		case <-ticker.C:
			p.processCvEventBatch(ctx)
		}
	}
}

// processCvEventBatch fetches and processes a batch of new CV events.
func (p *Poller) processCvEventBatch(ctx context.Context) {
	p.mu.Lock()
	afterID := p.lastProcessedCvID
	p.mu.Unlock()

	events, err := p.cvRepo.FetchNew(ctx, afterID)
	if err != nil {
		p.logger.Error("cv_events poller: failed to fetch", "error", err)
		return
	}

	if len(events) == 0 {
		return
	}

	p.logger.Debug("cv_events poller: processing batch", "count", len(events))

	for i := range events {
		event := &events[i]

		posID := p.resolvePosByCamera(ctx, event.CameraID)
		if posID == "" {
			p.logger.Warn("cv_events poller: could not resolve pos_id for camera",
				"camera_id", event.CameraID)
			continue
		}

		// Update FSM with CV event
		_, newState, _ := p.fsm.TransitionCvEvent(posID, event)

		// Run applicable rules based on event type
		switch event.EventType {
		case "item_in_bag":
			p.ruleEngine.CheckUnscannedItem(ctx, event, posID)
		case "hand_to_drawer":
			p.ruleEngine.CheckDrawerWithoutSale(ctx, event, posID)
		}

		// Check no_cashier_on_sale when entering Scanning or Payment
		if newState == model.StateScanning || newState == model.StatePayment {
			p.ruleEngine.CheckNoCashierOnSale(ctx, posID, event.TimestampMs)
		}

		// Update last processed ID
		p.mu.Lock()
		if event.ID > p.lastProcessedCvID {
			p.lastProcessedCvID = event.ID
		}
		p.mu.Unlock()
	}
}

// pollSpeechTranscripts periodically reads new speech transcripts.
func (p *Poller) pollSpeechTranscripts(ctx context.Context) {
	ticker := time.NewTicker(p.pollIntervalCv) // same interval as CV events
	defer ticker.Stop()

	p.logger.Info("speech_transcripts poller started", "interval", p.pollIntervalCv)

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("speech_transcripts poller stopped")
			return
		case <-ticker.C:
			p.processSpeechBatch(ctx)
		}
	}
}

// processSpeechBatch fetches and processes a batch of new speech transcripts.
func (p *Poller) processSpeechBatch(ctx context.Context) {
	p.mu.Lock()
	afterID := p.lastProcessedSpeechID
	p.mu.Unlock()

	transcripts, err := p.speechRepo.FetchNew(ctx, afterID)
	if err != nil {
		p.logger.Error("speech poller: failed to fetch", "error", err)
		return
	}

	if len(transcripts) == 0 {
		return
	}

	p.logger.Debug("speech poller: processing batch", "count", len(transcripts))

	for i := range transcripts {
		transcript := &transcripts[i]

		// Check upsell completion through speech
		p.coPilot.CheckUpsellCompletion(ctx, transcript)

		// Update last processed ID
		p.mu.Lock()
		if transcript.ID > p.lastProcessedSpeechID {
			p.lastProcessedSpeechID = transcript.ID
		}
		p.mu.Unlock()
	}
}

// pollCompletedTasks checks for video export tasks that Python workers have completed.
func (p *Poller) pollCompletedTasks(ctx context.Context) {
	ticker := time.NewTicker(p.pollIntervalTasks)
	defer ticker.Stop()

	p.logger.Info("tasks poller started", "interval", p.pollIntervalTasks)

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("tasks poller stopped")
			return
		case <-ticker.C:
			p.processCompletedTasks(ctx)
			p.processFailedTasks(ctx)
		}
	}
}

// processCompletedTasks handles tasks with status='completed'.
func (p *Poller) processCompletedTasks(ctx context.Context) {
	tasks, err := p.taskRepo.FetchCompleted(ctx)
	if err != nil {
		p.logger.Error("tasks poller: failed to fetch completed tasks", "error", err)
		return
	}

	for _, task := range tasks {
		p.logger.Info("tasks poller: processing completed task",
			"task_id", task.ID,
			"violation_id", task.ViolationID,
			"result_path", task.ResultPath,
		)

		// Update violation with the proof video path
		if task.ViolationID != nil && task.ResultPath != nil {
			if err := p.violationRepo.UpdateProofVideo(ctx, *task.ViolationID, *task.ResultPath); err != nil {
				p.logger.Error("tasks poller: failed to update violation proof",
					"error", err,
					"violation_id", *task.ViolationID,
				)
				continue
			}
		}

		// Mark task as processed (acknowledged by backend)
		if err := p.taskRepo.MarkProcessed(ctx, task.ID); err != nil {
			p.logger.Error("tasks poller: failed to mark task processed",
				"error", err,
				"task_id", task.ID,
			)
			continue
		}

		// Broadcast task completion to operator UI
		violationID := int64(0)
		videoPath := ""
		if task.ViolationID != nil {
			violationID = *task.ViolationID
		}
		if task.ResultPath != nil {
			videoPath = *task.ResultPath
		}

		p.hub.BroadcastTaskStatus(model.TaskStatusUpdate{
			TaskID:      task.ID,
			ViolationID: violationID,
			Status:      "completed",
			VideoPath:   videoPath,
		})
	}
}

// processFailedTasks handles tasks with status='failed'.
func (p *Poller) processFailedTasks(ctx context.Context) {
	tasks, err := p.taskRepo.FetchFailed(ctx)
	if err != nil {
		p.logger.Error("tasks poller: failed to fetch failed tasks", "error", err)
		return
	}

	for _, task := range tasks {
		errMsg := ""
		if task.ErrorMessage != nil {
			errMsg = *task.ErrorMessage
		}

		p.logger.Warn("tasks poller: task failed",
			"task_id", task.ID,
			"violation_id", task.ViolationID,
			"error_message", errMsg,
		)

		// Mark as processed so we don't re-fetch it
		if err := p.taskRepo.MarkProcessed(ctx, task.ID); err != nil {
			p.logger.Error("tasks poller: failed to mark failed task processed",
				"error", err,
				"task_id", task.ID,
			)
			continue
		}

		// Broadcast failure to operator
		violationID := int64(0)
		if task.ViolationID != nil {
			violationID = *task.ViolationID
		}

		p.hub.BroadcastTaskStatus(model.TaskStatusUpdate{
			TaskID:      task.ID,
			ViolationID: violationID,
			Status:      "failed",
		})
	}
}

// resolvePosByCamera looks up the pos_id for a given camera_id.
func (p *Poller) resolvePosByCamera(ctx context.Context, cameraID string) string {
	camera, err := p.cameraRepo.GetByID(ctx, cameraID)
	if err != nil {
		p.logger.Error("cv_events poller: failed to resolve camera", "error", err, "camera_id", cameraID)
		return ""
	}
	if camera == nil {
		return ""
	}
	return camera.PosID
}
