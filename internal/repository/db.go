package repository

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool creates a new pgxpool connection pool with the given configuration.
func NewPool(ctx context.Context, databaseURL string, maxConns int32) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	config.MaxConns = maxConns

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connectivity
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	slog.Info("database connection pool established",
		"max_conns", maxConns,
	)

	return pool, nil
}

// RunMigrations creates all required database tables, indexes, and constraints.
// It is idempotent — safe to run on every startup.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	slog.Info("running database migrations...")

	ddl := `
	-- Таблица IP-камер Dahua
	CREATE TABLE IF NOT EXISTS cameras (
		id VARCHAR(50) PRIMARY KEY,
		ip_address VARCHAR(45) NOT NULL,
		username VARCHAR(100) NOT NULL,
		password VARCHAR(100) NOT NULL,
		pos_id VARCHAR(50) NOT NULL,
		status VARCHAR(20) NOT NULL DEFAULT 'inactive',
		roi_config JSONB NOT NULL DEFAULT '{}'::jsonb,
		source_stream_url TEXT NOT NULL DEFAULT '',
		analytics_stream_url TEXT NOT NULL DEFAULT '',
		analytics_stream_type VARCHAR(20) NOT NULL DEFAULT '',
		analytics_stream_status VARCHAR(20) NOT NULL DEFAULT 'unknown',
		analytics_stream_updated_at TIMESTAMPTZ,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);
	ALTER TABLE cameras ADD COLUMN IF NOT EXISTS source_stream_url TEXT NOT NULL DEFAULT '';
	ALTER TABLE cameras ADD COLUMN IF NOT EXISTS analytics_stream_url TEXT NOT NULL DEFAULT '';
	ALTER TABLE cameras ADD COLUMN IF NOT EXISTS analytics_stream_type VARCHAR(20) NOT NULL DEFAULT '';
	ALTER TABLE cameras ADD COLUMN IF NOT EXISTS analytics_stream_status VARCHAR(20) NOT NULL DEFAULT 'unknown';
	ALTER TABLE cameras ADD COLUMN IF NOT EXISTS analytics_stream_updated_at TIMESTAMPTZ;

	-- Таблица кассовых событий от 1С (POS)
	CREATE TABLE IF NOT EXISTS pos_events (
		id BIGSERIAL PRIMARY KEY,
		pos_id VARCHAR(50) NOT NULL,
		event_type VARCHAR(50) NOT NULL,
		timestamp_ms BIGINT NOT NULL,
		receipt_id VARCHAR(100) NOT NULL,
		details_jsonb JSONB NOT NULL DEFAULT '{}'::jsonb
	);
	CREATE INDEX IF NOT EXISTS idx_pos_events_time_pos ON pos_events(timestamp_ms, pos_id);

	-- Таблица событий видеоаналитики от Python/YOLOv11
	CREATE TABLE IF NOT EXISTS cv_events (
		id BIGSERIAL PRIMARY KEY,
		camera_id VARCHAR(50) NOT NULL,
		event_type VARCHAR(50) NOT NULL,
		timestamp_ms BIGINT NOT NULL,
		confidence DOUBLE PRECISION NOT NULL,
		model_name VARCHAR(100) NOT NULL,
		weights_version VARCHAR(50) NOT NULL,
		inference_time_ms INTEGER NOT NULL,
		bbox_jsonb JSONB NOT NULL,
		snapshot_path VARCHAR(255) NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_cv_events_time_cam ON cv_events(timestamp_ms, camera_id);

	-- Таблица расшифровок аудиодиалогов (STT)
	CREATE TABLE IF NOT EXISTS speech_transcripts (
		id BIGSERIAL PRIMARY KEY,
		pos_id VARCHAR(50) NOT NULL,
		transcript TEXT NOT NULL,
		timestamp_ms BIGINT NOT NULL,
		duration_ms INTEGER NOT NULL,
		confidence DOUBLE PRECISION NOT NULL,
		model_name VARCHAR(100) NOT NULL,
		weights_version VARCHAR(50) NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_speech_time_pos ON speech_transcripts(timestamp_ms, pos_id);

	-- Правила речевых подсказок (AI Co-Pilot) — extended with suggestion fields
	CREATE TABLE IF NOT EXISTS upsell_rules (
		id SERIAL PRIMARY KEY,
		trigger_category VARCHAR(100) NOT NULL,
		required_keywords TEXT[] NOT NULL,
		suggestion_text TEXT NOT NULL DEFAULT '',
		suggestion_image_url VARCHAR(255) DEFAULT ''
	);

	-- Пользователи для авторизации операторов, администраторов и кассиров
	CREATE TABLE IF NOT EXISTS users (
		id BIGSERIAL PRIMARY KEY,
		username VARCHAR(100) NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		role VARCHAR(20) NOT NULL CHECK (role IN ('admin', 'operator', 'cashier')),
		pos_id VARCHAR(50),
		is_active BOOLEAN NOT NULL DEFAULT TRUE,
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
	CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

	-- Таблица зарегистрированных инцидентов / нарушений
	CREATE TABLE IF NOT EXISTS violations (
		id BIGSERIAL PRIMARY KEY,
		pos_id VARCHAR(50) NOT NULL,
		violation_type VARCHAR(50) NOT NULL,
		timestamp_ms BIGINT NOT NULL,
		proof_video_path VARCHAR(255),
		proof_image_path VARCHAR(255),
		cv_event_id BIGINT REFERENCES cv_events(id) ON DELETE SET NULL,
		pos_event_id BIGINT REFERENCES pos_events(id) ON DELETE SET NULL,
		speech_transcript_id BIGINT REFERENCES speech_transcripts(id) ON DELETE SET NULL,
		confidence_aggregate DOUBLE PRECISION NOT NULL,
		status VARCHAR(20) NOT NULL DEFAULT 'new'
	);
	CREATE INDEX IF NOT EXISTS idx_violations_time ON violations(timestamp_ms);

	-- Таблица задач (очередь видео-экспорта для Python-воркера)
	CREATE TABLE IF NOT EXISTS tasks (
		id BIGSERIAL PRIMARY KEY,
		task_type VARCHAR(50) NOT NULL,
		camera_id VARCHAR(50) NOT NULL,
		violation_id BIGINT REFERENCES violations(id) ON DELETE SET NULL,
		payload JSONB NOT NULL DEFAULT '{}'::jsonb,
		status VARCHAR(20) NOT NULL DEFAULT 'pending',
		result_path VARCHAR(255),
		error_message TEXT,
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
		processed_at TIMESTAMPTZ
	);
	CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
	`

	_, err := pool.Exec(ctx, ddl)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	slog.Info("database migrations completed successfully")
	return nil
}
