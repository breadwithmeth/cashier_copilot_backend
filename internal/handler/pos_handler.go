package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"cashier_copilot_backend/internal/model"
	"cashier_copilot_backend/internal/repository"
	"cashier_copilot_backend/internal/service"
)

// PosHandler handles HTTP requests related to POS terminal events.
type PosHandler struct {
	posRepo    *repository.PosEventRepo
	fsm        *service.FSMManager
	ruleEngine *service.RuleEngine
	coPilot    *service.CoPilot
	logger     *slog.Logger
}

// NewPosHandler creates a new PosHandler.
func NewPosHandler(
	posRepo *repository.PosEventRepo,
	fsm *service.FSMManager,
	ruleEngine *service.RuleEngine,
	coPilot *service.CoPilot,
	logger *slog.Logger,
) *PosHandler {
	return &PosHandler{
		posRepo:    posRepo,
		fsm:        fsm,
		ruleEngine: ruleEngine,
		coPilot:    coPilot,
		logger:     logger,
	}
}

// HandlePosEvent processes incoming POS webhooks from 1C terminals.
// POST /api/v1/pos/event
func (h *PosHandler) HandlePosEvent(w http.ResponseWriter, r *http.Request) {
	var req model.PosEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("pos_handler: invalid request body", "error", err)
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{
			Error:   "invalid request body",
			Details: err.Error(),
		})
		return
	}

	// Validate required fields
	if req.PosID == "" || req.EventType == "" || req.TimestampMs == 0 {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{
			Error: "missing required fields: pos_id, event_type, timestamp_ms",
		})
		return
	}

	// Default details to empty JSON object if not provided
	if req.Details == nil {
		req.Details = json.RawMessage(`{}`)
	}

	// Create POS event model
	posEvent := &model.PosEvent{
		PosID:       req.PosID,
		EventType:   req.EventType,
		TimestampMs: req.TimestampMs,
		ReceiptID:   req.ReceiptID,
		Details:     req.Details,
	}

	// Insert into database
	id, err := h.posRepo.Insert(r.Context(), posEvent)
	if err != nil {
		h.logger.Error("pos_handler: failed to insert event", "error", err)
		writeJSON(w, http.StatusInternalServerError, model.ErrorResponse{
			Error: "failed to store event",
		})
		return
	}
	posEvent.ID = id

	h.logger.Info("pos_handler: event received and stored",
		"id", id,
		"pos_id", req.PosID,
		"event_type", req.EventType,
		"receipt_id", req.ReceiptID,
		"timestamp_ms", req.TimestampMs,
	)

	// Update FSM state
	oldState, newState, fsmErr := h.fsm.TransitionPosEvent(req.PosID, posEvent)
	if fsmErr != nil {
		h.logger.Warn("pos_handler: FSM transition warning", "error", fsmErr)
	}

	// Run rule engine checks based on event type
	switch req.EventType {
	case "receipt_cancelled", "item_removed":
		runDetached(20*time.Second, func(ctx context.Context) {
			h.ruleEngine.CheckVoidWithoutReturn(ctx, posEvent)
		})

	case "loyalty_card_applied":
		runDetached(20*time.Second, func(ctx context.Context) {
			h.ruleEngine.CheckLoyaltyCardAbuse(ctx, posEvent)
		})

	case "item_scanned":
		// Check for age-restricted items
		var details model.PosEventDetails
		if err := json.Unmarshal(posEvent.Details, &details); err == nil && details.AgeRestricted {
			runDetached(30*time.Second, func(ctx context.Context) {
				h.ruleEngine.CheckAgeVerification(ctx, posEvent)
			})
		}

		// AI Co-Pilot: check for upsell opportunities
		runDetached(20*time.Second, func(ctx context.Context) {
			h.coPilot.HandleItemScanned(ctx, posEvent)
		})

	case "receipt_closed":
		// Clear active upsell tracking for this receipt
		h.coPilot.ClearReceipt(posEvent.ReceiptID)
	}

	// Check no_cashier_on_sale on specific state transitions
	if (newState == model.StateScanning || newState == model.StatePayment) && oldState != newState {
		runDetached(20*time.Second, func(ctx context.Context) {
			h.ruleEngine.CheckNoCashierOnSale(ctx, req.PosID, req.TimestampMs)
		})
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":        id,
		"status":    "accepted",
		"fsm_state": newState,
	})
}

// writeJSON is a helper to write JSON responses.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func runDetached(timeout time.Duration, fn func(context.Context)) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		fn(ctx)
	}()
}
