package repository

import (
	"context"
	"fmt"

	"cashier_copilot_backend/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CvEventRepo handles reading of computer vision detection events.
type CvEventRepo struct {
	pool *pgxpool.Pool
}

// NewCvEventRepo creates a new CvEventRepo.
func NewCvEventRepo(pool *pgxpool.Pool) *CvEventRepo {
	return &CvEventRepo{pool: pool}
}

// FetchNew retrieves CV events with ID greater than afterID, ordered by ID ascending.
// Used by the background poller to process new detections written by the Python YOLO worker.
func (r *CvEventRepo) FetchNew(ctx context.Context, afterID int64) ([]model.CvEvent, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, camera_id, event_type, timestamp_ms, confidence,
		        model_name, weights_version, inference_time_ms, bbox_jsonb, snapshot_path
		 FROM cv_events
		 WHERE id > $1
		 ORDER BY id ASC
		 LIMIT 100`,
		afterID,
	)
	if err != nil {
		return nil, fmt.Errorf("fetch new cv_events: %w", err)
	}
	defer rows.Close()

	var events []model.CvEvent
	for rows.Next() {
		var e model.CvEvent
		if err := rows.Scan(
			&e.ID, &e.CameraID, &e.EventType, &e.TimestampMs, &e.Confidence,
			&e.ModelName, &e.WeightsVersion, &e.InferenceTimeMs, &e.BboxJsonb, &e.SnapshotPath,
		); err != nil {
			return nil, fmt.Errorf("scan cv_event: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// FindInWindow searches for CV events matching a given type and camera within a time window [fromMs, toMs].
func (r *CvEventRepo) FindInWindow(ctx context.Context, cameraID, eventType string, fromMs, toMs int64) ([]model.CvEvent, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, camera_id, event_type, timestamp_ms, confidence,
		        model_name, weights_version, inference_time_ms, bbox_jsonb, snapshot_path
		 FROM cv_events
		 WHERE camera_id = $1 AND event_type = $2 AND timestamp_ms BETWEEN $3 AND $4
		 ORDER BY timestamp_ms ASC`,
		cameraID, eventType, fromMs, toMs,
	)
	if err != nil {
		return nil, fmt.Errorf("find cv_events in window: %w", err)
	}
	defer rows.Close()

	var events []model.CvEvent
	for rows.Next() {
		var e model.CvEvent
		if err := rows.Scan(
			&e.ID, &e.CameraID, &e.EventType, &e.TimestampMs, &e.Confidence,
			&e.ModelName, &e.WeightsVersion, &e.InferenceTimeMs, &e.BboxJsonb, &e.SnapshotPath,
		); err != nil {
			return nil, fmt.Errorf("scan cv_event: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// FindInWindowByCamera searches for CV events matching a given type across any camera within a time window.
// This is useful when we know the pos_id but need to check all cameras mapped to it.
func (r *CvEventRepo) FindInWindowByCamera(ctx context.Context, cameraID string, eventTypes []string, fromMs, toMs int64) ([]model.CvEvent, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, camera_id, event_type, timestamp_ms, confidence,
		        model_name, weights_version, inference_time_ms, bbox_jsonb, snapshot_path
		 FROM cv_events
		 WHERE camera_id = $1 AND event_type = ANY($2) AND timestamp_ms BETWEEN $3 AND $4
		 ORDER BY timestamp_ms ASC`,
		cameraID, eventTypes, fromMs, toMs,
	)
	if err != nil {
		return nil, fmt.Errorf("find cv_events by camera multi-type: %w", err)
	}
	defer rows.Close()

	var events []model.CvEvent
	for rows.Next() {
		var e model.CvEvent
		if err := rows.Scan(
			&e.ID, &e.CameraID, &e.EventType, &e.TimestampMs, &e.Confidence,
			&e.ModelName, &e.WeightsVersion, &e.InferenceTimeMs, &e.BboxJsonb, &e.SnapshotPath,
		); err != nil {
			return nil, fmt.Errorf("scan cv_event: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// FindLatestByType returns the most recent CV event of a given type for a camera.
// Used by the no_cashier_on_sale rule to check cashier presence.
func (r *CvEventRepo) FindLatestByType(ctx context.Context, cameraID, eventType string) (*model.CvEvent, error) {
	var e model.CvEvent
	err := r.pool.QueryRow(ctx,
		`SELECT id, camera_id, event_type, timestamp_ms, confidence,
		        model_name, weights_version, inference_time_ms, bbox_jsonb, snapshot_path
		 FROM cv_events
		 WHERE camera_id = $1 AND event_type = $2
		 ORDER BY timestamp_ms DESC
		 LIMIT 1`,
		cameraID, eventType,
	).Scan(
		&e.ID, &e.CameraID, &e.EventType, &e.TimestampMs, &e.Confidence,
		&e.ModelName, &e.WeightsVersion, &e.InferenceTimeMs, &e.BboxJsonb, &e.SnapshotPath,
	)
	if err != nil {
		return nil, err // pgx.ErrNoRows is a valid "not found" case
	}
	return &e, nil
}
