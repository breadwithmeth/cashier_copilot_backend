package repository

import (
	"context"
	"fmt"

	"cashier_copilot_backend/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TaskRepo handles the PostgreSQL-based task queue for video export operations.
type TaskRepo struct {
	pool *pgxpool.Pool
}

// NewTaskRepo creates a new TaskRepo.
func NewTaskRepo(pool *pgxpool.Pool) *TaskRepo {
	return &TaskRepo{pool: pool}
}

// Insert creates a new task in the queue with status 'pending'.
func (r *TaskRepo) Insert(ctx context.Context, task *model.Task) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx,
		`INSERT INTO tasks (task_type, camera_id, violation_id, payload, status)
		 VALUES ($1, $2, $3, $4, 'pending')
		 RETURNING id`,
		task.TaskType, task.CameraID, task.ViolationID, task.Payload,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert task: %w", err)
	}
	return id, nil
}

// FetchCompleted retrieves tasks that have been completed by the Python worker
// but not yet processed (acknowledged) by the Go backend.
func (r *TaskRepo) FetchCompleted(ctx context.Context) ([]model.Task, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, task_type, camera_id, violation_id, payload, status,
		        result_path, error_message, created_at, updated_at, processed_at
		 FROM tasks
		 WHERE status = 'completed' AND processed_at IS NULL
		 ORDER BY updated_at ASC
		 LIMIT 50`,
	)
	if err != nil {
		return nil, fmt.Errorf("fetch completed tasks: %w", err)
	}
	defer rows.Close()

	var tasks []model.Task
	for rows.Next() {
		var t model.Task
		if err := rows.Scan(
			&t.ID, &t.TaskType, &t.CameraID, &t.ViolationID, &t.Payload, &t.Status,
			&t.ResultPath, &t.ErrorMessage, &t.CreatedAt, &t.UpdatedAt, &t.ProcessedAt,
		); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

// FetchFailed retrieves tasks that failed, not yet acknowledged by the backend.
func (r *TaskRepo) FetchFailed(ctx context.Context) ([]model.Task, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, task_type, camera_id, violation_id, payload, status,
		        result_path, error_message, created_at, updated_at, processed_at
		 FROM tasks
		 WHERE status = 'failed' AND processed_at IS NULL
		 ORDER BY updated_at ASC
		 LIMIT 50`,
	)
	if err != nil {
		return nil, fmt.Errorf("fetch failed tasks: %w", err)
	}
	defer rows.Close()

	var tasks []model.Task
	for rows.Next() {
		var t model.Task
		if err := rows.Scan(
			&t.ID, &t.TaskType, &t.CameraID, &t.ViolationID, &t.Payload, &t.Status,
			&t.ResultPath, &t.ErrorMessage, &t.CreatedAt, &t.UpdatedAt, &t.ProcessedAt,
		); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

// MarkProcessed sets the processed_at timestamp to acknowledge a completed/failed task.
func (r *TaskRepo) MarkProcessed(ctx context.Context, taskID int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE tasks SET processed_at = CURRENT_TIMESTAMP WHERE id = $1`,
		taskID,
	)
	if err != nil {
		return fmt.Errorf("mark task processed: %w", err)
	}
	return nil
}
