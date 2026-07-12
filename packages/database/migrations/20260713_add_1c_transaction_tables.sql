BEGIN;

CREATE TABLE IF NOT EXISTS product_scans (
  id BIGSERIAL PRIMARY KEY,
  organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  store_id BIGINT NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
  workplace_id BIGINT NOT NULL REFERENCES workplaces(id) ON DELETE CASCADE,
  external_scan_id TEXT NOT NULL,
  external_receipt_id TEXT,
  barcode TEXT NOT NULL,
  product_name TEXT,
  quantity NUMERIC(18, 6),
  price NUMERIC(18, 2),
  currency TEXT,
  occurred_at TIMESTAMPTZ NOT NULL,
  received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  payload JSONB NOT NULL DEFAULT '{}',
  UNIQUE (organization_id, external_scan_id)
);

CREATE INDEX IF NOT EXISTS idx_product_scans_workplace_time ON product_scans(workplace_id, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_product_scans_receipt ON product_scans(external_receipt_id);

CREATE TABLE IF NOT EXISTS receipts (
  id BIGSERIAL PRIMARY KEY,
  organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  store_id BIGINT NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
  workplace_id BIGINT NOT NULL REFERENCES workplaces(id) ON DELETE CASCADE,
  external_receipt_id TEXT NOT NULL,
  external_order_id TEXT,
  cashier_external_id TEXT,
  payment_method TEXT,
  occurred_at TIMESTAMPTZ NOT NULL,
  received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  items JSONB NOT NULL DEFAULT '[]',
  totals JSONB NOT NULL DEFAULT '{}',
  payload JSONB NOT NULL DEFAULT '{}',
  UNIQUE (organization_id, external_receipt_id)
);

CREATE INDEX IF NOT EXISTS idx_receipts_workplace_time ON receipts(workplace_id, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_receipts_order ON receipts(external_order_id);

COMMIT;
