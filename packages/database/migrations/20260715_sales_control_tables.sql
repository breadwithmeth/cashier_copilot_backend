BEGIN;

ALTER TABLE receipts ADD COLUMN IF NOT EXISTS employee_id BIGINT REFERENCES employees(id) ON DELETE SET NULL;
ALTER TABLE receipts ADD COLUMN IF NOT EXISTS operation_type TEXT NOT NULL DEFAULT 'SALE';
ALTER TABLE receipts ADD COLUMN IF NOT EXISTS receipt_status TEXT NOT NULL DEFAULT 'CLOSED';
ALTER TABLE receipts ADD COLUMN IF NOT EXISTS receipt_total NUMERIC(18, 2);
ALTER TABLE receipts ADD COLUMN IF NOT EXISTS paid_amount NUMERIC(18, 2);
ALTER TABLE receipts ADD COLUMN IF NOT EXISTS change_amount NUMERIC(18, 2);
ALTER TABLE receipts ADD COLUMN IF NOT EXISTS bonus_amount NUMERIC(18, 2);
ALTER TABLE receipts ADD COLUMN IF NOT EXISTS discount_amount NUMERIC(18, 2);
ALTER TABLE receipts ADD COLUMN IF NOT EXISTS printed_at TIMESTAMPTZ;
ALTER TABLE receipts ADD COLUMN IF NOT EXISTS closed_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_receipts_employee_time ON receipts(employee_id, occurred_at DESC);

CREATE TABLE IF NOT EXISTS receipt_items (
  id BIGSERIAL PRIMARY KEY,
  receipt_id BIGINT NOT NULL REFERENCES receipts(id) ON DELETE CASCADE,
  organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  store_id BIGINT NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
  workplace_id BIGINT NOT NULL REFERENCES workplaces(id) ON DELETE CASCADE,
  line_number INTEGER NOT NULL,
  external_product_id TEXT,
  barcode TEXT,
  product_name TEXT NOT NULL,
  quantity NUMERIC(18, 6) NOT NULL,
  price NUMERIC(18, 2) NOT NULL,
  line_total NUMERIC(18, 2) NOT NULL,
  discount_amount NUMERIC(18, 2),
  is_container BOOLEAN NOT NULL DEFAULT FALSE,
  container_type TEXT,
  payload JSONB NOT NULL DEFAULT '{}',
  UNIQUE (receipt_id, line_number)
);
CREATE INDEX IF NOT EXISTS idx_receipt_items_barcode ON receipt_items(barcode);
CREATE INDEX IF NOT EXISTS idx_receipt_items_product ON receipt_items(organization_id, external_product_id);

CREATE TABLE IF NOT EXISTS sale_sessions (
  id BIGSERIAL PRIMARY KEY,
  organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  store_id BIGINT NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
  workplace_id BIGINT NOT NULL REFERENCES workplaces(id) ON DELETE CASCADE,
  receipt_id BIGINT UNIQUE REFERENCES receipts(id) ON DELETE SET NULL,
  employee_id BIGINT REFERENCES employees(id) ON DELETE SET NULL,
  operation_type TEXT NOT NULL DEFAULT 'SALE',
  started_at TIMESTAMPTZ NOT NULL,
  finished_at TIMESTAMPTZ,
  customer_present BOOLEAN,
  customer_present_confidence DOUBLE PRECISION,
  service_score DOUBLE PRECISION,
  status TEXT NOT NULL DEFAULT 'OPEN',
  metadata JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_sale_sessions_workplace_time ON sale_sessions(workplace_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_sale_sessions_receipt ON sale_sessions(receipt_id);
CREATE INDEX IF NOT EXISTS idx_sale_sessions_employee_time ON sale_sessions(employee_id, started_at DESC);

CREATE TABLE IF NOT EXISTS violation_types (
  id BIGSERIAL PRIMARY KEY,
  organization_id BIGINT REFERENCES organizations(id) ON DELETE CASCADE,
  code TEXT NOT NULL,
  name TEXT NOT NULL,
  risk_level TEXT NOT NULL,
  employee_notification_text TEXT,
  visible_to_roles JSONB NOT NULL DEFAULT '[]',
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (organization_id, code)
);
CREATE INDEX IF NOT EXISTS idx_violation_types_risk ON violation_types(risk_level);

INSERT INTO violation_types (organization_id, code, name, risk_level, employee_notification_text, visible_to_roles)
SELECT NULL, v.code, v.name, v.risk_level, v.employee_notification_text, v.visible_to_roles::jsonb
FROM (VALUES
  ('ITEM_HANDED_NOT_SCANNED', 'Товар отдан, но не пробит', 'CRITICAL', 'Проверьте чек: товар передан, но не пробит', '["ORGANIZATION_ADMIN","MANAGER","OPERATOR"]'),
  ('SCAN_IMITATION', 'Имитация сканирования', 'CRITICAL', 'Сканер был поднесен, но товар не пикнут', '["ORGANIZATION_ADMIN","MANAGER","OPERATOR"]'),
  ('PAYMENT_MISMATCH', 'Несоответствие оплаты', 'CRITICAL', 'Проверьте способ оплаты и сумму чека', '["ORGANIZATION_ADMIN","MANAGER","OPERATOR"]'),
  ('STORNO_WITH_HANDOVER', 'Возврат/сторно при передаче товара', 'CRITICAL', 'Проверьте операцию: товар передан при отмене/сторно', '["ORGANIZATION_ADMIN","MANAGER","OPERATOR"]'),
  ('CONTAINER_NOT_SCANNED', 'Не пробита тара', 'WARNING', 'Проверьте тару: тара не пробита', '["ORGANIZATION_ADMIN","MANAGER"]'),
  ('RECEIPT_NOT_GIVEN', 'Не выдан чек', 'WARNING', 'Не забудьте выдать чек покупателю', '["ORGANIZATION_ADMIN","MANAGER"]'),
  ('BUSINESS_CARD_NOT_GIVEN', 'Не выдана визитка', 'WARNING', 'Не забудьте положить визитку', '["ORGANIZATION_ADMIN","MANAGER"]'),
  ('AMOUNT_NOT_SPOKEN', 'Не озвучена сумма', 'WARNING', 'Озвучьте сумму покупки покупателю', '["ORGANIZATION_ADMIN","MANAGER"]'),
  ('CHANGE_NOT_SPOKEN', 'Не озвучена сдача', 'WARNING', 'При наличной оплате необходимо озвучить сдачу', '["ORGANIZATION_ADMIN","MANAGER"]'),
  ('SERVICE_SEQUENCE_BROKEN', 'Нарушена последовательность обслуживания', 'INFO', 'Проверьте последовательность обслуживания', '["ORGANIZATION_ADMIN","MANAGER"]'),
  ('UPSELL_NOT_OFFERED', 'Не предложены сопутствующие товары', 'INFO', 'Предложите сопутствующие товары', '["ORGANIZATION_ADMIN","MANAGER"]'),
  ('NO_FAREWELL', 'Нет прощания', 'INFO', 'Не забудьте попрощаться с покупателем', '["ORGANIZATION_ADMIN","MANAGER"]'),
  ('PRODUCT_SCANNED_WITHOUT_CUSTOMER', 'Товар отсканирован без клиента', 'WARNING', 'Проверьте продажу: сканирование без клиента в зоне кассы', '["ORGANIZATION_ADMIN","MANAGER","OPERATOR"]')
) AS v(code, name, risk_level, employee_notification_text, visible_to_roles)
WHERE NOT EXISTS (
  SELECT 1
  FROM violation_types existing
  WHERE existing.organization_id IS NULL
    AND existing.code = v.code
);

ALTER TABLE analytics_events ADD COLUMN IF NOT EXISTS receipt_id BIGINT REFERENCES receipts(id) ON DELETE SET NULL;
ALTER TABLE analytics_events ADD COLUMN IF NOT EXISTS sale_session_id BIGINT REFERENCES sale_sessions(id) ON DELETE SET NULL;
ALTER TABLE analytics_events ADD COLUMN IF NOT EXISTS violation_type_id BIGINT REFERENCES violation_types(id) ON DELETE SET NULL;
ALTER TABLE analytics_events ADD COLUMN IF NOT EXISTS operation_type TEXT;
ALTER TABLE analytics_events ADD COLUMN IF NOT EXISTS risk_amount NUMERIC(18, 2);
CREATE INDEX IF NOT EXISTS idx_analytics_events_receipt ON analytics_events(receipt_id);
CREATE INDEX IF NOT EXISTS idx_analytics_events_sale_session ON analytics_events(sale_session_id);
CREATE INDEX IF NOT EXISTS idx_analytics_events_violation_type ON analytics_events(violation_type_id);

ALTER TABLE event_evidence ADD COLUMN IF NOT EXISTS camera_id BIGINT REFERENCES cameras(id) ON DELETE SET NULL;
ALTER TABLE event_evidence ADD COLUMN IF NOT EXISTS receipt_id BIGINT REFERENCES receipts(id) ON DELETE SET NULL;
ALTER TABLE event_evidence ADD COLUMN IF NOT EXISTS availability_status TEXT NOT NULL DEFAULT 'AVAILABLE';
ALTER TABLE event_evidence ADD COLUMN IF NOT EXISTS video_started_at TIMESTAMPTZ;
ALTER TABLE event_evidence ADD COLUMN IF NOT EXISTS video_finished_at TIMESTAMPTZ;
ALTER TABLE event_evidence ADD COLUMN IF NOT EXISTS pre_seconds INTEGER;
ALTER TABLE event_evidence ADD COLUMN IF NOT EXISTS post_seconds INTEGER;
CREATE INDEX IF NOT EXISTS idx_event_evidence_receipt ON event_evidence(receipt_id);
CREATE INDEX IF NOT EXISTS idx_event_evidence_camera ON event_evidence(camera_id, captured_at);

CREATE TABLE IF NOT EXISTS video_observations (
  id BIGSERIAL PRIMARY KEY,
  organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  store_id BIGINT NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
  workplace_id BIGINT NOT NULL REFERENCES workplaces(id) ON DELETE CASCADE,
  camera_id BIGINT REFERENCES cameras(id) ON DELETE SET NULL,
  receipt_id BIGINT REFERENCES receipts(id) ON DELETE SET NULL,
  sale_session_id BIGINT REFERENCES sale_sessions(id) ON DELETE SET NULL,
  evidence_id BIGINT REFERENCES event_evidence(id) ON DELETE SET NULL,
  observed_at TIMESTAMPTZ NOT NULL,
  observation_type TEXT NOT NULL,
  barcode TEXT,
  product_name TEXT,
  quantity NUMERIC(18, 6),
  confidence DOUBLE PRECISION,
  metadata JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_video_observations_workplace_time ON video_observations(workplace_id, observed_at DESC);
CREATE INDEX IF NOT EXISTS idx_video_observations_receipt ON video_observations(receipt_id);
CREATE INDEX IF NOT EXISTS idx_video_observations_sale_session ON video_observations(sale_session_id);
CREATE INDEX IF NOT EXISTS idx_video_observations_type_time ON video_observations(observation_type, observed_at DESC);

CREATE TABLE IF NOT EXISTS service_check_results (
  id BIGSERIAL PRIMARY KEY,
  organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  sale_session_id BIGINT NOT NULL REFERENCES sale_sessions(id) ON DELETE CASCADE,
  criterion_code TEXT NOT NULL,
  result TEXT NOT NULL,
  confidence DOUBLE PRECISION,
  evidence_id BIGINT REFERENCES event_evidence(id) ON DELETE SET NULL,
  comment TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (sale_session_id, criterion_code)
);
CREATE INDEX IF NOT EXISTS idx_service_checks_criterion ON service_check_results(organization_id, criterion_code);

CREATE TABLE IF NOT EXISTS integration_errors (
  id BIGSERIAL PRIMARY KEY,
  organization_id BIGINT REFERENCES organizations(id) ON DELETE CASCADE,
  store_id BIGINT REFERENCES stores(id) ON DELETE SET NULL,
  workplace_id BIGINT REFERENCES workplaces(id) ON DELETE SET NULL,
  source_system TEXT NOT NULL,
  entity_type TEXT NOT NULL,
  external_id TEXT,
  error_code TEXT NOT NULL,
  error_message TEXT NOT NULL,
  payload JSONB NOT NULL DEFAULT '{}',
  status TEXT NOT NULL DEFAULT 'OPEN',
  occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  resolved_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_integration_errors_status ON integration_errors(status, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_integration_errors_external ON integration_errors(source_system, entity_type, external_id);

CREATE TABLE IF NOT EXISTS receivings (
  id BIGSERIAL PRIMARY KEY,
  organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  store_id BIGINT NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
  workplace_id BIGINT REFERENCES workplaces(id) ON DELETE SET NULL,
  employee_id BIGINT REFERENCES employees(id) ON DELETE SET NULL,
  external_invoice_id TEXT NOT NULL,
  supplier_id TEXT,
  supplier_name TEXT,
  started_at TIMESTAMPTZ NOT NULL,
  finished_at TIMESTAMPTZ,
  status TEXT NOT NULL DEFAULT 'OPEN',
  payload JSONB NOT NULL DEFAULT '{}',
  UNIQUE (organization_id, external_invoice_id)
);
CREATE INDEX IF NOT EXISTS idx_receivings_store_time ON receivings(store_id, started_at DESC);

CREATE TABLE IF NOT EXISTS receiving_items (
  id BIGSERIAL PRIMARY KEY,
  receiving_id BIGINT NOT NULL REFERENCES receivings(id) ON DELETE CASCADE,
  organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  store_id BIGINT NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
  workplace_id BIGINT REFERENCES workplaces(id) ON DELETE SET NULL,
  external_product_id TEXT,
  barcode TEXT,
  product_name TEXT NOT NULL,
  expected_quantity NUMERIC(18, 6) NOT NULL,
  actual_quantity NUMERIC(18, 6),
  expiry_date DATE,
  package_damaged BOOLEAN,
  defect_detected BOOLEAN,
  discrepancy_reported BOOLEAN,
  payload JSONB NOT NULL DEFAULT '{}'
);
CREATE INDEX IF NOT EXISTS idx_receiving_items_receiving ON receiving_items(receiving_id);
CREATE INDEX IF NOT EXISTS idx_receiving_items_barcode ON receiving_items(barcode);

ALTER TABLE receipts DROP CONSTRAINT IF EXISTS chk_receipts_operation_type;
ALTER TABLE receipts ADD CONSTRAINT chk_receipts_operation_type
  CHECK (operation_type IN ('SALE','RETURN','CANCEL','STORNO'));

ALTER TABLE receipts DROP CONSTRAINT IF EXISTS chk_receipts_status;
ALTER TABLE receipts ADD CONSTRAINT chk_receipts_status
  CHECK (receipt_status IN ('OPEN','CLOSED','CANCELLED','RETURNED'));

ALTER TABLE sale_sessions DROP CONSTRAINT IF EXISTS chk_sale_sessions_status;
ALTER TABLE sale_sessions ADD CONSTRAINT chk_sale_sessions_status
  CHECK (status IN ('OPEN','CLOSED','REVIEW_REQUIRED','ERROR'));

ALTER TABLE service_check_results DROP CONSTRAINT IF EXISTS chk_service_check_results_result;
ALTER TABLE service_check_results ADD CONSTRAINT chk_service_check_results_result
  CHECK (result IN ('YES','NO','NOT_REQUIRED','UNKNOWN'));

COMMIT;
