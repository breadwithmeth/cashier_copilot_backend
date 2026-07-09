package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"cashier_copilot_backend/internal/model"
	"cashier_copilot_backend/internal/repository"
)

// RuleEngine evaluates business rules against incoming events to detect violations.
// Each rule operates by correlating events across time windows in PostgreSQL.
type RuleEngine struct {
	posRepo       *repository.PosEventRepo
	cvRepo        *repository.CvEventRepo
	speechRepo    *repository.SpeechRepo
	violationRepo *repository.ViolationRepo
	taskRepo      *repository.TaskRepo
	cameraRepo    *repository.CameraRepo
	fsm           *FSMManager
	hub           *Hub
	threshold     float64
	logger        *slog.Logger
}

// NewRuleEngine creates a new RuleEngine.
func NewRuleEngine(
	posRepo *repository.PosEventRepo,
	cvRepo *repository.CvEventRepo,
	speechRepo *repository.SpeechRepo,
	violationRepo *repository.ViolationRepo,
	taskRepo *repository.TaskRepo,
	cameraRepo *repository.CameraRepo,
	fsm *FSMManager,
	hub *Hub,
	threshold float64,
	logger *slog.Logger,
) *RuleEngine {
	return &RuleEngine{
		posRepo:       posRepo,
		cvRepo:        cvRepo,
		speechRepo:    speechRepo,
		violationRepo: violationRepo,
		taskRepo:      taskRepo,
		cameraRepo:    cameraRepo,
		fsm:           fsm,
		hub:           hub,
		threshold:     threshold,
		logger:        logger,
	}
}

// --- Rule 1: Unscanned Item (Пропуск сканирования) ---

// CheckUnscannedItem is triggered when a CV event of type "item_in_bag" is detected.
// It verifies that a corresponding "item_scanned" POS event exists within [-3000ms, +1500ms].
func (re *RuleEngine) CheckUnscannedItem(ctx context.Context, cvEvent *model.CvEvent, posID string) {
	re.logger.Info("rule_engine: checking unscanned_item",
		"cv_event_id", cvEvent.ID,
		"camera_id", cvEvent.CameraID,
		"timestamp_ms", cvEvent.TimestampMs,
		"confidence", cvEvent.Confidence,
	)

	// Precondition: confidence must be >= 0.80
	if cvEvent.Confidence < 0.80 {
		re.logger.Debug("rule_engine: unscanned_item — cv confidence below 0.80, skipping",
			"confidence", cvEvent.Confidence,
		)
		return
	}

	// Precondition: FSM state must be Scanning
	state := re.fsm.GetState(posID)
	if state != model.StateScanning {
		re.logger.Debug("rule_engine: unscanned_item — state is not Scanning, skipping",
			"state", state,
			"pos_id", posID,
		)
		return
	}

	// Check for item_scanned in POS events within [T_cv - 3000ms, T_cv + 1500ms]
	fromMs := cvEvent.TimestampMs - 3000
	toMs := cvEvent.TimestampMs + 1500

	posEvents, err := re.posRepo.FindInWindow(ctx, posID, "item_scanned", fromMs, toMs)
	if err != nil {
		re.logger.Error("rule_engine: unscanned_item — failed to query pos_events", "error", err)
		return
	}

	if len(posEvents) > 0 {
		re.logger.Debug("rule_engine: unscanned_item — scan event found, no violation",
			"pos_events_count", len(posEvents),
		)
		return
	}

	// No corresponding scan event found — generate violation
	// Confidence aggregate: P(item_in_bag) * (1 - P(item_scanned))
	// Since no scan found, P(item_scanned) = 0
	confidenceAggregate := cvEvent.Confidence

	re.createViolation(ctx, &model.Violation{
		PosID:               posID,
		ViolationType:       "unscanned_item",
		TimestampMs:         cvEvent.TimestampMs,
		CvEventID:           &cvEvent.ID,
		ConfidenceAggregate: confidenceAggregate,
	}, cvEvent.CameraID, cvEvent.TimestampMs)
}

// --- Rule 2: Void Without Return (Отмена без возврата товара) ---

// CheckVoidWithoutReturn is triggered by receipt_cancelled or item_removed POS events.
// It checks that a physical return movement was observed on video within [-5s, +10s].
func (re *RuleEngine) CheckVoidWithoutReturn(ctx context.Context, posEvent *model.PosEvent) {
	re.logger.Info("rule_engine: checking void_without_return",
		"pos_event_id", posEvent.ID,
		"pos_id", posEvent.PosID,
		"event_type", posEvent.EventType,
		"timestamp_ms", posEvent.TimestampMs,
	)

	// Resolve camera_id from pos_id
	camera, err := re.cameraRepo.GetByPosID(ctx, posEvent.PosID)
	if err != nil || camera == nil {
		re.logger.Warn("rule_engine: void_without_return — no camera for pos_id", "pos_id", posEvent.PosID)
		return
	}

	fromMs := posEvent.TimestampMs - 5000
	toMs := posEvent.TimestampMs + 10000

	// Look for a return movement (item_return or hand_to_scanner events indicating a physical return)
	returnEvents, err := re.cvRepo.FindInWindowByCamera(ctx, camera.ID,
		[]string{"item_return", "hand_to_scanner"}, fromMs, toMs)
	if err != nil {
		re.logger.Error("rule_engine: void_without_return — failed to query cv_events", "error", err)
		return
	}

	if len(returnEvents) > 0 {
		re.logger.Debug("rule_engine: void_without_return — physical return detected, no violation")
		return
	}

	// Check if customer is still present (if customer left without return, it's suspicious)
	customerEvents, err := re.cvRepo.FindInWindow(ctx, camera.ID, "customer_present", fromMs, toMs)
	if err != nil {
		re.logger.Error("rule_engine: void_without_return — failed to check customer presence", "error", err)
		return
	}

	// If customer is still present, there may be a legitimate interaction
	// We still flag it but with lower confidence
	confidence := 0.90
	if len(customerEvents) > 0 {
		confidence = 0.60
	}

	re.createViolation(ctx, &model.Violation{
		PosID:               posEvent.PosID,
		ViolationType:       "void_without_return",
		TimestampMs:         posEvent.TimestampMs,
		PosEventID:          &posEvent.ID,
		ConfidenceAggregate: confidence,
	}, camera.ID, posEvent.TimestampMs)
}

// --- Rule 3: Loyalty Card Abuse (Использование кассиром своего QR-кода) ---

// CheckLoyaltyCardAbuse is triggered when loyalty_card_applied is received.
// It checks for phone_scanned_by_cashier CV events within ±2 seconds.
func (re *RuleEngine) CheckLoyaltyCardAbuse(ctx context.Context, posEvent *model.PosEvent) {
	re.logger.Info("rule_engine: checking loyalty_card_abuse",
		"pos_event_id", posEvent.ID,
		"pos_id", posEvent.PosID,
		"timestamp_ms", posEvent.TimestampMs,
	)

	camera, err := re.cameraRepo.GetByPosID(ctx, posEvent.PosID)
	if err != nil || camera == nil {
		re.logger.Warn("rule_engine: loyalty_card_abuse — no camera for pos_id", "pos_id", posEvent.PosID)
		return
	}

	fromMs := posEvent.TimestampMs - 2000
	toMs := posEvent.TimestampMs + 2000

	// Look for phone_scanned_by_cashier — cashier scanning their own phone
	cashierPhoneEvents, err := re.cvRepo.FindInWindow(ctx, camera.ID, "phone_scanned_by_cashier", fromMs, toMs)
	if err != nil {
		re.logger.Error("rule_engine: loyalty_card_abuse — failed to query cv_events", "error", err)
		return
	}

	if len(cashierPhoneEvents) == 0 {
		re.logger.Debug("rule_engine: loyalty_card_abuse — no cashier phone scan detected, no violation")
		return
	}

	// Cashier phone detected in the scanning zone — violation
	confidence := cashierPhoneEvents[0].Confidence

	re.createViolation(ctx, &model.Violation{
		PosID:               posEvent.PosID,
		ViolationType:       "loyalty_card_abuse",
		TimestampMs:         posEvent.TimestampMs,
		PosEventID:          &posEvent.ID,
		CvEventID:           &cashierPhoneEvents[0].ID,
		ConfidenceAggregate: confidence,
	}, camera.ID, posEvent.TimestampMs)
}

// --- Rule 4: Age Verification Failed (Непроверенный возраст) ---

// CheckAgeVerification is triggered when an age-restricted item is scanned.
// It waits 15 seconds before verifying document presentation or verbal confirmation.
func (re *RuleEngine) CheckAgeVerification(ctx context.Context, posEvent *model.PosEvent) {
	re.logger.Info("rule_engine: scheduling age_verification check",
		"pos_event_id", posEvent.ID,
		"pos_id", posEvent.PosID,
		"timestamp_ms", posEvent.TimestampMs,
	)

	timer := time.NewTimer(15 * time.Second)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return
	case <-timer.C:
	}

	re.executeAgeVerificationCheck(ctx, posEvent)
}

// executeAgeVerificationCheck performs the actual check after the delay.
func (re *RuleEngine) executeAgeVerificationCheck(ctx context.Context, posEvent *model.PosEvent) {
	re.logger.Info("rule_engine: executing age_verification check",
		"pos_event_id", posEvent.ID,
		"pos_id", posEvent.PosID,
	)

	camera, err := re.cameraRepo.GetByPosID(ctx, posEvent.PosID)
	if err != nil || camera == nil {
		re.logger.Warn("rule_engine: age_verification — no camera for pos_id", "pos_id", posEvent.PosID)
		return
	}

	fromMs := posEvent.TimestampMs - 5000
	toMs := posEvent.TimestampMs + 15000

	// Check 1: Was a document (passport) presented on camera?
	docEvents, err := re.cvRepo.FindInWindow(ctx, camera.ID, "document_presented", fromMs, toMs)
	if err != nil {
		re.logger.Error("rule_engine: age_verification — failed to query document events", "error", err)
		return
	}

	if len(docEvents) > 0 {
		re.logger.Debug("rule_engine: age_verification — document presented, no violation")
		return
	}

	// Check 2: Was there verbal verification (speech with age-related keywords)?
	ageKeywords := []string{"паспорт", "документ", "возраст", "18", "восемнадцать", "лет", "рождения"}
	speechMatches, err := re.speechRepo.FindWithKeywords(ctx, posEvent.PosID, fromMs, toMs, ageKeywords)
	if err != nil {
		re.logger.Error("rule_engine: age_verification — failed to query speech transcripts", "error", err)
		return
	}

	if len(speechMatches) > 0 {
		re.logger.Debug("rule_engine: age_verification — verbal verification detected, no violation",
			"matched_transcripts", len(speechMatches),
		)
		return
	}

	// Neither document nor verbal verification found — create violation
	re.createViolation(ctx, &model.Violation{
		PosID:               posEvent.PosID,
		ViolationType:       "age_verification_failed",
		TimestampMs:         posEvent.TimestampMs,
		PosEventID:          &posEvent.ID,
		ConfidenceAggregate: 0.95, // high confidence — no verification was observed
	}, camera.ID, posEvent.TimestampMs)
}

// --- Rule 5: Drawer Opened Without Sale (Открытие ящика без чека) ---

// CheckDrawerWithoutSale is triggered when a CV event "hand_to_drawer" is detected.
// Violation if FSM state is Idle or CustomerDetected (no active receipt/payment).
func (re *RuleEngine) CheckDrawerWithoutSale(ctx context.Context, cvEvent *model.CvEvent, posID string) {
	re.logger.Info("rule_engine: checking drawer_opened_without_sale",
		"cv_event_id", cvEvent.ID,
		"camera_id", cvEvent.CameraID,
		"timestamp_ms", cvEvent.TimestampMs,
	)

	state := re.fsm.GetState(posID)
	if state != model.StateIdle && state != model.StateCustomerDetected {
		re.logger.Debug("rule_engine: drawer_opened_without_sale — state allows drawer access",
			"state", state,
		)
		return
	}

	re.createViolation(ctx, &model.Violation{
		PosID:               posID,
		ViolationType:       "drawer_opened_without_sale",
		TimestampMs:         cvEvent.TimestampMs,
		CvEventID:           &cvEvent.ID,
		ConfidenceAggregate: cvEvent.Confidence,
	}, cvEvent.CameraID, cvEvent.TimestampMs)
}

// --- Rule 6: No Cashier On Sale (Отсутствие кассира) ---

// CheckNoCashierOnSale is triggered when the FSM transitions to Scanning or Payment.
// It checks for the latest cashier presence event — if the cashier is absent, flag a violation.
func (re *RuleEngine) CheckNoCashierOnSale(ctx context.Context, posID string, triggerTimestampMs int64) {
	re.logger.Info("rule_engine: checking no_cashier_on_sale",
		"pos_id", posID,
		"trigger_timestamp_ms", triggerTimestampMs,
	)

	camera, err := re.cameraRepo.GetByPosID(ctx, posID)
	if err != nil || camera == nil {
		re.logger.Warn("rule_engine: no_cashier_on_sale — no camera for pos_id", "pos_id", posID)
		return
	}

	// Check for the latest "no_cashier" or "cashier_absent" event
	latestAbsent, err := re.cvRepo.FindLatestByType(ctx, camera.ID, "no_cashier")
	if err != nil {
		// No event found — assume cashier is present
		re.logger.Debug("rule_engine: no_cashier_on_sale — no absence event found, assuming present")
		return
	}

	// Check if the latest cashier presence event is more recent than the absence
	latestPresent, err := re.cvRepo.FindLatestByType(ctx, camera.ID, "cashier_present")
	if err == nil && latestPresent != nil && latestPresent.TimestampMs > latestAbsent.TimestampMs {
		re.logger.Debug("rule_engine: no_cashier_on_sale — cashier is present (more recent than absence)")
		return
	}

	// Cashier is absent during an active sale
	re.createViolation(ctx, &model.Violation{
		PosID:               posID,
		ViolationType:       "no_cashier_on_sale",
		TimestampMs:         triggerTimestampMs,
		CvEventID:           &latestAbsent.ID,
		ConfidenceAggregate: latestAbsent.Confidence,
	}, camera.ID, triggerTimestampMs)
}

// --- Shared violation creation logic ---

// createViolation inserts a violation, optionally creates a video export task,
// and broadcasts the alert via WebSocket.
func (re *RuleEngine) createViolation(ctx context.Context, violation *model.Violation, cameraID string, eventTimestampMs int64) {
	// Apply confidence threshold filter
	status := "new"
	if violation.ConfidenceAggregate < re.threshold {
		status = "auto_filtered"
		re.logger.Info("rule_engine: violation auto_filtered (below threshold)",
			"type", violation.ViolationType,
			"confidence", violation.ConfidenceAggregate,
			"threshold", re.threshold,
		)
	}
	violation.Status = status

	// Insert violation
	violationID, err := re.violationRepo.Insert(ctx, violation)
	if err != nil {
		re.logger.Error("rule_engine: failed to insert violation", "error", err, "type", violation.ViolationType)
		return
	}
	violation.ID = violationID

	re.logger.Info("rule_engine: violation created",
		"id", violationID,
		"type", violation.ViolationType,
		"pos_id", violation.PosID,
		"confidence", violation.ConfidenceAggregate,
		"status", status,
	)

	// Create video export task (clip: T_event ±10 seconds)
	if status == "new" {
		payload := model.VideoExportPayload{
			StartTimestampMs: eventTimestampMs - 10000,
			EndTimestampMs:   eventTimestampMs + 10000,
		}
		payloadJSON, _ := json.Marshal(payload)

		task := &model.Task{
			TaskType:    "video_export",
			CameraID:    cameraID,
			ViolationID: &violationID,
			Payload:     payloadJSON,
		}

		taskID, err := re.taskRepo.Insert(ctx, task)
		if err != nil {
			re.logger.Error("rule_engine: failed to create video export task", "error", err)
		} else {
			re.logger.Info("rule_engine: video export task created",
				"task_id", taskID,
				"camera_id", cameraID,
				"violation_id", violationID,
			)
		}

		// Broadcast violation alert to operators
		re.hub.BroadcastViolationAlert(model.ViolationAlert{
			Violation: *violation,
		})
	}
}
