-- CreateSchema
CREATE SCHEMA IF NOT EXISTS "public";

-- CreateTable
CREATE TABLE "analytics_events" (
    "id" BIGSERIAL NOT NULL,
    "organization_id" BIGINT NOT NULL,
    "store_id" BIGINT NOT NULL,
    "workplace_id" BIGINT,
    "camera_id" BIGINT,
    "event_type_id" BIGINT NOT NULL,
    "rule_id" BIGINT,
    "started_at" TIMESTAMPTZ(6) NOT NULL,
    "finished_at" TIMESTAMPTZ(6),
    "severity" TEXT NOT NULL DEFAULT 'INFO',
    "status" TEXT NOT NULL DEFAULT 'NEW',
    "confidence" DOUBLE PRECISION,
    "duration_ms" BIGINT,
    "employee_id" BIGINT,
    "shift_id" BIGINT,
    "external_order_id" TEXT,
    "external_receipt_id" TEXT,
    "receipt_id" BIGINT,
    "sale_session_id" BIGINT,
    "violation_type_id" BIGINT,
    "operation_type" TEXT,
    "risk_amount" DECIMAL(18,2),
    "title" TEXT NOT NULL,
    "description" TEXT,
    "metadata" JSONB NOT NULL DEFAULT '{}',
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "processing_session_id" BIGINT,
    "worker_id" BIGINT,
    "deduplication_key" TEXT,
    "received_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "analytics_events_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "camera_metrics" (
    "id" BIGSERIAL NOT NULL,
    "camera_id" BIGINT NOT NULL,
    "recorded_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "status" TEXT NOT NULL DEFAULT 'unknown',
    "input_fps" DOUBLE PRECISION,
    "processing_fps" DOUBLE PRECISION,
    "latency_ms" DOUBLE PRECISION,
    "dropped_frames" BIGINT,
    "reconnect_count" BIGINT,
    "cpu_percent" DOUBLE PRECISION,
    "gpu_percent" DOUBLE PRECISION,
    "gpu_memory_mb" DOUBLE PRECISION,
    "metadata" JSONB NOT NULL DEFAULT '{}',
    "worker_id" BIGINT,

    CONSTRAINT "camera_metrics_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "camera_models" (
    "id" BIGSERIAL NOT NULL,
    "camera_id" BIGINT NOT NULL,
    "model_version_id" BIGINT NOT NULL,
    "roi_id" BIGINT,
    "process_fps" DOUBLE PRECISION,
    "confidence_threshold" DOUBLE PRECISION,
    "config" JSONB NOT NULL DEFAULT '{}',
    "is_enabled" BOOLEAN NOT NULL DEFAULT true,

    CONSTRAINT "camera_models_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "camera_rois" (
    "id" BIGSERIAL NOT NULL,
    "camera_id" BIGINT NOT NULL,
    "roi_type_id" BIGINT NOT NULL,
    "name" TEXT NOT NULL,
    "shape_type" TEXT NOT NULL DEFAULT 'polygon',
    "coordinates" JSONB NOT NULL,
    "is_enabled" BOOLEAN NOT NULL DEFAULT true,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "camera_rois_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "camera_streams" (
    "id" BIGSERIAL NOT NULL,
    "camera_id" BIGINT NOT NULL,
    "stream_type" TEXT NOT NULL,
    "stream_url" TEXT NOT NULL,
    "subtype" TEXT,
    "width" INTEGER,
    "height" INTEGER,
    "source_fps" DOUBLE PRECISION,
    "process_fps" DOUBLE PRECISION,
    "transport" TEXT NOT NULL DEFAULT 'tcp',
    "is_primary" BOOLEAN NOT NULL DEFAULT false,
    "is_enabled" BOOLEAN NOT NULL DEFAULT true,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "camera_streams_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "cameras" (
    "id" BIGSERIAL NOT NULL,
    "store_id" BIGINT NOT NULL,
    "workplace_id" BIGINT NOT NULL,
    "name" TEXT NOT NULL,
    "code" TEXT NOT NULL,
    "manufacturer" TEXT,
    "model" TEXT,
    "nvr_channel" TEXT,
    "location_description" TEXT,
    "status" TEXT NOT NULL DEFAULT 'unknown',
    "last_online_at" TIMESTAMPTZ(6),
    "last_frame_at" TIMESTAMPTZ(6),
    "processing_enabled" BOOLEAN NOT NULL DEFAULT true,
    "is_active" BOOLEAN NOT NULL DEFAULT true,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "cameras_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "detections" (
    "id" BIGSERIAL NOT NULL,
    "camera_id" BIGINT NOT NULL,
    "processing_session_id" BIGINT,
    "model_version_id" BIGINT,
    "roi_id" BIGINT,
    "detected_at" TIMESTAMPTZ(6) NOT NULL,
    "frame_number" BIGINT,
    "class_name" TEXT NOT NULL,
    "confidence" DOUBLE PRECISION NOT NULL,
    "x1" DOUBLE PRECISION NOT NULL,
    "y1" DOUBLE PRECISION NOT NULL,
    "x2" DOUBLE PRECISION NOT NULL,
    "y2" DOUBLE PRECISION NOT NULL,
    "track_id" TEXT,
    "attributes" JSONB NOT NULL DEFAULT '{}',

    CONSTRAINT "detections_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "employees" (
    "id" BIGSERIAL NOT NULL,
    "organization_id" BIGINT NOT NULL,
    "external_id" TEXT,
    "full_name" TEXT NOT NULL,
    "role" TEXT NOT NULL DEFAULT 'cashier',
    "is_active" BOOLEAN NOT NULL DEFAULT true,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "employees_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "event_evidence" (
    "id" BIGSERIAL NOT NULL,
    "event_id" BIGINT NOT NULL,
    "camera_id" BIGINT,
    "receipt_id" BIGINT,
    "evidence_type" TEXT NOT NULL,
    "storage_type" TEXT NOT NULL,
    "file_path" TEXT NOT NULL,
    "mime_type" TEXT,
    "file_size" BIGINT,
    "captured_at" TIMESTAMPTZ(6) NOT NULL,
    "expires_at" TIMESTAMPTZ(6),
    "metadata" JSONB NOT NULL DEFAULT '{}',
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "public_url" TEXT,
    "availability_status" TEXT NOT NULL DEFAULT 'AVAILABLE',
    "video_started_at" TIMESTAMPTZ(6),
    "video_finished_at" TIMESTAMPTZ(6),
    "pre_seconds" INTEGER,
    "post_seconds" INTEGER,

    CONSTRAINT "event_evidence_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "event_objects" (
    "id" BIGSERIAL NOT NULL,
    "event_id" BIGINT NOT NULL,
    "object_role" TEXT,
    "object_type" TEXT NOT NULL,
    "track_id" BIGINT,
    "confidence" DOUBLE PRECISION,
    "bbox" JSONB,
    "attributes" JSONB NOT NULL DEFAULT '{}',
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "event_objects_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "event_reviews" (
    "id" BIGSERIAL NOT NULL,
    "event_id" BIGINT NOT NULL,
    "reviewer_id" BIGINT,
    "decision" TEXT NOT NULL,
    "comment" TEXT,
    "previous_status" TEXT,
    "new_status" TEXT,
    "reviewed_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "event_reviews_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "event_types" (
    "id" BIGSERIAL NOT NULL,
    "code" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "category" TEXT NOT NULL,
    "default_severity" TEXT NOT NULL DEFAULT 'INFO',
    "description" TEXT,

    CONSTRAINT "event_types_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "external_events" (
    "id" BIGSERIAL NOT NULL,
    "source_system" TEXT NOT NULL,
    "event_type" TEXT NOT NULL,
    "external_event_id" TEXT NOT NULL,
    "store_id" BIGINT,
    "workplace_id" BIGINT,
    "occurred_at" TIMESTAMPTZ(6) NOT NULL,
    "received_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "payload" JSONB NOT NULL,
    "processing_status" TEXT NOT NULL DEFAULT 'pending',
    "processing_error" TEXT,
    "organization_id" BIGINT,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "external_events_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "model_versions" (
    "id" BIGSERIAL NOT NULL,
    "model_id" BIGINT NOT NULL,
    "version" TEXT NOT NULL,
    "weights_path" TEXT NOT NULL,
    "config" JSONB NOT NULL DEFAULT '{}',
    "confidence_threshold" DOUBLE PRECISION,
    "iou_threshold" DOUBLE PRECISION,
    "metrics" JSONB NOT NULL DEFAULT '{}',
    "is_active" BOOLEAN NOT NULL DEFAULT true,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "model_versions_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "models" (
    "id" BIGSERIAL NOT NULL,
    "name" TEXT NOT NULL,
    "model_type" TEXT NOT NULL,
    "framework" TEXT NOT NULL,
    "task_type" TEXT NOT NULL,
    "description" TEXT,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "models_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "notifications" (
    "id" BIGSERIAL NOT NULL,
    "event_id" BIGINT NOT NULL,
    "channel" TEXT NOT NULL,
    "recipient" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'PENDING',
    "payload" JSONB NOT NULL DEFAULT '{}',
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "sent_at" TIMESTAMPTZ(6),
    "acknowledged_at" TIMESTAMPTZ(6),
    "error_message" TEXT,
    "attempts" INTEGER NOT NULL DEFAULT 0,

    CONSTRAINT "notifications_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "organizations" (
    "id" BIGSERIAL NOT NULL,
    "name" TEXT NOT NULL,
    "code" TEXT NOT NULL,
    "timezone" TEXT NOT NULL DEFAULT 'UTC',
    "is_active" BOOLEAN NOT NULL DEFAULT true,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "organizations_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "processing_sessions" (
    "id" BIGSERIAL NOT NULL,
    "camera_id" BIGINT NOT NULL,
    "stream_id" BIGINT,
    "worker_name" TEXT NOT NULL,
    "worker_host" TEXT,
    "started_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "finished_at" TIMESTAMPTZ(6),
    "status" TEXT NOT NULL DEFAULT 'running',
    "frames_read" BIGINT NOT NULL DEFAULT 0,
    "frames_processed" BIGINT NOT NULL DEFAULT 0,
    "frames_dropped" BIGINT NOT NULL DEFAULT 0,
    "average_fps" DOUBLE PRECISION,
    "average_latency_ms" DOUBLE PRECISION,
    "error_message" TEXT,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "worker_id" BIGINT,
    "metadata" JSONB NOT NULL DEFAULT '{}',

    CONSTRAINT "processing_sessions_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "roi_types" (
    "id" BIGSERIAL NOT NULL,
    "code" TEXT NOT NULL,
    "name" TEXT NOT NULL,

    CONSTRAINT "roi_types_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "rule_assignments" (
    "id" BIGSERIAL NOT NULL,
    "rule_id" BIGINT NOT NULL,
    "organization_id" BIGINT,
    "store_id" BIGINT,
    "workplace_id" BIGINT,
    "camera_id" BIGINT,
    "settings_override" JSONB NOT NULL DEFAULT '{}',
    "is_enabled" BOOLEAN NOT NULL DEFAULT true,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "rule_assignments_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "rules" (
    "id" BIGSERIAL NOT NULL,
    "code" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "event_type_id" BIGINT NOT NULL,
    "rule_type" TEXT NOT NULL,
    "severity" TEXT NOT NULL DEFAULT 'medium',
    "conditions" JSONB NOT NULL DEFAULT '{}',
    "settings" JSONB NOT NULL DEFAULT '{}',
    "is_active" BOOLEAN NOT NULL DEFAULT true,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "rules_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "shifts" (
    "id" BIGSERIAL NOT NULL,
    "store_id" BIGINT NOT NULL,
    "workplace_id" BIGINT,
    "employee_id" BIGINT,
    "external_shift_id" TEXT,
    "opened_at" TIMESTAMPTZ(6) NOT NULL,
    "closed_at" TIMESTAMPTZ(6),
    "status" TEXT NOT NULL DEFAULT 'open',
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "shifts_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "stores" (
    "id" BIGSERIAL NOT NULL,
    "organization_id" BIGINT NOT NULL,
    "name" TEXT NOT NULL,
    "code" TEXT NOT NULL,
    "city" TEXT,
    "address" TEXT,
    "is_active" BOOLEAN NOT NULL DEFAULT true,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "stores_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "tracks" (
    "id" BIGSERIAL NOT NULL,
    "camera_id" BIGINT NOT NULL,
    "processing_session_id" BIGINT,
    "tracker_track_id" TEXT NOT NULL,
    "object_type" TEXT NOT NULL,
    "first_seen_at" TIMESTAMPTZ(6) NOT NULL,
    "last_seen_at" TIMESTAMPTZ(6) NOT NULL,
    "max_confidence" DOUBLE PRECISION,
    "duration_ms" BIGINT,
    "start_roi_id" BIGINT,
    "end_roi_id" BIGINT,
    "attributes" JSONB NOT NULL DEFAULT '{}',

    CONSTRAINT "tracks_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "workplaces" (
    "id" BIGSERIAL NOT NULL,
    "store_id" BIGINT NOT NULL,
    "name" TEXT NOT NULL,
    "workplace_type" TEXT NOT NULL DEFAULT 'checkout',
    "external_id" TEXT NOT NULL,
    "is_active" BOOLEAN NOT NULL DEFAULT true,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "workplaces_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "product_scans" (
    "id" BIGSERIAL NOT NULL,
    "organization_id" BIGINT NOT NULL,
    "store_id" BIGINT NOT NULL,
    "workplace_id" BIGINT NOT NULL,
    "external_scan_id" TEXT NOT NULL,
    "external_receipt_id" TEXT,
    "receipt_id" BIGINT,
    "barcode" TEXT NOT NULL,
    "product_name" TEXT,
    "quantity" DECIMAL(18,6),
    "price" DECIMAL(18,2),
    "currency" TEXT,
    "occurred_at" TIMESTAMPTZ(6) NOT NULL,
    "received_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "payload" JSONB NOT NULL DEFAULT '{}',

    CONSTRAINT "product_scans_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "receipts" (
    "id" BIGSERIAL NOT NULL,
    "organization_id" BIGINT NOT NULL,
    "store_id" BIGINT NOT NULL,
    "workplace_id" BIGINT NOT NULL,
    "external_receipt_id" TEXT NOT NULL,
    "external_order_id" TEXT,
    "cashier_external_id" TEXT,
    "employee_id" BIGINT,
    "operation_type" TEXT NOT NULL DEFAULT 'SALE',
    "receipt_status" TEXT NOT NULL DEFAULT 'CLOSED',
    "payment_method" TEXT,
    "receipt_total" DECIMAL(18,2),
    "paid_amount" DECIMAL(18,2),
    "change_amount" DECIMAL(18,2),
    "bonus_amount" DECIMAL(18,2),
    "discount_amount" DECIMAL(18,2),
    "occurred_at" TIMESTAMPTZ(6) NOT NULL,
    "printed_at" TIMESTAMPTZ(6),
    "closed_at" TIMESTAMPTZ(6),
    "received_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "items" JSONB NOT NULL DEFAULT '[]',
    "totals" JSONB NOT NULL DEFAULT '{}',
    "payload" JSONB NOT NULL DEFAULT '{}',

    CONSTRAINT "receipts_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "receipt_items" (
    "id" BIGSERIAL NOT NULL,
    "receipt_id" BIGINT NOT NULL,
    "organization_id" BIGINT NOT NULL,
    "store_id" BIGINT NOT NULL,
    "workplace_id" BIGINT NOT NULL,
    "line_number" INTEGER NOT NULL,
    "external_product_id" TEXT,
    "barcode" TEXT,
    "product_name" TEXT NOT NULL,
    "quantity" DECIMAL(18,6) NOT NULL,
    "price" DECIMAL(18,2) NOT NULL,
    "line_total" DECIMAL(18,2) NOT NULL,
    "discount_amount" DECIMAL(18,2),
    "is_container" BOOLEAN NOT NULL DEFAULT false,
    "container_type" TEXT,
    "payload" JSONB NOT NULL DEFAULT '{}',

    CONSTRAINT "receipt_items_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "sale_sessions" (
    "id" BIGSERIAL NOT NULL,
    "organization_id" BIGINT NOT NULL,
    "store_id" BIGINT NOT NULL,
    "workplace_id" BIGINT NOT NULL,
    "receipt_id" BIGINT,
    "employee_id" BIGINT,
    "operation_type" TEXT NOT NULL DEFAULT 'SALE',
    "started_at" TIMESTAMPTZ(6) NOT NULL,
    "finished_at" TIMESTAMPTZ(6),
    "customer_present" BOOLEAN,
    "customer_present_confidence" DOUBLE PRECISION,
    "service_score" DOUBLE PRECISION,
    "status" TEXT NOT NULL DEFAULT 'OPEN',
    "metadata" JSONB NOT NULL DEFAULT '{}',
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "sale_sessions_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "video_observations" (
    "id" BIGSERIAL NOT NULL,
    "organization_id" BIGINT NOT NULL,
    "store_id" BIGINT NOT NULL,
    "workplace_id" BIGINT NOT NULL,
    "camera_id" BIGINT,
    "receipt_id" BIGINT,
    "sale_session_id" BIGINT,
    "evidence_id" BIGINT,
    "observed_at" TIMESTAMPTZ(6) NOT NULL,
    "observation_type" TEXT NOT NULL,
    "barcode" TEXT,
    "product_name" TEXT,
    "quantity" DECIMAL(18,6),
    "confidence" DOUBLE PRECISION,
    "metadata" JSONB NOT NULL DEFAULT '{}',
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "video_observations_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "service_check_results" (
    "id" BIGSERIAL NOT NULL,
    "organization_id" BIGINT NOT NULL,
    "sale_session_id" BIGINT NOT NULL,
    "criterion_code" TEXT NOT NULL,
    "result" TEXT NOT NULL,
    "confidence" DOUBLE PRECISION,
    "evidence_id" BIGINT,
    "comment" TEXT,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "service_check_results_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "violation_types" (
    "id" BIGSERIAL NOT NULL,
    "organization_id" BIGINT,
    "code" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "risk_level" TEXT NOT NULL,
    "employee_notification_text" TEXT,
    "visible_to_roles" JSONB NOT NULL DEFAULT '[]',
    "is_active" BOOLEAN NOT NULL DEFAULT true,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "violation_types_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "integration_errors" (
    "id" BIGSERIAL NOT NULL,
    "organization_id" BIGINT,
    "store_id" BIGINT,
    "workplace_id" BIGINT,
    "source_system" TEXT NOT NULL,
    "entity_type" TEXT NOT NULL,
    "external_id" TEXT,
    "error_code" TEXT NOT NULL,
    "error_message" TEXT NOT NULL,
    "payload" JSONB NOT NULL DEFAULT '{}',
    "status" TEXT NOT NULL DEFAULT 'OPEN',
    "occurred_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "resolved_at" TIMESTAMPTZ(6),

    CONSTRAINT "integration_errors_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "receivings" (
    "id" BIGSERIAL NOT NULL,
    "organization_id" BIGINT NOT NULL,
    "store_id" BIGINT NOT NULL,
    "workplace_id" BIGINT,
    "employee_id" BIGINT,
    "external_invoice_id" TEXT NOT NULL,
    "supplier_id" TEXT,
    "supplier_name" TEXT,
    "started_at" TIMESTAMPTZ(6) NOT NULL,
    "finished_at" TIMESTAMPTZ(6),
    "status" TEXT NOT NULL DEFAULT 'OPEN',
    "payload" JSONB NOT NULL DEFAULT '{}',

    CONSTRAINT "receivings_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "receiving_items" (
    "id" BIGSERIAL NOT NULL,
    "receiving_id" BIGINT NOT NULL,
    "organization_id" BIGINT NOT NULL,
    "store_id" BIGINT NOT NULL,
    "workplace_id" BIGINT,
    "external_product_id" TEXT,
    "barcode" TEXT,
    "product_name" TEXT NOT NULL,
    "expected_quantity" DECIMAL(18,6) NOT NULL,
    "actual_quantity" DECIMAL(18,6),
    "expiry_date" DATE,
    "package_damaged" BOOLEAN,
    "defect_detected" BOOLEAN,
    "discrepancy_reported" BOOLEAN,
    "payload" JSONB NOT NULL DEFAULT '{}',

    CONSTRAINT "receiving_items_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "analytics_workers" (
    "id" BIGSERIAL NOT NULL,
    "organization_id" BIGINT NOT NULL,
    "name" TEXT NOT NULL,
    "api_key_hash" TEXT NOT NULL,
    "host" TEXT NOT NULL,
    "version" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'OFFLINE',
    "last_heartbeat_at" TIMESTAMPTZ(6),
    "capabilities" JSONB NOT NULL DEFAULT '{}',
    "metadata" JSONB NOT NULL DEFAULT '{}',
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "analytics_workers_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "audit_logs" (
    "id" BIGSERIAL NOT NULL,
    "organization_id" BIGINT,
    "user_id" BIGINT,
    "action" TEXT NOT NULL,
    "entity_type" TEXT NOT NULL,
    "entity_id" TEXT,
    "old_value" JSONB,
    "new_value" JSONB,
    "ip_address" TEXT,
    "user_agent" TEXT,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "audit_logs_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "event_transcripts" (
    "id" BIGSERIAL NOT NULL,
    "event_id" BIGINT,
    "organization_id" BIGINT,
    "store_id" BIGINT,
    "workplace_id" BIGINT,
    "camera_id" BIGINT,
    "receipt_id" BIGINT,
    "sale_session_id" BIGINT,
    "external_transcript_id" TEXT,
    "source_service" TEXT,
    "audio_url" TEXT,
    "started_at" TIMESTAMPTZ(6) NOT NULL,
    "finished_at" TIMESTAMPTZ(6),
    "speaker" TEXT NOT NULL DEFAULT 'UNKNOWN',
    "text" TEXT NOT NULL,
    "language" TEXT,
    "confidence" DOUBLE PRECISION,
    "words" JSONB,
    "metadata" JSONB NOT NULL DEFAULT '{}',
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "event_transcripts_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "refresh_tokens" (
    "id" BIGSERIAL NOT NULL,
    "user_id" BIGINT NOT NULL,
    "token_hash" TEXT NOT NULL,
    "expires_at" TIMESTAMPTZ(6) NOT NULL,
    "revoked_at" TIMESTAMPTZ(6),
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "ip_address" TEXT,
    "user_agent" TEXT,

    CONSTRAINT "refresh_tokens_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "users" (
    "id" BIGSERIAL NOT NULL,
    "organization_id" BIGINT,
    "email" TEXT NOT NULL,
    "password_hash" TEXT NOT NULL,
    "full_name" TEXT NOT NULL,
    "role" TEXT NOT NULL,
    "is_active" BOOLEAN NOT NULL DEFAULT true,
    "last_login_at" TIMESTAMPTZ(6),
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "users_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "worker_camera_assignments" (
    "id" BIGSERIAL NOT NULL,
    "worker_id" BIGINT NOT NULL,
    "camera_id" BIGINT NOT NULL,
    "is_enabled" BOOLEAN NOT NULL DEFAULT true,
    "config_version" INTEGER NOT NULL DEFAULT 1,
    "assigned_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "worker_camera_assignments_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE INDEX "idx_analytics_events_camera_time" ON "analytics_events"("camera_id", "started_at" DESC);

-- CreateIndex
CREATE INDEX "idx_analytics_events_metadata_gin" ON "analytics_events" USING GIN ("metadata");

-- CreateIndex
CREATE INDEX "idx_analytics_events_org_time" ON "analytics_events"("organization_id", "started_at" DESC);

-- CreateIndex
CREATE INDEX "idx_analytics_events_status" ON "analytics_events"("status", "severity", "started_at" DESC);

-- CreateIndex
CREATE INDEX "idx_analytics_events_store_time" ON "analytics_events"("store_id", "started_at" DESC);

-- CreateIndex
CREATE INDEX "idx_analytics_events_org_created" ON "analytics_events"("organization_id", "created_at" DESC);

-- CreateIndex
CREATE INDEX "idx_analytics_events_receipt" ON "analytics_events"("receipt_id");

-- CreateIndex
CREATE INDEX "idx_analytics_events_sale_session" ON "analytics_events"("sale_session_id");

-- CreateIndex
CREATE INDEX "idx_analytics_events_violation_type" ON "analytics_events"("violation_type_id");

-- CreateIndex
CREATE INDEX "idx_camera_metrics_camera_time" ON "camera_metrics"("camera_id", "recorded_at" DESC);

-- CreateIndex
CREATE INDEX "idx_camera_models_camera" ON "camera_models"("camera_id");

-- CreateIndex
CREATE UNIQUE INDEX "camera_models_camera_id_model_version_id_roi_id_key" ON "camera_models"("camera_id", "model_version_id", "roi_id");

-- CreateIndex
CREATE INDEX "idx_camera_rois_camera" ON "camera_rois"("camera_id");

-- CreateIndex
CREATE UNIQUE INDEX "camera_rois_camera_id_name_key" ON "camera_rois"("camera_id", "name");

-- CreateIndex
CREATE INDEX "idx_camera_streams_camera" ON "camera_streams"("camera_id");

-- CreateIndex
CREATE INDEX "idx_cameras_store" ON "cameras"("store_id");

-- CreateIndex
CREATE INDEX "idx_cameras_workplace" ON "cameras"("workplace_id");

-- CreateIndex
CREATE UNIQUE INDEX "cameras_store_id_code_key" ON "cameras"("store_id", "code");

-- CreateIndex
CREATE INDEX "idx_detections_attributes_gin" ON "detections" USING GIN ("attributes");

-- CreateIndex
CREATE INDEX "idx_detections_camera_time" ON "detections"("camera_id", "detected_at" DESC);

-- CreateIndex
CREATE INDEX "idx_detections_session_frame" ON "detections"("processing_session_id", "frame_number");

-- CreateIndex
CREATE UNIQUE INDEX "employees_organization_id_external_id_key" ON "employees"("organization_id", "external_id");

-- CreateIndex
CREATE INDEX "idx_event_evidence_event" ON "event_evidence"("event_id", "captured_at");

-- CreateIndex
CREATE INDEX "idx_event_evidence_receipt" ON "event_evidence"("receipt_id");

-- CreateIndex
CREATE INDEX "idx_event_evidence_camera" ON "event_evidence"("camera_id", "captured_at");

-- CreateIndex
CREATE INDEX "idx_event_objects_event" ON "event_objects"("event_id");

-- CreateIndex
CREATE INDEX "idx_event_reviews_event" ON "event_reviews"("event_id", "reviewed_at" DESC);

-- CreateIndex
CREATE UNIQUE INDEX "event_types_code_key" ON "event_types"("code");

-- CreateIndex
CREATE INDEX "idx_external_events_payload_gin" ON "external_events" USING GIN ("payload");

-- CreateIndex
CREATE INDEX "idx_external_events_status" ON "external_events"("processing_status", "received_at");

-- CreateIndex
CREATE UNIQUE INDEX "external_events_source_system_external_event_id_key" ON "external_events"("source_system", "external_event_id");

-- CreateIndex
CREATE UNIQUE INDEX "model_versions_model_id_version_key" ON "model_versions"("model_id", "version");

-- CreateIndex
CREATE UNIQUE INDEX "models_name_model_type_key" ON "models"("name", "model_type");

-- CreateIndex
CREATE INDEX "idx_notifications_status" ON "notifications"("status", "created_at");

-- CreateIndex
CREATE UNIQUE INDEX "organizations_code_key" ON "organizations"("code");

-- CreateIndex
CREATE INDEX "idx_processing_sessions_camera_started" ON "processing_sessions"("camera_id", "started_at" DESC);

-- CreateIndex
CREATE UNIQUE INDEX "roi_types_code_key" ON "roi_types"("code");

-- CreateIndex
CREATE UNIQUE INDEX "rules_code_key" ON "rules"("code");

-- CreateIndex
CREATE INDEX "idx_shifts_employee_time" ON "shifts"("employee_id", "opened_at" DESC);

-- CreateIndex
CREATE UNIQUE INDEX "shifts_store_id_external_shift_id_key" ON "shifts"("store_id", "external_shift_id");

-- CreateIndex
CREATE INDEX "idx_stores_organization" ON "stores"("organization_id");

-- CreateIndex
CREATE UNIQUE INDEX "stores_organization_id_code_key" ON "stores"("organization_id", "code");

-- CreateIndex
CREATE INDEX "idx_tracks_camera_time" ON "tracks"("camera_id", "first_seen_at" DESC);

-- CreateIndex
CREATE UNIQUE INDEX "tracks_processing_session_id_tracker_track_id_key" ON "tracks"("processing_session_id", "tracker_track_id");

-- CreateIndex
CREATE INDEX "idx_workplaces_store" ON "workplaces"("store_id");

-- CreateIndex
CREATE UNIQUE INDEX "workplaces_store_id_external_id_key" ON "workplaces"("store_id", "external_id");

-- CreateIndex
CREATE INDEX "idx_product_scans_receipt_id" ON "product_scans"("receipt_id");

-- CreateIndex
CREATE INDEX "idx_product_scans_workplace_time" ON "product_scans"("workplace_id", "occurred_at" DESC);

-- CreateIndex
CREATE INDEX "idx_product_scans_receipt" ON "product_scans"("external_receipt_id");

-- CreateIndex
CREATE UNIQUE INDEX "product_scans_organization_id_external_scan_id_key" ON "product_scans"("organization_id", "external_scan_id");

-- CreateIndex
CREATE INDEX "idx_receipts_workplace_time" ON "receipts"("workplace_id", "occurred_at" DESC);

-- CreateIndex
CREATE INDEX "idx_receipts_order" ON "receipts"("external_order_id");

-- CreateIndex
CREATE INDEX "idx_receipts_employee_time" ON "receipts"("employee_id", "occurred_at" DESC);

-- CreateIndex
CREATE UNIQUE INDEX "receipts_organization_id_external_receipt_id_key" ON "receipts"("organization_id", "external_receipt_id");

-- CreateIndex
CREATE INDEX "idx_receipt_items_barcode" ON "receipt_items"("barcode");

-- CreateIndex
CREATE INDEX "idx_receipt_items_product" ON "receipt_items"("organization_id", "external_product_id");

-- CreateIndex
CREATE UNIQUE INDEX "receipt_items_receipt_id_line_number_key" ON "receipt_items"("receipt_id", "line_number");

-- CreateIndex
CREATE INDEX "idx_sale_sessions_workplace_time" ON "sale_sessions"("workplace_id", "started_at" DESC);

-- CreateIndex
CREATE INDEX "idx_sale_sessions_receipt" ON "sale_sessions"("receipt_id");

-- CreateIndex
CREATE INDEX "idx_sale_sessions_employee_time" ON "sale_sessions"("employee_id", "started_at" DESC);

-- CreateIndex
CREATE UNIQUE INDEX "sale_sessions_receipt_id_key" ON "sale_sessions"("receipt_id");

-- CreateIndex
CREATE INDEX "idx_video_observations_workplace_time" ON "video_observations"("workplace_id", "observed_at" DESC);

-- CreateIndex
CREATE INDEX "idx_video_observations_receipt" ON "video_observations"("receipt_id");

-- CreateIndex
CREATE INDEX "idx_video_observations_sale_session" ON "video_observations"("sale_session_id");

-- CreateIndex
CREATE INDEX "idx_video_observations_type_time" ON "video_observations"("observation_type", "observed_at" DESC);

-- CreateIndex
CREATE INDEX "idx_service_checks_criterion" ON "service_check_results"("organization_id", "criterion_code");

-- CreateIndex
CREATE UNIQUE INDEX "service_check_results_sale_session_id_criterion_code_key" ON "service_check_results"("sale_session_id", "criterion_code");

-- CreateIndex
CREATE INDEX "idx_violation_types_risk" ON "violation_types"("risk_level");

-- CreateIndex
CREATE UNIQUE INDEX "violation_types_organization_id_code_key" ON "violation_types"("organization_id", "code");

-- CreateIndex
CREATE INDEX "idx_integration_errors_status" ON "integration_errors"("status", "occurred_at" DESC);

-- CreateIndex
CREATE INDEX "idx_integration_errors_external" ON "integration_errors"("source_system", "entity_type", "external_id");

-- CreateIndex
CREATE INDEX "idx_receivings_store_time" ON "receivings"("store_id", "started_at" DESC);

-- CreateIndex
CREATE UNIQUE INDEX "receivings_organization_id_external_invoice_id_key" ON "receivings"("organization_id", "external_invoice_id");

-- CreateIndex
CREATE INDEX "idx_receiving_items_receiving" ON "receiving_items"("receiving_id");

-- CreateIndex
CREATE INDEX "idx_receiving_items_barcode" ON "receiving_items"("barcode");

-- CreateIndex
CREATE INDEX "idx_analytics_workers_heartbeat" ON "analytics_workers"("last_heartbeat_at");

-- CreateIndex
CREATE UNIQUE INDEX "analytics_workers_organization_id_name_key" ON "analytics_workers"("organization_id", "name");

-- CreateIndex
CREATE INDEX "idx_audit_logs_org_created" ON "audit_logs"("organization_id", "created_at" DESC);

-- CreateIndex
CREATE INDEX "idx_event_transcripts_event" ON "event_transcripts"("event_id", "started_at");

-- CreateIndex
CREATE INDEX "idx_event_transcripts_receipt" ON "event_transcripts"("receipt_id", "started_at");

-- CreateIndex
CREATE INDEX "idx_event_transcripts_sale_session" ON "event_transcripts"("sale_session_id", "started_at");

-- CreateIndex
CREATE INDEX "idx_event_transcripts_workplace" ON "event_transcripts"("workplace_id", "started_at");

-- CreateIndex
CREATE INDEX "idx_event_transcripts_camera" ON "event_transcripts"("camera_id", "started_at");

-- CreateIndex
CREATE UNIQUE INDEX "event_transcripts_source_service_external_transcript_id_key" ON "event_transcripts"("source_service", "external_transcript_id");

-- CreateIndex
CREATE UNIQUE INDEX "refresh_tokens_token_hash_key" ON "refresh_tokens"("token_hash");

-- CreateIndex
CREATE INDEX "idx_refresh_tokens_user_expires" ON "refresh_tokens"("user_id", "expires_at");

-- CreateIndex
CREATE UNIQUE INDEX "users_email_key" ON "users"("email");

-- CreateIndex
CREATE UNIQUE INDEX "worker_camera_assignments_camera_id_key" ON "worker_camera_assignments"("camera_id");

-- AddForeignKey
ALTER TABLE "analytics_events" ADD CONSTRAINT "analytics_events_camera_id_fkey" FOREIGN KEY ("camera_id") REFERENCES "cameras"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "analytics_events" ADD CONSTRAINT "analytics_events_employee_id_fkey" FOREIGN KEY ("employee_id") REFERENCES "employees"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "analytics_events" ADD CONSTRAINT "analytics_events_event_type_id_fkey" FOREIGN KEY ("event_type_id") REFERENCES "event_types"("id") ON DELETE RESTRICT ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "analytics_events" ADD CONSTRAINT "analytics_events_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "analytics_events" ADD CONSTRAINT "analytics_events_processing_session_id_fkey" FOREIGN KEY ("processing_session_id") REFERENCES "processing_sessions"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "analytics_events" ADD CONSTRAINT "analytics_events_rule_id_fkey" FOREIGN KEY ("rule_id") REFERENCES "rules"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "analytics_events" ADD CONSTRAINT "analytics_events_shift_id_fkey" FOREIGN KEY ("shift_id") REFERENCES "shifts"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "analytics_events" ADD CONSTRAINT "analytics_events_store_id_fkey" FOREIGN KEY ("store_id") REFERENCES "stores"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "analytics_events" ADD CONSTRAINT "analytics_events_worker_id_fkey" FOREIGN KEY ("worker_id") REFERENCES "analytics_workers"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "analytics_events" ADD CONSTRAINT "analytics_events_workplace_id_fkey" FOREIGN KEY ("workplace_id") REFERENCES "workplaces"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "analytics_events" ADD CONSTRAINT "analytics_events_receipt_id_fkey" FOREIGN KEY ("receipt_id") REFERENCES "receipts"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "analytics_events" ADD CONSTRAINT "analytics_events_sale_session_id_fkey" FOREIGN KEY ("sale_session_id") REFERENCES "sale_sessions"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "analytics_events" ADD CONSTRAINT "analytics_events_violation_type_id_fkey" FOREIGN KEY ("violation_type_id") REFERENCES "violation_types"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "camera_metrics" ADD CONSTRAINT "camera_metrics_camera_id_fkey" FOREIGN KEY ("camera_id") REFERENCES "cameras"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "camera_metrics" ADD CONSTRAINT "camera_metrics_worker_id_fkey" FOREIGN KEY ("worker_id") REFERENCES "analytics_workers"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "camera_models" ADD CONSTRAINT "camera_models_camera_id_fkey" FOREIGN KEY ("camera_id") REFERENCES "cameras"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "camera_models" ADD CONSTRAINT "camera_models_model_version_id_fkey" FOREIGN KEY ("model_version_id") REFERENCES "model_versions"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "camera_models" ADD CONSTRAINT "camera_models_roi_id_fkey" FOREIGN KEY ("roi_id") REFERENCES "camera_rois"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "camera_rois" ADD CONSTRAINT "camera_rois_camera_id_fkey" FOREIGN KEY ("camera_id") REFERENCES "cameras"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "camera_rois" ADD CONSTRAINT "camera_rois_roi_type_id_fkey" FOREIGN KEY ("roi_type_id") REFERENCES "roi_types"("id") ON DELETE RESTRICT ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "camera_streams" ADD CONSTRAINT "camera_streams_camera_id_fkey" FOREIGN KEY ("camera_id") REFERENCES "cameras"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "cameras" ADD CONSTRAINT "cameras_store_id_fkey" FOREIGN KEY ("store_id") REFERENCES "stores"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "cameras" ADD CONSTRAINT "cameras_workplace_id_fkey" FOREIGN KEY ("workplace_id") REFERENCES "workplaces"("id") ON DELETE RESTRICT ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "detections" ADD CONSTRAINT "detections_camera_id_fkey" FOREIGN KEY ("camera_id") REFERENCES "cameras"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "detections" ADD CONSTRAINT "detections_model_version_id_fkey" FOREIGN KEY ("model_version_id") REFERENCES "model_versions"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "detections" ADD CONSTRAINT "detections_processing_session_id_fkey" FOREIGN KEY ("processing_session_id") REFERENCES "processing_sessions"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "detections" ADD CONSTRAINT "detections_roi_id_fkey" FOREIGN KEY ("roi_id") REFERENCES "camera_rois"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "employees" ADD CONSTRAINT "employees_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "event_evidence" ADD CONSTRAINT "event_evidence_event_id_fkey" FOREIGN KEY ("event_id") REFERENCES "analytics_events"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "event_evidence" ADD CONSTRAINT "event_evidence_camera_id_fkey" FOREIGN KEY ("camera_id") REFERENCES "cameras"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "event_evidence" ADD CONSTRAINT "event_evidence_receipt_id_fkey" FOREIGN KEY ("receipt_id") REFERENCES "receipts"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "event_objects" ADD CONSTRAINT "event_objects_event_id_fkey" FOREIGN KEY ("event_id") REFERENCES "analytics_events"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "event_objects" ADD CONSTRAINT "event_objects_track_id_fkey" FOREIGN KEY ("track_id") REFERENCES "tracks"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "event_reviews" ADD CONSTRAINT "event_reviews_event_id_fkey" FOREIGN KEY ("event_id") REFERENCES "analytics_events"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "event_reviews" ADD CONSTRAINT "event_reviews_reviewer_id_fkey" FOREIGN KEY ("reviewer_id") REFERENCES "employees"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "external_events" ADD CONSTRAINT "external_events_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "external_events" ADD CONSTRAINT "external_events_store_id_fkey" FOREIGN KEY ("store_id") REFERENCES "stores"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "external_events" ADD CONSTRAINT "external_events_workplace_id_fkey" FOREIGN KEY ("workplace_id") REFERENCES "workplaces"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "model_versions" ADD CONSTRAINT "model_versions_model_id_fkey" FOREIGN KEY ("model_id") REFERENCES "models"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "notifications" ADD CONSTRAINT "notifications_event_id_fkey" FOREIGN KEY ("event_id") REFERENCES "analytics_events"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "processing_sessions" ADD CONSTRAINT "processing_sessions_camera_id_fkey" FOREIGN KEY ("camera_id") REFERENCES "cameras"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "processing_sessions" ADD CONSTRAINT "processing_sessions_stream_id_fkey" FOREIGN KEY ("stream_id") REFERENCES "camera_streams"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "processing_sessions" ADD CONSTRAINT "processing_sessions_worker_id_fkey" FOREIGN KEY ("worker_id") REFERENCES "analytics_workers"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "rule_assignments" ADD CONSTRAINT "rule_assignments_camera_id_fkey" FOREIGN KEY ("camera_id") REFERENCES "cameras"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "rule_assignments" ADD CONSTRAINT "rule_assignments_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "rule_assignments" ADD CONSTRAINT "rule_assignments_rule_id_fkey" FOREIGN KEY ("rule_id") REFERENCES "rules"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "rule_assignments" ADD CONSTRAINT "rule_assignments_store_id_fkey" FOREIGN KEY ("store_id") REFERENCES "stores"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "rule_assignments" ADD CONSTRAINT "rule_assignments_workplace_id_fkey" FOREIGN KEY ("workplace_id") REFERENCES "workplaces"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "rules" ADD CONSTRAINT "rules_event_type_id_fkey" FOREIGN KEY ("event_type_id") REFERENCES "event_types"("id") ON DELETE RESTRICT ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "shifts" ADD CONSTRAINT "shifts_employee_id_fkey" FOREIGN KEY ("employee_id") REFERENCES "employees"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "shifts" ADD CONSTRAINT "shifts_store_id_fkey" FOREIGN KEY ("store_id") REFERENCES "stores"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "shifts" ADD CONSTRAINT "shifts_workplace_id_fkey" FOREIGN KEY ("workplace_id") REFERENCES "workplaces"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "stores" ADD CONSTRAINT "stores_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "tracks" ADD CONSTRAINT "tracks_camera_id_fkey" FOREIGN KEY ("camera_id") REFERENCES "cameras"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "tracks" ADD CONSTRAINT "tracks_end_roi_id_fkey" FOREIGN KEY ("end_roi_id") REFERENCES "camera_rois"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "tracks" ADD CONSTRAINT "tracks_processing_session_id_fkey" FOREIGN KEY ("processing_session_id") REFERENCES "processing_sessions"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "tracks" ADD CONSTRAINT "tracks_start_roi_id_fkey" FOREIGN KEY ("start_roi_id") REFERENCES "camera_rois"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "workplaces" ADD CONSTRAINT "workplaces_store_id_fkey" FOREIGN KEY ("store_id") REFERENCES "stores"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "product_scans" ADD CONSTRAINT "product_scans_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "product_scans" ADD CONSTRAINT "product_scans_receipt_id_fkey" FOREIGN KEY ("receipt_id") REFERENCES "receipts"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "product_scans" ADD CONSTRAINT "product_scans_store_id_fkey" FOREIGN KEY ("store_id") REFERENCES "stores"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "product_scans" ADD CONSTRAINT "product_scans_workplace_id_fkey" FOREIGN KEY ("workplace_id") REFERENCES "workplaces"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "receipts" ADD CONSTRAINT "receipts_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "receipts" ADD CONSTRAINT "receipts_employee_id_fkey" FOREIGN KEY ("employee_id") REFERENCES "employees"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "receipts" ADD CONSTRAINT "receipts_store_id_fkey" FOREIGN KEY ("store_id") REFERENCES "stores"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "receipts" ADD CONSTRAINT "receipts_workplace_id_fkey" FOREIGN KEY ("workplace_id") REFERENCES "workplaces"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "receipt_items" ADD CONSTRAINT "receipt_items_receipt_id_fkey" FOREIGN KEY ("receipt_id") REFERENCES "receipts"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "receipt_items" ADD CONSTRAINT "receipt_items_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "receipt_items" ADD CONSTRAINT "receipt_items_store_id_fkey" FOREIGN KEY ("store_id") REFERENCES "stores"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "receipt_items" ADD CONSTRAINT "receipt_items_workplace_id_fkey" FOREIGN KEY ("workplace_id") REFERENCES "workplaces"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "sale_sessions" ADD CONSTRAINT "sale_sessions_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "sale_sessions" ADD CONSTRAINT "sale_sessions_store_id_fkey" FOREIGN KEY ("store_id") REFERENCES "stores"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "sale_sessions" ADD CONSTRAINT "sale_sessions_workplace_id_fkey" FOREIGN KEY ("workplace_id") REFERENCES "workplaces"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "sale_sessions" ADD CONSTRAINT "sale_sessions_receipt_id_fkey" FOREIGN KEY ("receipt_id") REFERENCES "receipts"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "sale_sessions" ADD CONSTRAINT "sale_sessions_employee_id_fkey" FOREIGN KEY ("employee_id") REFERENCES "employees"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "video_observations" ADD CONSTRAINT "video_observations_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "video_observations" ADD CONSTRAINT "video_observations_store_id_fkey" FOREIGN KEY ("store_id") REFERENCES "stores"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "video_observations" ADD CONSTRAINT "video_observations_workplace_id_fkey" FOREIGN KEY ("workplace_id") REFERENCES "workplaces"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "video_observations" ADD CONSTRAINT "video_observations_camera_id_fkey" FOREIGN KEY ("camera_id") REFERENCES "cameras"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "video_observations" ADD CONSTRAINT "video_observations_receipt_id_fkey" FOREIGN KEY ("receipt_id") REFERENCES "receipts"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "video_observations" ADD CONSTRAINT "video_observations_sale_session_id_fkey" FOREIGN KEY ("sale_session_id") REFERENCES "sale_sessions"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "video_observations" ADD CONSTRAINT "video_observations_evidence_id_fkey" FOREIGN KEY ("evidence_id") REFERENCES "event_evidence"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "service_check_results" ADD CONSTRAINT "service_check_results_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "service_check_results" ADD CONSTRAINT "service_check_results_sale_session_id_fkey" FOREIGN KEY ("sale_session_id") REFERENCES "sale_sessions"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "service_check_results" ADD CONSTRAINT "service_check_results_evidence_id_fkey" FOREIGN KEY ("evidence_id") REFERENCES "event_evidence"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "violation_types" ADD CONSTRAINT "violation_types_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "integration_errors" ADD CONSTRAINT "integration_errors_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "integration_errors" ADD CONSTRAINT "integration_errors_store_id_fkey" FOREIGN KEY ("store_id") REFERENCES "stores"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "integration_errors" ADD CONSTRAINT "integration_errors_workplace_id_fkey" FOREIGN KEY ("workplace_id") REFERENCES "workplaces"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "receivings" ADD CONSTRAINT "receivings_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "receivings" ADD CONSTRAINT "receivings_store_id_fkey" FOREIGN KEY ("store_id") REFERENCES "stores"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "receivings" ADD CONSTRAINT "receivings_workplace_id_fkey" FOREIGN KEY ("workplace_id") REFERENCES "workplaces"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "receivings" ADD CONSTRAINT "receivings_employee_id_fkey" FOREIGN KEY ("employee_id") REFERENCES "employees"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "receiving_items" ADD CONSTRAINT "receiving_items_receiving_id_fkey" FOREIGN KEY ("receiving_id") REFERENCES "receivings"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "receiving_items" ADD CONSTRAINT "receiving_items_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "receiving_items" ADD CONSTRAINT "receiving_items_store_id_fkey" FOREIGN KEY ("store_id") REFERENCES "stores"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "receiving_items" ADD CONSTRAINT "receiving_items_workplace_id_fkey" FOREIGN KEY ("workplace_id") REFERENCES "workplaces"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "analytics_workers" ADD CONSTRAINT "analytics_workers_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "audit_logs" ADD CONSTRAINT "audit_logs_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "audit_logs" ADD CONSTRAINT "audit_logs_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "event_transcripts" ADD CONSTRAINT "event_transcripts_event_id_fkey" FOREIGN KEY ("event_id") REFERENCES "analytics_events"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "event_transcripts" ADD CONSTRAINT "event_transcripts_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "event_transcripts" ADD CONSTRAINT "event_transcripts_store_id_fkey" FOREIGN KEY ("store_id") REFERENCES "stores"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "event_transcripts" ADD CONSTRAINT "event_transcripts_workplace_id_fkey" FOREIGN KEY ("workplace_id") REFERENCES "workplaces"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "event_transcripts" ADD CONSTRAINT "event_transcripts_camera_id_fkey" FOREIGN KEY ("camera_id") REFERENCES "cameras"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "event_transcripts" ADD CONSTRAINT "event_transcripts_receipt_id_fkey" FOREIGN KEY ("receipt_id") REFERENCES "receipts"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "event_transcripts" ADD CONSTRAINT "event_transcripts_sale_session_id_fkey" FOREIGN KEY ("sale_session_id") REFERENCES "sale_sessions"("id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "refresh_tokens" ADD CONSTRAINT "refresh_tokens_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "users" ADD CONSTRAINT "users_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "worker_camera_assignments" ADD CONSTRAINT "worker_camera_assignments_camera_id_fkey" FOREIGN KEY ("camera_id") REFERENCES "cameras"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "worker_camera_assignments" ADD CONSTRAINT "worker_camera_assignments_worker_id_fkey" FOREIGN KEY ("worker_id") REFERENCES "analytics_workers"("id") ON DELETE CASCADE ON UPDATE NO ACTION;

