package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"cashier_copilot_backend/internal/model"
	"cashier_copilot_backend/internal/repository"
)

// ViolationHandler handles HTTP requests for the violations journal.
type ViolationHandler struct {
	violationRepo *repository.ViolationRepo
	logger        *slog.Logger
}

// NewViolationHandler creates a new ViolationHandler.
func NewViolationHandler(violationRepo *repository.ViolationRepo, logger *slog.Logger) *ViolationHandler {
	return &ViolationHandler{
		violationRepo: violationRepo,
		logger:        logger,
	}
}

// HandleListViolations returns a paginated list of violations with optional filters.
// GET /api/v1/violations?pos_id=xxx&type=xxx&status=xxx&from_ts=xxx&to_ts=xxx&limit=xxx&offset=xxx
func (h *ViolationHandler) HandleListViolations(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	// Parse filters
	filters := model.ViolationFilters{
		PosID:         query.Get("pos_id"),
		ViolationType: query.Get("type"),
		Status:        query.Get("status"),
	}

	if fromStr := query.Get("from_ts"); fromStr != "" {
		if ts, err := strconv.ParseInt(fromStr, 10, 64); err == nil {
			filters.FromTs = &ts
		}
	}

	if toStr := query.Get("to_ts"); toStr != "" {
		if ts, err := strconv.ParseInt(toStr, 10, 64); err == nil {
			filters.ToTs = &ts
		}
	}

	// Parse pagination
	limit := 50
	if limitStr := query.Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 200 {
			limit = l
		}
	}

	offset := 0
	if offsetStr := query.Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Query database
	violations, total, err := h.violationRepo.List(r.Context(), filters, limit, offset)
	if err != nil {
		h.logger.Error("violation_handler: failed to list violations", "error", err)
		writeJSON(w, http.StatusInternalServerError, model.ErrorResponse{
			Error: "failed to retrieve violations",
		})
		return
	}

	// Ensure non-nil array for JSON serialization
	if violations == nil {
		violations = []model.Violation{}
	}

	writeJSON(w, http.StatusOK, model.ViolationListResponse{
		Data:   violations,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}
