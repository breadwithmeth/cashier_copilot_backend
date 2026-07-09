package repository

import (
	"context"
	"fmt"

	"cashier_copilot_backend/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PosEventRepo handles persistence of POS terminal events.
type PosEventRepo struct {
	pool *pgxpool.Pool
}

// NewPosEventRepo creates a new PosEventRepo.
func NewPosEventRepo(pool *pgxpool.Pool) *PosEventRepo {
	return &PosEventRepo{pool: pool}
}

// Insert saves a new POS event to the database and returns the generated ID.
func (r *PosEventRepo) Insert(ctx context.Context, event *model.PosEvent) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx,
		`INSERT INTO pos_events (pos_id, event_type, timestamp_ms, receipt_id, details_jsonb)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id`,
		event.PosID, event.EventType, event.TimestampMs, event.ReceiptID, event.Details,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert pos_event: %w", err)
	}
	return id, nil
}

// FindInWindow searches for POS events of a given type within a time window [fromMs, toMs].
// Used by the Rule Engine for temporal correlation with CV events.
func (r *PosEventRepo) FindInWindow(ctx context.Context, posID, eventType string, fromMs, toMs int64) ([]model.PosEvent, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, pos_id, event_type, timestamp_ms, receipt_id, details_jsonb
		 FROM pos_events
		 WHERE pos_id = $1 AND event_type = $2 AND timestamp_ms BETWEEN $3 AND $4
		 ORDER BY timestamp_ms ASC`,
		posID, eventType, fromMs, toMs,
	)
	if err != nil {
		return nil, fmt.Errorf("find pos_events in window: %w", err)
	}
	defer rows.Close()

	var events []model.PosEvent
	for rows.Next() {
		var e model.PosEvent
		if err := rows.Scan(&e.ID, &e.PosID, &e.EventType, &e.TimestampMs, &e.ReceiptID, &e.Details); err != nil {
			return nil, fmt.Errorf("scan pos_event: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// FindByReceiptID retrieves all events for a specific receipt.
func (r *PosEventRepo) FindByReceiptID(ctx context.Context, receiptID string) ([]model.PosEvent, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, pos_id, event_type, timestamp_ms, receipt_id, details_jsonb
		 FROM pos_events
		 WHERE receipt_id = $1
		 ORDER BY timestamp_ms ASC`,
		receiptID,
	)
	if err != nil {
		return nil, fmt.Errorf("find pos_events by receipt: %w", err)
	}
	defer rows.Close()

	var events []model.PosEvent
	for rows.Next() {
		var e model.PosEvent
		if err := rows.Scan(&e.ID, &e.PosID, &e.EventType, &e.TimestampMs, &e.ReceiptID, &e.Details); err != nil {
			return nil, fmt.Errorf("scan pos_event: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}
