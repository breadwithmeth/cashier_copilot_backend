package repository

import (
	"context"
	"fmt"

	"cashier_copilot_backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CameraRepo handles persistence of IP camera configurations.
type CameraRepo struct {
	pool *pgxpool.Pool
}

// NewCameraRepo creates a new CameraRepo.
func NewCameraRepo(pool *pgxpool.Pool) *CameraRepo {
	return &CameraRepo{pool: pool}
}

// Insert saves a new camera configuration.
func (r *CameraRepo) Insert(ctx context.Context, cam *model.Camera) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO cameras (
			id, ip_address, username, password, pos_id, status, roi_config,
			source_stream_url, analytics_stream_url, analytics_stream_type, analytics_stream_status
		 )
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		cam.ID, cam.IPAddress, cam.Username, cam.Password, cam.PosID, cam.Status, cam.ROIConfig,
		cam.SourceStreamURL, cam.AnalyticsStreamURL, cam.AnalyticsStreamType, cam.AnalyticsStreamStatus,
	)
	if err != nil {
		return fmt.Errorf("insert camera: %w", err)
	}
	return nil
}

// List retrieves all cameras.
func (r *CameraRepo) List(ctx context.Context) ([]model.Camera, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, ip_address, username, password, pos_id, status, roi_config,
		        source_stream_url, analytics_stream_url, analytics_stream_type,
		        analytics_stream_status, analytics_stream_updated_at, created_at
		 FROM cameras
		 ORDER BY created_at ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list cameras: %w", err)
	}
	defer rows.Close()

	var cameras []model.Camera
	for rows.Next() {
		var c model.Camera
		if err := scanCamera(rows.Scan, &c); err != nil {
			return nil, fmt.Errorf("scan camera: %w", err)
		}
		cameras = append(cameras, c)
	}
	return cameras, rows.Err()
}

// GetByID finds a camera by its identifier.
// Returns nil, nil if no camera is found.
func (r *CameraRepo) GetByID(ctx context.Context, cameraID string) (*model.Camera, error) {
	var c model.Camera
	err := r.pool.QueryRow(ctx,
		`SELECT id, ip_address, username, password, pos_id, status, roi_config,
		        source_stream_url, analytics_stream_url, analytics_stream_type,
		        analytics_stream_status, analytics_stream_updated_at, created_at
		 FROM cameras
		 WHERE id = $1
		 LIMIT 1`,
		cameraID,
	).Scan(
		&c.ID, &c.IPAddress, &c.Username, &c.Password, &c.PosID, &c.Status, &c.ROIConfig,
		&c.SourceStreamURL, &c.AnalyticsStreamURL, &c.AnalyticsStreamType,
		&c.AnalyticsStreamStatus, &c.AnalyticsStreamUpdatedAt, &c.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get camera by id: %w", err)
	}
	return &c, nil
}

// GetByPosID finds the camera associated with a given POS terminal.
// Returns pgx.ErrNoRows if no camera is found.
func (r *CameraRepo) GetByPosID(ctx context.Context, posID string) (*model.Camera, error) {
	var c model.Camera
	err := r.pool.QueryRow(ctx,
		`SELECT id, ip_address, username, password, pos_id, status, roi_config,
		        source_stream_url, analytics_stream_url, analytics_stream_type,
		        analytics_stream_status, analytics_stream_updated_at, created_at
		 FROM cameras
		 WHERE pos_id = $1
		 LIMIT 1`,
		posID,
	).Scan(
		&c.ID, &c.IPAddress, &c.Username, &c.Password, &c.PosID, &c.Status, &c.ROIConfig,
		&c.SourceStreamURL, &c.AnalyticsStreamURL, &c.AnalyticsStreamType,
		&c.AnalyticsStreamStatus, &c.AnalyticsStreamUpdatedAt, &c.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get camera by pos_id: %w", err)
	}
	return &c, nil
}

// UpdateStreams updates source and analytics stream metadata for a camera.
func (r *CameraRepo) UpdateStreams(ctx context.Context, cameraID string, req model.CameraStreamUpdateRequest) (*model.Camera, error) {
	var c model.Camera
	err := r.pool.QueryRow(ctx,
		`UPDATE cameras
		 SET source_stream_url = COALESCE(NULLIF($2, ''), source_stream_url),
		     analytics_stream_url = COALESCE(NULLIF($3, ''), analytics_stream_url),
		     analytics_stream_type = COALESCE(NULLIF($4, ''), analytics_stream_type),
		     analytics_stream_status = COALESCE(NULLIF($5, ''), analytics_stream_status),
		     analytics_stream_updated_at = CASE
		       WHEN NULLIF($3, '') IS NOT NULL OR NULLIF($5, '') IS NOT NULL THEN CURRENT_TIMESTAMP
		       ELSE analytics_stream_updated_at
		     END
		 WHERE id = $1
		 RETURNING id, ip_address, username, password, pos_id, status, roi_config,
		           source_stream_url, analytics_stream_url, analytics_stream_type,
		           analytics_stream_status, analytics_stream_updated_at, created_at`,
		cameraID, req.SourceStreamURL, req.AnalyticsStreamURL, req.AnalyticsStreamType, req.AnalyticsStreamStatus,
	).Scan(
		&c.ID, &c.IPAddress, &c.Username, &c.Password, &c.PosID, &c.Status, &c.ROIConfig,
		&c.SourceStreamURL, &c.AnalyticsStreamURL, &c.AnalyticsStreamType,
		&c.AnalyticsStreamStatus, &c.AnalyticsStreamUpdatedAt, &c.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("update camera streams: %w", err)
	}
	return &c, nil
}

type cameraScanner func(dest ...interface{}) error

func scanCamera(scan cameraScanner, c *model.Camera) error {
	return scan(
		&c.ID, &c.IPAddress, &c.Username, &c.Password, &c.PosID, &c.Status, &c.ROIConfig,
		&c.SourceStreamURL, &c.AnalyticsStreamURL, &c.AnalyticsStreamType,
		&c.AnalyticsStreamStatus, &c.AnalyticsStreamUpdatedAt, &c.CreatedAt,
	)
}
