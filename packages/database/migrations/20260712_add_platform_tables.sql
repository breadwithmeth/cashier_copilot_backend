BEGIN;

CREATE TABLE IF NOT EXISTS users (
  id BIGSERIAL PRIMARY KEY,
  organization_id BIGINT REFERENCES organizations(id) ON DELETE CASCADE,
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  full_name TEXT NOT NULL,
  role TEXT NOT NULL CHECK (role IN ('SUPER_ADMIN','ORGANIZATION_ADMIN','MANAGER','OPERATOR','TECHNICIAN','VIEWER')),
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  last_login_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (role = 'SUPER_ADMIN' OR organization_id IS NOT NULL)
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  ip_address TEXT,
  user_agent TEXT
);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_expires ON refresh_tokens(user_id, expires_at);

CREATE TABLE IF NOT EXISTS analytics_workers (
  id BIGSERIAL PRIMARY KEY,
  organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  api_key_hash TEXT NOT NULL,
  host TEXT NOT NULL,
  version TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'OFFLINE' CHECK (status IN ('ONLINE','OFFLINE','BUSY','ERROR','DISABLED')),
  last_heartbeat_at TIMESTAMPTZ,
  capabilities JSONB NOT NULL DEFAULT '{}',
  metadata JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (organization_id, name)
);
CREATE INDEX IF NOT EXISTS idx_analytics_workers_heartbeat ON analytics_workers(last_heartbeat_at);

CREATE TABLE IF NOT EXISTS worker_camera_assignments (
  id BIGSERIAL PRIMARY KEY,
  worker_id BIGINT NOT NULL REFERENCES analytics_workers(id) ON DELETE CASCADE,
  camera_id BIGINT NOT NULL REFERENCES cameras(id) ON DELETE CASCADE,
  is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
  config_version INTEGER NOT NULL DEFAULT 1,
  assigned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (camera_id)
);

CREATE TABLE IF NOT EXISTS event_transcripts (
  id BIGSERIAL PRIMARY KEY,
  event_id BIGINT NOT NULL REFERENCES analytics_events(id) ON DELETE CASCADE,
  started_at TIMESTAMPTZ NOT NULL,
  finished_at TIMESTAMPTZ,
  speaker TEXT NOT NULL DEFAULT 'UNKNOWN' CHECK (speaker IN ('CASHIER','CUSTOMER','UNKNOWN')),
  text TEXT NOT NULL,
  language TEXT,
  confidence DOUBLE PRECISION CHECK (confidence BETWEEN 0 AND 1),
  words JSONB,
  metadata JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_event_transcripts_event ON event_transcripts(event_id, started_at);

CREATE TABLE IF NOT EXISTS audit_logs (
  id BIGSERIAL PRIMARY KEY,
  organization_id BIGINT REFERENCES organizations(id) ON DELETE SET NULL,
  user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  action TEXT NOT NULL,
  entity_type TEXT NOT NULL,
  entity_id TEXT,
  old_value JSONB,
  new_value JSONB,
  ip_address TEXT,
  user_agent TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_audit_logs_org_created ON audit_logs(organization_id, created_at DESC);

ALTER TABLE analytics_events ADD COLUMN IF NOT EXISTS processing_session_id BIGINT REFERENCES processing_sessions(id) ON DELETE SET NULL;
ALTER TABLE analytics_events ADD COLUMN IF NOT EXISTS worker_id BIGINT REFERENCES analytics_workers(id) ON DELETE SET NULL;
ALTER TABLE analytics_events ADD COLUMN IF NOT EXISTS deduplication_key TEXT;
ALTER TABLE analytics_events ADD COLUMN IF NOT EXISTS received_at TIMESTAMPTZ NOT NULL DEFAULT now();
CREATE UNIQUE INDEX IF NOT EXISTS uq_analytics_events_deduplication_key ON analytics_events(deduplication_key) WHERE deduplication_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_analytics_events_org_created ON analytics_events(organization_id, created_at DESC);

ALTER TABLE processing_sessions ADD COLUMN IF NOT EXISTS worker_id BIGINT REFERENCES analytics_workers(id) ON DELETE SET NULL;
ALTER TABLE processing_sessions ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}';
ALTER TABLE camera_metrics ADD COLUMN IF NOT EXISTS worker_id BIGINT REFERENCES analytics_workers(id) ON DELETE SET NULL;
ALTER TABLE event_evidence ADD COLUMN IF NOT EXISTS public_url TEXT;
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS attempts INTEGER NOT NULL DEFAULT 0;
ALTER TABLE external_events ADD COLUMN IF NOT EXISTS organization_id BIGINT REFERENCES organizations(id) ON DELETE CASCADE;
ALTER TABLE external_events ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE external_events ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

COMMIT;
