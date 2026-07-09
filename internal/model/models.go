package model

import (
	"encoding/json"
	"time"
)

// --- Cashier State Machine ---

// CashierState represents the current state of a POS terminal.
type CashierState string

const (
	StateIdle             CashierState = "Idle"
	StateCustomerDetected CashierState = "CustomerDetected"
	StateReceiptOpened    CashierState = "ReceiptOpened"
	StateScanning         CashierState = "Scanning"
	StatePayment          CashierState = "Payment"
	StateReceiptClosed    CashierState = "ReceiptClosed"
)

// --- Camera ---

// Camera represents an IP camera configuration.
type Camera struct {
	ID                       string          `json:"id"`
	IPAddress                string          `json:"ip_address"`
	Username                 string          `json:"username"`
	Password                 string          `json:"password,omitempty"`
	PosID                    string          `json:"pos_id"`
	Status                   string          `json:"status"`
	ROIConfig                json.RawMessage `json:"roi_config"`
	SourceStreamURL          string          `json:"source_stream_url,omitempty"`
	AnalyticsStreamURL       string          `json:"analytics_stream_url,omitempty"`
	AnalyticsStreamType      string          `json:"analytics_stream_type,omitempty"`
	AnalyticsStreamStatus    string          `json:"analytics_stream_status,omitempty"`
	AnalyticsStreamUpdatedAt *time.Time      `json:"analytics_stream_updated_at,omitempty"`
	CreatedAt                time.Time       `json:"created_at"`
}

// CameraStreamUpdateRequest updates source and analytics stream metadata.
type CameraStreamUpdateRequest struct {
	SourceStreamURL       string `json:"source_stream_url,omitempty"`
	AnalyticsStreamURL    string `json:"analytics_stream_url,omitempty"`
	AnalyticsStreamType   string `json:"analytics_stream_type,omitempty"`
	AnalyticsStreamStatus string `json:"analytics_stream_status,omitempty"`
}

// CameraStreamInfo is the frontend-facing stream contract for a camera.
type CameraStreamInfo struct {
	CameraID                 string          `json:"camera_id"`
	PosID                    string          `json:"pos_id"`
	AnalyticsStreamURL       string          `json:"analytics_stream_url,omitempty"`
	AnalyticsStreamType      string          `json:"analytics_stream_type,omitempty"`
	AnalyticsStreamStatus    string          `json:"analytics_stream_status,omitempty"`
	AnalyticsStreamUpdatedAt *time.Time      `json:"analytics_stream_updated_at,omitempty"`
	SourceStreamURL          string          `json:"source_stream_url,omitempty"`
	ROIConfig                json.RawMessage `json:"roi_config"`
	OverlayEnabled           bool            `json:"overlay_enabled"`
}

// --- POS Events ---

// PosEvent represents a transactional event from 1C POS terminal.
type PosEvent struct {
	ID          int64           `json:"id"`
	PosID       string          `json:"pos_id"`
	EventType   string          `json:"event_type"`
	TimestampMs int64           `json:"timestamp_ms"`
	ReceiptID   string          `json:"receipt_id"`
	Details     json.RawMessage `json:"details"`
}

// PosEventDetails holds the parsed details of a POS event.
type PosEventDetails struct {
	SKU           string  `json:"sku,omitempty"`
	ItemName      string  `json:"item_name,omitempty"`
	Category      string  `json:"category,omitempty"`
	Price         float64 `json:"price,omitempty"`
	Quantity      int     `json:"quantity,omitempty"`
	AgeRestricted bool    `json:"age_restricted,omitempty"`
}

// --- CV Events ---

// CvEvent represents a computer vision detection event from the YOLO service.
type CvEvent struct {
	ID              int64           `json:"id"`
	CameraID        string          `json:"camera_id"`
	EventType       string          `json:"event_type"`
	TimestampMs     int64           `json:"timestamp_ms"`
	Confidence      float64         `json:"confidence"`
	ModelName       string          `json:"model_name"`
	WeightsVersion  string          `json:"weights_version"`
	InferenceTimeMs int             `json:"inference_time_ms"`
	BboxJsonb       json.RawMessage `json:"bbox_jsonb"`
	SnapshotPath    string          `json:"snapshot_path"`
}

// --- Speech Transcripts ---

// SpeechTranscript represents a transcribed audio segment from the STT service.
type SpeechTranscript struct {
	ID             int64   `json:"id"`
	PosID          string  `json:"pos_id"`
	Transcript     string  `json:"transcript"`
	TimestampMs    int64   `json:"timestamp_ms"`
	DurationMs     int     `json:"duration_ms"`
	Confidence     float64 `json:"confidence"`
	ModelName      string  `json:"model_name"`
	WeightsVersion string  `json:"weights_version"`
}

// --- Auth / Users ---

const (
	RoleAdmin    = "admin"
	RoleOperator = "operator"
	RoleCashier  = "cashier"
)

// User represents an authenticated backend user.
type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	PosID        *string   `json:"pos_id,omitempty"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
}

// AuthUser is safe to expose in API responses and request context.
type AuthUser struct {
	ID       int64   `json:"id"`
	Username string  `json:"username"`
	Role     string  `json:"role"`
	PosID    *string `json:"pos_id,omitempty"`
}

// LoginRequest is the body for POST /api/v1/auth/login.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse is returned after successful login.
type LoginResponse struct {
	AccessToken string   `json:"access_token"`
	TokenType   string   `json:"token_type"`
	ExpiresAt   int64    `json:"expires_at"`
	User        AuthUser `json:"user"`
}

// CreateUserRequest is used by admins to create users.
type CreateUserRequest struct {
	Username string  `json:"username"`
	Password string  `json:"password"`
	Role     string  `json:"role"`
	PosID    *string `json:"pos_id,omitempty"`
	IsActive *bool   `json:"is_active,omitempty"`
}

// --- Upsell Rules ---

// UpsellRule defines an AI Co-Pilot suggestion rule.
type UpsellRule struct {
	ID                 int      `json:"id"`
	TriggerCategory    string   `json:"trigger_category"`
	RequiredKeywords   []string `json:"required_keywords"`
	SuggestionText     string   `json:"suggestion_text"`
	SuggestionImageURL string   `json:"suggestion_image_url,omitempty"`
}

// --- Violations ---

// Violation represents a detected policy violation / incident.
type Violation struct {
	ID                  int64   `json:"id"`
	PosID               string  `json:"pos_id"`
	ViolationType       string  `json:"violation_type"`
	TimestampMs         int64   `json:"timestamp_ms"`
	ProofVideoPath      *string `json:"proof_video_path,omitempty"`
	ProofImagePath      *string `json:"proof_image_path,omitempty"`
	CvEventID           *int64  `json:"cv_event_id,omitempty"`
	PosEventID          *int64  `json:"pos_event_id,omitempty"`
	SpeechTranscriptID  *int64  `json:"speech_transcript_id,omitempty"`
	ConfidenceAggregate float64 `json:"confidence_aggregate"`
	Status              string  `json:"status"`
}

// --- Tasks (Video Export Queue) ---

// Task represents a video export task in the PostgreSQL-based task queue.
type Task struct {
	ID           int64           `json:"id"`
	TaskType     string          `json:"task_type"`
	CameraID     string          `json:"camera_id"`
	ViolationID  *int64          `json:"violation_id,omitempty"`
	Payload      json.RawMessage `json:"payload"`
	Status       string          `json:"status"`
	ResultPath   *string         `json:"result_path,omitempty"`
	ErrorMessage *string         `json:"error_message,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	ProcessedAt  *time.Time      `json:"processed_at,omitempty"`
}

// VideoExportPayload holds the parameters for a video clip export task.
type VideoExportPayload struct {
	StartTimestampMs int64 `json:"start_timestamp_ms"`
	EndTimestampMs   int64 `json:"end_timestamp_ms"`
}

// --- WebSocket Messages ---

// WSMessage is the envelope for all WebSocket messages sent to clients.
type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// ViolationAlert is the payload sent to the operator via WebSocket.
type ViolationAlert struct {
	Violation Violation `json:"violation"`
	PosEvent  *PosEvent `json:"pos_event,omitempty"`
	CvEvent   *CvEvent  `json:"cv_event,omitempty"`
}

// TaskStatusUpdate is sent when a video export task completes.
type TaskStatusUpdate struct {
	TaskID      int64  `json:"task_id"`
	ViolationID int64  `json:"violation_id"`
	Status      string `json:"status"`
	VideoPath   string `json:"video_path,omitempty"`
}

// UpsellCard is sent to the cashier terminal with a product suggestion.
type UpsellCard struct {
	PosID           string `json:"pos_id"`
	ReceiptID       string `json:"receipt_id"`
	TriggerItem     string `json:"trigger_item"`
	SuggestionText  string `json:"suggestion_text"`
	SuggestionImage string `json:"suggestion_image,omitempty"`
	Status          string `json:"status"` // "pending", "completed"
}

// UpsellStatusUpdate is sent when the cashier verbally completes an upsell.
type UpsellStatusUpdate struct {
	PosID     string `json:"pos_id"`
	ReceiptID string `json:"receipt_id"`
	Status    string `json:"status"` // "completed"
}

// --- API Request / Response types ---

// PosEventRequest is the inbound JSON body for POST /api/v1/pos/event.
type PosEventRequest struct {
	PosEventID  string          `json:"pos_event_id"`
	PosID       string          `json:"pos_id"`
	ReceiptID   string          `json:"receipt_id"`
	EventType   string          `json:"event_type"`
	TimestampMs int64           `json:"timestamp_ms"`
	Details     json.RawMessage `json:"details"`
}

// ViolationListResponse is the paginated response for GET /api/v1/violations.
type ViolationListResponse struct {
	Data   []Violation `json:"data"`
	Total  int64       `json:"total"`
	Limit  int         `json:"limit"`
	Offset int         `json:"offset"`
}

// ViolationFilters holds query parameters for filtering violations.
type ViolationFilters struct {
	PosID         string
	ViolationType string
	Status        string
	FromTs        *int64
	ToTs          *int64
}

// ErrorResponse is a standard API error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}
