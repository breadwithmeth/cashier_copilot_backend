package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"cashier_copilot_backend/internal/model"
	"cashier_copilot_backend/internal/repository"
)

// CameraHandler handles HTTP requests for camera configuration.
type CameraHandler struct {
	cameraRepo *repository.CameraRepo
	logger     *slog.Logger
}

// NewCameraHandler creates a new CameraHandler.
func NewCameraHandler(cameraRepo *repository.CameraRepo, logger *slog.Logger) *CameraHandler {
	return &CameraHandler{
		cameraRepo: cameraRepo,
		logger:     logger,
	}
}

// HandleCreateCamera adds a new camera configuration.
// POST /api/v1/cameras
func (h *CameraHandler) HandleCreateCamera(w http.ResponseWriter, r *http.Request) {
	var cam model.Camera
	if err := json.NewDecoder(r.Body).Decode(&cam); err != nil {
		h.logger.Warn("camera_handler: invalid request body", "error", err)
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{
			Error:   "invalid request body",
			Details: err.Error(),
		})
		return
	}

	// Validate required fields
	if cam.ID == "" || cam.IPAddress == "" || cam.PosID == "" {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{
			Error: "missing required fields: id, ip_address, pos_id",
		})
		return
	}

	// Set defaults
	if cam.Status == "" {
		cam.Status = "active"
	}
	if cam.ROIConfig == nil {
		cam.ROIConfig = json.RawMessage(`{}`)
	}

	if err := h.cameraRepo.Insert(r.Context(), &cam); err != nil {
		h.logger.Error("camera_handler: failed to insert camera", "error", err)
		writeJSON(w, http.StatusInternalServerError, model.ErrorResponse{
			Error: "failed to create camera",
		})
		return
	}

	h.logger.Info("camera_handler: camera created",
		"id", cam.ID,
		"pos_id", cam.PosID,
		"ip_address", cam.IPAddress,
	)

	writeJSON(w, http.StatusCreated, cam)
}

// HandleListCameras returns all configured cameras.
// GET /api/v1/cameras
func (h *CameraHandler) HandleListCameras(w http.ResponseWriter, r *http.Request) {
	cameras, err := h.cameraRepo.List(r.Context())
	if err != nil {
		h.logger.Error("camera_handler: failed to list cameras", "error", err)
		writeJSON(w, http.StatusInternalServerError, model.ErrorResponse{
			Error: "failed to retrieve cameras",
		})
		return
	}

	// Ensure non-nil array for JSON serialization
	if cameras == nil {
		cameras = []model.Camera{}
	}

	writeJSON(w, http.StatusOK, cameras)
}
