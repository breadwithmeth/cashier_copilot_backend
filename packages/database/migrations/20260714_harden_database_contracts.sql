BEGIN;

ALTER TABLE product_scans ADD COLUMN IF NOT EXISTS receipt_id BIGINT REFERENCES receipts(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_product_scans_receipt_id ON product_scans(receipt_id);
ALTER TABLE receipts ADD COLUMN IF NOT EXISTS payment_method TEXT;

UPDATE workplaces
SET external_id = 'legacy-workplace-' || id::text
WHERE external_id IS NULL OR btrim(external_id) = '';

INSERT INTO workplaces (store_id, name, workplace_type, external_id, is_active)
SELECT c.store_id, 'Legacy workplace for ' || c.name, 'checkout', 'legacy-camera-' || c.id::text, TRUE
FROM cameras c
WHERE c.workplace_id IS NULL
  AND NOT EXISTS (
    SELECT 1
    FROM workplaces w
    WHERE w.store_id = c.store_id
      AND w.external_id = 'legacy-camera-' || c.id::text
  );

UPDATE cameras c
SET workplace_id = w.id
FROM workplaces w
WHERE c.workplace_id IS NULL
  AND w.store_id = c.store_id
  AND w.external_id = 'legacy-camera-' || c.id::text;

ALTER TABLE workplaces ALTER COLUMN external_id SET NOT NULL;
ALTER TABLE cameras ALTER COLUMN workplace_id SET NOT NULL;

UPDATE analytics_events
SET status = CASE
  WHEN status IN ('open', 'new') THEN 'NEW'
  WHEN status IN ('in_review') THEN 'IN_REVIEW'
  WHEN status IN ('confirmed') THEN 'CONFIRMED'
  WHEN status IN ('false_positive') THEN 'FALSE_POSITIVE'
  WHEN status IN ('resolved') THEN 'RESOLVED'
  ELSE status
END;

UPDATE analytics_events
SET severity = upper(severity)
WHERE severity IN ('info', 'warning', 'critical', 'medium');

UPDATE event_types
SET default_severity = upper(default_severity)
WHERE default_severity IN ('info', 'warning', 'critical', 'medium');

UPDATE notifications
SET status = upper(status)
WHERE status IN ('pending', 'sent', 'failed');

ALTER TABLE analytics_events ALTER COLUMN status SET DEFAULT 'NEW';
ALTER TABLE analytics_events ALTER COLUMN severity SET DEFAULT 'INFO';
ALTER TABLE event_types ALTER COLUMN default_severity SET DEFAULT 'INFO';
ALTER TABLE notifications ALTER COLUMN status SET DEFAULT 'PENDING';

ALTER TABLE workplaces DROP CONSTRAINT IF EXISTS uq_workplaces_id_store_id;
ALTER TABLE workplaces ADD CONSTRAINT uq_workplaces_id_store_id UNIQUE (id, store_id);

ALTER TABLE cameras DROP CONSTRAINT IF EXISTS fk_cameras_workplace_store;
ALTER TABLE cameras ADD CONSTRAINT fk_cameras_workplace_store
  FOREIGN KEY (workplace_id, store_id) REFERENCES workplaces(id, store_id);

ALTER TABLE product_scans DROP CONSTRAINT IF EXISTS fk_product_scans_workplace_store;
ALTER TABLE product_scans ADD CONSTRAINT fk_product_scans_workplace_store
  FOREIGN KEY (workplace_id, store_id) REFERENCES workplaces(id, store_id);

ALTER TABLE receipts DROP CONSTRAINT IF EXISTS fk_receipts_workplace_store;
ALTER TABLE receipts ADD CONSTRAINT fk_receipts_workplace_store
  FOREIGN KEY (workplace_id, store_id) REFERENCES workplaces(id, store_id);

UPDATE product_scans ps
SET receipt_id = r.id
FROM receipts r
WHERE ps.receipt_id IS NULL
  AND ps.organization_id = r.organization_id
  AND ps.external_receipt_id = r.external_receipt_id;

ALTER TABLE analytics_events DROP CONSTRAINT IF EXISTS chk_analytics_events_status;
ALTER TABLE analytics_events ADD CONSTRAINT chk_analytics_events_status
  CHECK (status IN ('NEW','IN_REVIEW','CONFIRMED','FALSE_POSITIVE','RESOLVED'));

ALTER TABLE analytics_events DROP CONSTRAINT IF EXISTS chk_analytics_events_severity;
ALTER TABLE analytics_events ADD CONSTRAINT chk_analytics_events_severity
  CHECK (severity IN ('INFO','WARNING','CRITICAL','MEDIUM'));

ALTER TABLE event_types DROP CONSTRAINT IF EXISTS chk_event_types_default_severity;
ALTER TABLE event_types ADD CONSTRAINT chk_event_types_default_severity
  CHECK (default_severity IN ('INFO','WARNING','CRITICAL','MEDIUM'));

ALTER TABLE notifications DROP CONSTRAINT IF EXISTS chk_notifications_status;
ALTER TABLE notifications ADD CONSTRAINT chk_notifications_status
  CHECK (status IN ('PENDING','SENT','FAILED'));

ALTER TABLE camera_streams DROP CONSTRAINT IF EXISTS chk_camera_streams_stream_type;
ALTER TABLE camera_streams ADD CONSTRAINT chk_camera_streams_stream_type
  CHECK (stream_type IN ('RTSP_VIDEO','RTSP_AUDIO','VIDEO','AUDIO'));

ALTER TABLE camera_streams DROP CONSTRAINT IF EXISTS chk_camera_streams_transport;
ALTER TABLE camera_streams ADD CONSTRAINT chk_camera_streams_transport
  CHECK (transport IN ('tcp','udp','http','https'));

UPDATE receipts
SET payment_method = upper(payment_method)
WHERE payment_method IS NOT NULL;

ALTER TABLE receipts DROP CONSTRAINT IF EXISTS chk_receipts_payment_method;
ALTER TABLE receipts ADD CONSTRAINT chk_receipts_payment_method
  CHECK (payment_method IS NULL OR payment_method IN ('CASH','CARD','BONUS','MIXED'));

COMMIT;
