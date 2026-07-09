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
		`INSERT INTO cameras (id, ip_address, username, password, pos_id, status, roi_config)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		cam.ID, cam.IPAddress, cam.Username, cam.Password, cam.PosID, cam.Status, cam.ROIConfig,
	)
	if err != nil {
		return fmt.Errorf("insert camera: %w", err)
	}
	return nil
}

// List retrieves all cameras.
func (r *CameraRepo) List(ctx context.Context) ([]model.Camera, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, ip_address, username, password, pos_id, status, roi_config, created_at
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
		if err := rows.Scan(&c.ID, &c.IPAddress, &c.Username, &c.Password,
			&c.PosID, &c.Status, &c.ROIConfig, &c.CreatedAt); err != nil {
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
		`SELECT id, ip_address, username, password, pos_id, status, roi_config, created_at
		 FROM cameras
		 WHERE id = $1
		 LIMIT 1`,
		cameraID,
	).Scan(&c.ID, &c.IPAddress, &c.Username, &c.Password,
		&c.PosID, &c.Status, &c.ROIConfig, &c.CreatedAt)
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
		`SELECT id, ip_address, username, password, pos_id, status, roi_config, created_at
		 FROM cameras
		 WHERE pos_id = $1
		 LIMIT 1`,
		posID,
	).Scan(&c.ID, &c.IPAddress, &c.Username, &c.Password,
		&c.PosID, &c.Status, &c.ROIConfig, &c.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get camera by pos_id: %w", err)
	}
	return &c, nil
}
