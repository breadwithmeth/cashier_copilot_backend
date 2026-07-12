BEGIN;

ALTER TABLE event_transcripts ALTER COLUMN event_id DROP NOT NULL;
ALTER TABLE event_transcripts ADD COLUMN IF NOT EXISTS organization_id BIGINT REFERENCES organizations(id) ON DELETE CASCADE;
ALTER TABLE event_transcripts ADD COLUMN IF NOT EXISTS store_id BIGINT REFERENCES stores(id) ON DELETE SET NULL;
ALTER TABLE event_transcripts ADD COLUMN IF NOT EXISTS workplace_id BIGINT REFERENCES workplaces(id) ON DELETE SET NULL;
ALTER TABLE event_transcripts ADD COLUMN IF NOT EXISTS camera_id BIGINT REFERENCES cameras(id) ON DELETE SET NULL;
ALTER TABLE event_transcripts ADD COLUMN IF NOT EXISTS receipt_id BIGINT REFERENCES receipts(id) ON DELETE SET NULL;
ALTER TABLE event_transcripts ADD COLUMN IF NOT EXISTS sale_session_id BIGINT REFERENCES sale_sessions(id) ON DELETE SET NULL;
ALTER TABLE event_transcripts ADD COLUMN IF NOT EXISTS external_transcript_id TEXT;
ALTER TABLE event_transcripts ADD COLUMN IF NOT EXISTS source_service TEXT;
ALTER TABLE event_transcripts ADD COLUMN IF NOT EXISTS audio_url TEXT;

UPDATE event_transcripts t
SET
  organization_id = e.organization_id,
  store_id = e.store_id,
  workplace_id = e.workplace_id,
  camera_id = e.camera_id,
  receipt_id = e.receipt_id,
  sale_session_id = e.sale_session_id
FROM analytics_events e
WHERE t.event_id = e.id
  AND t.organization_id IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_event_transcripts_external
  ON event_transcripts(source_service, external_transcript_id)
  WHERE source_service IS NOT NULL AND external_transcript_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_event_transcripts_receipt ON event_transcripts(receipt_id, started_at);
CREATE INDEX IF NOT EXISTS idx_event_transcripts_sale_session ON event_transcripts(sale_session_id, started_at);
CREATE INDEX IF NOT EXISTS idx_event_transcripts_workplace ON event_transcripts(workplace_id, started_at);
CREATE INDEX IF NOT EXISTS idx_event_transcripts_camera ON event_transcripts(camera_id, started_at);

COMMIT;
