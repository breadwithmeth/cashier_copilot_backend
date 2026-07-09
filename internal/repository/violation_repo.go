package repository

import (
	"context"
	"fmt"
	"strings"

	"cashier_copilot_backend/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ViolationRepo handles persistence of detected violations / incidents.
type ViolationRepo struct {
	pool *pgxpool.Pool
}

// NewViolationRepo creates a new ViolationRepo.
func NewViolationRepo(pool *pgxpool.Pool) *ViolationRepo {
	return &ViolationRepo{pool: pool}
}

// Insert saves a new violation record and returns the generated ID.
func (r *ViolationRepo) Insert(ctx context.Context, v *model.Violation) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx,
		`INSERT INTO violations
		 (pos_id, violation_type, timestamp_ms, proof_video_path, proof_image_path,
		  cv_event_id, pos_event_id, speech_transcript_id, confidence_aggregate, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 RETURNING id`,
		v.PosID, v.ViolationType, v.TimestampMs, v.ProofVideoPath, v.ProofImagePath,
		v.CvEventID, v.PosEventID, v.SpeechTranscriptID, v.ConfidenceAggregate, v.Status,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert violation: %w", err)
	}
	return id, nil
}

// List retrieves violations with pagination and optional filters.
func (r *ViolationRepo) List(ctx context.Context, filters model.ViolationFilters, limit, offset int) ([]model.Violation, int64, error) {
	// Build WHERE clause dynamically
	conditions := []string{"1=1"}
	args := []interface{}{}
	argIdx := 1

	if filters.PosID != "" {
		conditions = append(conditions, fmt.Sprintf("pos_id = $%d", argIdx))
		args = append(args, filters.PosID)
		argIdx++
	}
	if filters.ViolationType != "" {
		conditions = append(conditions, fmt.Sprintf("violation_type = $%d", argIdx))
		args = append(args, filters.ViolationType)
		argIdx++
	}
	if filters.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, filters.Status)
		argIdx++
	}
	if filters.FromTs != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp_ms >= $%d", argIdx))
		args = append(args, *filters.FromTs)
		argIdx++
	}
	if filters.ToTs != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp_ms <= $%d", argIdx))
		args = append(args, *filters.ToTs)
		argIdx++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count total matching rows
	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM violations WHERE %s", whereClause)
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count violations: %w", err)
	}

	// Fetch paginated results
	dataQuery := fmt.Sprintf(
		`SELECT id, pos_id, violation_type, timestamp_ms, proof_video_path, proof_image_path,
		        cv_event_id, pos_event_id, speech_transcript_id, confidence_aggregate, status
		 FROM violations
		 WHERE %s
		 ORDER BY timestamp_ms DESC
		 LIMIT $%d OFFSET $%d`,
		whereClause, argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list violations: %w", err)
	}
	defer rows.Close()

	var violations []model.Violation
	for rows.Next() {
		var v model.Violation
		if err := rows.Scan(
			&v.ID, &v.PosID, &v.ViolationType, &v.TimestampMs,
			&v.ProofVideoPath, &v.ProofImagePath,
			&v.CvEventID, &v.PosEventID, &v.SpeechTranscriptID,
			&v.ConfidenceAggregate, &v.Status,
		); err != nil {
			return nil, 0, fmt.Errorf("scan violation: %w", err)
		}
		violations = append(violations, v)
	}

	return violations, total, rows.Err()
}

// UpdateProofVideo sets the proof video path for a violation.
func (r *ViolationRepo) UpdateProofVideo(ctx context.Context, id int64, videoPath string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE violations SET proof_video_path = $1 WHERE id = $2`,
		videoPath, id,
	)
	if err != nil {
		return fmt.Errorf("update violation proof video: %w", err)
	}
	return nil
}
