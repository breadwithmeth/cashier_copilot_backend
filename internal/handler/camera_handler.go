package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"cashier_copilot_backend/internal/model"
	"cashier_copilot_backend/internal/repository"

	"github.com/go-chi/chi/v5"
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
	if cam.SourceStreamURL == "" {
		cam.SourceStreamURL = streamURLFromROI(cam.ROIConfig)
	}
	if cam.AnalyticsStreamType == "" {
		cam.AnalyticsStreamType = inferStreamType(cam.AnalyticsStreamURL)
	}
	if cam.AnalyticsStreamStatus == "" {
		cam.AnalyticsStreamStatus = "unknown"
		if cam.AnalyticsStreamURL != "" {
			cam.AnalyticsStreamStatus = "online"
		}
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

// HandleGetCameraStreams returns stream metadata for a camera.
// GET /api/v1/cameras/{id}/streams
func (h *CameraHandler) HandleGetCameraStreams(w http.ResponseWriter, r *http.Request) {
	cameraID := chi.URLParam(r, "id")
	if cameraID == "" {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "missing camera id"})
		return
	}

	camera, err := h.cameraRepo.GetByID(r.Context(), cameraID)
	if err != nil {
		h.logger.Error("camera_handler: failed to get camera streams", "error", err, "camera_id", cameraID)
		writeJSON(w, http.StatusInternalServerError, model.ErrorResponse{Error: "failed to retrieve camera streams"})
		return
	}
	if camera == nil {
		writeJSON(w, http.StatusNotFound, model.ErrorResponse{Error: "camera not found"})
		return
	}

	user := CurrentUser(r)
	includeSource := user != nil && user.Role == model.RoleAdmin
	writeJSON(w, http.StatusOK, cameraStreamInfo(camera, includeSource))
}

// HandleUpdateCameraStreams updates stream metadata for a camera.
// PATCH /api/v1/cameras/{id}/streams
// POST /api/v1/analytics/cameras/{id}/stream
func (h *CameraHandler) HandleUpdateCameraStreams(w http.ResponseWriter, r *http.Request) {
	cameraID := chi.URLParam(r, "id")
	if cameraID == "" {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "missing camera id"})
		return
	}

	var req model.CameraStreamUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{
			Error:   "invalid request body",
			Details: err.Error(),
		})
		return
	}

	if req.AnalyticsStreamType == "" {
		req.AnalyticsStreamType = inferStreamType(req.AnalyticsStreamURL)
	}
	if req.AnalyticsStreamStatus == "" && req.AnalyticsStreamURL != "" {
		req.AnalyticsStreamStatus = "online"
	}

	camera, err := h.cameraRepo.UpdateStreams(r.Context(), cameraID, req)
	if err != nil {
		h.logger.Error("camera_handler: failed to update camera streams", "error", err, "camera_id", cameraID)
		writeJSON(w, http.StatusInternalServerError, model.ErrorResponse{Error: "failed to update camera streams"})
		return
	}
	if camera == nil {
		writeJSON(w, http.StatusNotFound, model.ErrorResponse{Error: "camera not found"})
		return
	}

	user := CurrentUser(r)
	includeSource := user != nil && user.Role == model.RoleAdmin
	writeJSON(w, http.StatusOK, cameraStreamInfo(camera, includeSource))
}

func cameraStreamInfo(camera *model.Camera, includeSource bool) model.CameraStreamInfo {
	info := model.CameraStreamInfo{
		CameraID:                 camera.ID,
		PosID:                    camera.PosID,
		AnalyticsStreamURL:       camera.AnalyticsStreamURL,
		AnalyticsStreamType:      camera.AnalyticsStreamType,
		AnalyticsStreamStatus:    camera.AnalyticsStreamStatus,
		AnalyticsStreamUpdatedAt: camera.AnalyticsStreamUpdatedAt,
		ROIConfig:                camera.ROIConfig,
		OverlayEnabled:           camera.AnalyticsStreamURL != "",
	}
	if includeSource {
		info.SourceStreamURL = camera.SourceStreamURL
	}
	return info
}

func inferStreamType(streamURL string) string {
	u := strings.ToLower(streamURL)
	switch {
	case u == "":
		return ""
	case strings.Contains(u, ".m3u8"):
		return "hls"
	case strings.HasPrefix(u, "webrtc://") || strings.Contains(u, "/whep") || strings.Contains(u, "/whip"):
		return "webrtc"
	case strings.Contains(u, "mjpeg") || strings.Contains(u, "multipart"):
		return "mjpeg"
	case strings.HasPrefix(u, "rtsp://"):
		return "rtsp"
	case strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://"):
		return "http"
	default:
		return "unknown"
	}
}

func streamURLFromROI(raw json.RawMessage) string {
	var data map[string]interface{}
	if len(raw) == 0 || json.Unmarshal(raw, &data) != nil {
		return ""
	}
	for _, key := range []string{"source_stream_url", "rtsp_url", "stream_url"} {
		if val, ok := data[key].(string); ok {
			return val
		}
	}
	return ""
}
