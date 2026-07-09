# Cashier Copilot Backend

Backend service for cashier video, speech, and POS analytics. The service receives POS events from 1C terminals, polls AI events written to PostgreSQL by Python workers, runs cashier state and rule logic, creates video export tasks, and streams alerts to operator and cashier UIs over WebSocket.

## Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Project Structure](#project-structure)
- [Requirements](#requirements)
- [Configuration](#configuration)
- [Running Locally](#running-locally)
- [Authentication](#authentication)
- [Verification Commands](#verification-commands)
- [Database Schema](#database-schema)
- [Event Flow](#event-flow)
- [Finite State Machine](#finite-state-machine)
- [Rule Engine](#rule-engine)
- [AI Co-Pilot](#ai-co-pilot)
- [REST API](#rest-api)
- [WebSocket API](#websocket-api)
- [PostgreSQL Task Queue](#postgresql-task-queue)
- [Operational Notes](#operational-notes)
- [Known Limitations](#known-limitations)

## Overview

The service is the central coordinator for a cashier control system:

- Receives POS events from 1C cash registers through HTTP.
- Stores POS events in PostgreSQL.
- Polls `cv_events` produced by video analytics workers.
- Polls `speech_transcripts` produced by STT workers.
- Maintains an in-memory finite state machine per POS terminal.
- Detects violations by correlating POS, CV, and speech events in time windows.
- Creates video export tasks in PostgreSQL for Python workers.
- Tracks completed and failed video export tasks.
- Sends operator alerts and cashier upsell prompts through WebSocket.

PostgreSQL is used both as persistent storage and as the integration bus between the Go backend and Python workers.

## Architecture

Runtime components:

- **1C POS terminals** send transaction events to `POST /api/v1/pos/event`.
- **Python video analytics workers** insert object/action detections into `cv_events`.
- **Python STT workers** insert speech transcripts into `speech_transcripts`.
- **Go backend pollers** read new CV/STT records and completed video tasks.
- **Rule Engine** creates rows in `violations` and `tasks`.
- **Python video export worker** reads pending rows from `tasks`, exports clips, and updates task status.
- **Operator UI** connects to `GET /ws/operator`.
- **Cashier UI** connects to `GET /ws/cashier?pos_id=...`.

Main packages:

- `cmd/server`: application entry point.
- `internal/config`: environment-based configuration.
- `internal/model`: shared domain and API models.
- `internal/repository`: PostgreSQL access layer.
- `internal/service`: FSM, pollers, rule engine, WebSocket hub, AI Co-Pilot.
- `internal/handler`: HTTP and WebSocket handlers.

## Project Structure

```text
.
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── handler/
│   │   ├── auth_handler.go
│   │   ├── auth_middleware.go
│   │   ├── camera_handler.go
│   │   ├── pos_handler.go
│   │   ├── user_handler.go
│   │   ├── violation_handler.go
│   │   └── ws_handler.go
│   ├── model/
│   │   └── models.go
│   ├── repository/
│   │   ├── camera_repo.go
│   │   ├── cv_event_repo.go
│   │   ├── db.go
│   │   ├── pos_event_repo.go
│   │   ├── speech_repo.go
│   │   ├── task_repo.go
│   │   ├── user_repo.go
│   │   ├── upsell_repo.go
│   │   └── violation_repo.go
│   └── service/
│       ├── auth.go
│       ├── copilot.go
│       ├── fsm.go
│       ├── hub.go
│       ├── poller.go
│       └── rule_engine.go
├── .env
├── .gitignore
├── go.mod
├── go.sum
└── README.md
```

## Requirements

- Go 1.21 or newer.
- PostgreSQL 15 or newer.
- Network access from the backend host to PostgreSQL.
- Python workers for CV/STT/video export if running the complete system.

Go dependencies:

- `github.com/go-chi/chi/v5`
- `github.com/gorilla/websocket`
- `github.com/jackc/pgx/v5`
- `github.com/rs/cors`
- `golang.org/x/crypto/bcrypt`

## Configuration

Configuration is read from environment variables. A local `.env` file is present in the project root and is ignored by git.

Required:

| Variable | Description |
| --- | --- |
| `DATABASE_URL` | PostgreSQL connection string used by `pgxpool`. |
| `JWT_SECRET` | HMAC secret used to sign access tokens. |
| `POS_API_KEY` | API key required for `POST /api/v1/pos/event`. |
| `ANALYTICS_API_KEY` | API key for analytics-service callbacks. Defaults to `POS_API_KEY` if omitted. |

Optional:

| Variable | Default | Description |
| --- | ---: | --- |
| `SERVER_PORT` | `8080` | HTTP listen port. |
| `POLL_INTERVAL_CV_MS` | `500` | Poll interval for `cv_events` and `speech_transcripts`. |
| `POLL_INTERVAL_TASKS_MS` | `2000` | Poll interval for completed or failed video tasks. |
| `CONFIDENCE_THRESHOLD` | `0.75` | Minimum confidence for operator alerts and video export tasks. |
| `MAX_DB_CONNS` | `20` | Maximum PostgreSQL pool connections. |
| `ACCESS_TOKEN_TTL_MINUTES` | `480` | Access token lifetime. |
| `BOOTSTRAP_ADMIN_USERNAME` | empty | Creates an admin user on startup when username does not exist. |
| `BOOTSTRAP_ADMIN_PASSWORD` | empty | Password for the bootstrap admin user. |

The `.env` file may also contain split DB variables such as `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, and `DB_PASSWORD`, but the Go service currently uses `DATABASE_URL`.

Do not commit `.env` or credentials.

## Running Locally

From the project root:

```bash
set -a; source .env; set +a
go run ./cmd/server
```

If Go cannot write to the default build cache:

```bash
set -a; source .env; set +a
GOCACHE=/private/tmp/cashier-go-build go run ./cmd/server
```

Expected successful startup logs:

```text
database connection pool established
running database migrations...
database migrations completed successfully
all background pollers started
HTTP server starting
```

Health check:

```bash
curl -i http://127.0.0.1:8080/health
```

## Authentication

The backend uses local PostgreSQL users and signed HMAC access tokens.

Roles:

| Role | Access |
| --- | --- |
| `admin` | Login, user management, camera management, violations, operator WS, cashier WS. |
| `operator` | Login, violations, camera list, operator WS, cashier WS. |
| `cashier` | Login and cashier WS for the user's assigned `pos_id`. |

Bootstrap:

- On startup, the backend creates `BOOTSTRAP_ADMIN_USERNAME` if it does not already exist.
- The password is read from `BOOTSTRAP_ADMIN_PASSWORD`.
- Existing users are not overwritten.

Login:

```bash
curl -sS -X POST http://127.0.0.1:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d "{\"username\":\"$BOOTSTRAP_ADMIN_USERNAME\",\"password\":\"$BOOTSTRAP_ADMIN_PASSWORD\"}"
```

Use the returned token on protected REST requests:

```bash
curl -i http://127.0.0.1:8080/api/v1/violations \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

Use the returned token on WebSocket URLs:

```text
ws://127.0.0.1:8080/ws/operator?token=ACCESS_TOKEN
ws://127.0.0.1:8080/ws/cashier?pos_id=pos-1&token=ACCESS_TOKEN
```

POS webhook authorization uses an API key:

```bash
curl -i -X POST http://127.0.0.1:8080/api/v1/pos/event \
  -H "X-API-Key: $POS_API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"pos_id":"pos-1","event_type":"receipt_opened","timestamp_ms":1760000000000,"details":{}}'
```

## Verification Commands

Run from the project root:

```bash
GOCACHE=/private/tmp/cashier-go-build go test ./...
GOCACHE=/private/tmp/cashier-go-build go vet ./...
GOCACHE=/private/tmp/cashier-go-build go test -race ./...
```

At the time of this documentation, the project has no automated tests. These commands verify compilation, static checks, and race-instrumented package builds.

## Database Schema

Migrations are executed automatically at startup by `repository.RunMigrations`.

### `cameras`

Stores camera configuration and POS mapping.

| Column | Type | Notes |
| --- | --- | --- |
| `id` | `varchar(50)` | Primary key. |
| `ip_address` | `varchar(45)` | Camera IP address. |
| `username` | `varchar(100)` | Camera username. |
| `password` | `varchar(100)` | Camera password. |
| `pos_id` | `varchar(50)` | POS terminal mapped to the camera. |
| `status` | `varchar(20)` | Defaults to `inactive`. |
| `roi_config` | `jsonb` | ROI polygon/config JSON. |
| `source_stream_url` | `text` | Raw camera/source stream, usually RTSP. |
| `analytics_stream_url` | `text` | Browser-consumable analytics output stream with overlays. |
| `analytics_stream_type` | `varchar(20)` | `hls`, `mjpeg`, `webrtc`, `http`, `rtsp`, etc. |
| `analytics_stream_status` | `varchar(20)` | `unknown`, `online`, `offline`, `failed`. |
| `analytics_stream_updated_at` | `timestamptz` | Last analytics stream update time. |
| `created_at` | `timestamptz` | Creation timestamp. |

### `pos_events`

Stores events received from POS terminals.

| Column | Type | Notes |
| --- | --- | --- |
| `id` | `bigserial` | Primary key. |
| `pos_id` | `varchar(50)` | POS terminal ID. |
| `event_type` | `varchar(50)` | POS event type. |
| `timestamp_ms` | `bigint` | Event time in Unix milliseconds. |
| `receipt_id` | `varchar(100)` | Receipt ID. |
| `details_jsonb` | `jsonb` | Event-specific payload. |

Index:

- `idx_pos_events_time_pos(timestamp_ms, pos_id)`

### `cv_events`

Stores video analytics events written by Python CV workers.

| Column | Type | Notes |
| --- | --- | --- |
| `id` | `bigserial` | Primary key. |
| `camera_id` | `varchar(50)` | Camera ID. |
| `event_type` | `varchar(50)` | Detection/action type. |
| `timestamp_ms` | `bigint` | Event time in Unix milliseconds. |
| `confidence` | `double precision` | Model confidence. |
| `model_name` | `varchar(100)` | Model name. |
| `weights_version` | `varchar(50)` | Weights version. |
| `inference_time_ms` | `integer` | Inference time. |
| `bbox_jsonb` | `jsonb` | Bounding box or detection metadata. |
| `snapshot_path` | `varchar(255)` | Snapshot path. |

Index:

- `idx_cv_events_time_cam(timestamp_ms, camera_id)`

### `speech_transcripts`

Stores speech transcripts written by Python STT workers.

| Column | Type | Notes |
| --- | --- | --- |
| `id` | `bigserial` | Primary key. |
| `pos_id` | `varchar(50)` | POS terminal ID. |
| `transcript` | `text` | Transcribed text. |
| `timestamp_ms` | `bigint` | Transcript start time in Unix milliseconds. |
| `duration_ms` | `integer` | Segment duration. |
| `confidence` | `double precision` | STT confidence. |
| `model_name` | `varchar(100)` | STT model name. |
| `weights_version` | `varchar(50)` | Model version. |

Index:

- `idx_speech_time_pos(timestamp_ms, pos_id)`

### `upsell_rules`

Stores cashier AI Co-Pilot upsell rules.

| Column | Type | Notes |
| --- | --- | --- |
| `id` | `serial` | Primary key. |
| `trigger_category` | `varchar(100)` | Product category prefix. |
| `required_keywords` | `text[]` | Speech keywords for completion. |
| `suggestion_text` | `text` | Prompt shown to cashier. |
| `suggestion_image_url` | `varchar(255)` | Optional image URL. |

### `violations`

Stores detected incidents.

| Column | Type | Notes |
| --- | --- | --- |
| `id` | `bigserial` | Primary key. |
| `pos_id` | `varchar(50)` | POS terminal ID. |
| `violation_type` | `varchar(50)` | Violation type. |
| `timestamp_ms` | `bigint` | Violation timestamp. |
| `proof_video_path` | `varchar(255)` | Exported video path. |
| `proof_image_path` | `varchar(255)` | Snapshot path. |
| `cv_event_id` | `bigint` | Optional FK to `cv_events`. |
| `pos_event_id` | `bigint` | Optional FK to `pos_events`. |
| `speech_transcript_id` | `bigint` | Optional FK to `speech_transcripts`. |
| `confidence_aggregate` | `double precision` | Rule confidence. |
| `status` | `varchar(20)` | `new` or `auto_filtered`. |

Index:

- `idx_violations_time(timestamp_ms)`

### `tasks`

PostgreSQL-backed queue for video export tasks.

| Column | Type | Notes |
| --- | --- | --- |
| `id` | `bigserial` | Primary key. |
| `task_type` | `varchar(50)` | Currently `video_export`. |
| `camera_id` | `varchar(50)` | Camera used for export. |
| `violation_id` | `bigint` | Optional FK to `violations`. |
| `payload` | `jsonb` | Task parameters. |
| `status` | `varchar(20)` | `pending`, `processing`, `completed`, `failed`. |
| `result_path` | `varchar(255)` | Exported video path. |
| `error_message` | `text` | Failure reason. |
| `created_at` | `timestamptz` | Creation timestamp. |
| `updated_at` | `timestamptz` | Update timestamp. |
| `processed_at` | `timestamptz` | Backend acknowledgement timestamp. |

Index:

- `idx_tasks_status(status)`

## Event Flow

### POS Events

1. 1C sends `POST /api/v1/pos/event`.
2. Handler validates and stores the event in `pos_events`.
3. FSM is updated.
4. Matching rules are triggered.
5. Co-Pilot may send upsell suggestions.
6. HTTP response returns accepted event ID and current FSM state.

### CV Events

1. Python CV worker inserts rows into `cv_events`.
2. Go poller fetches rows where `id > lastProcessedCvID`.
3. Camera is resolved to `pos_id`.
4. FSM is updated for customer presence events.
5. Relevant rules run for event types such as `item_in_bag` and `hand_to_drawer`.

### Speech Events

1. Python STT worker inserts rows into `speech_transcripts`.
2. Go poller fetches rows where `id > lastProcessedSpeechID`.
3. Co-Pilot checks active upsell suggestions against transcript keywords.

### Video Task Events

1. Rule Engine creates a `tasks` row with `status = 'pending'`.
2. Python video worker picks the pending task.
3. Worker updates the task to `completed` or `failed`.
4. Go task poller handles unacknowledged completed/failed tasks.
5. For completed tasks, Go updates `violations.proof_video_path`.
6. Go marks task `processed_at = CURRENT_TIMESTAMP`.
7. Operator UI receives a task status WebSocket update.

## Finite State Machine

States:

- `Idle`
- `CustomerDetected`
- `ReceiptOpened`
- `Scanning`
- `Payment`
- `ReceiptClosed`

POS transitions:

| Event | Allowed From | New State |
| --- | --- | --- |
| `receipt_opened` | `Idle`, `CustomerDetected` | `ReceiptOpened` |
| `item_scanned` | `ReceiptOpened`, `Scanning` | `Scanning` |
| `item_removed` | `Scanning` | `Scanning` |
| `receipt_cancelled` | `ReceiptOpened`, `Scanning` | `Idle` |
| `loyalty_card_applied` | Any current state | No state change |
| `payment_started` | `Scanning` | `Payment` |
| `receipt_closed` | `ReceiptOpened`, `Scanning`, `Payment` | `ReceiptClosed` |

CV transitions:

| Event | Allowed From | New State |
| --- | --- | --- |
| `customer_present` | `Idle` | `CustomerDetected` |
| `customer_left` | `CustomerDetected` | `Idle` |
| `customer_left` | `ReceiptClosed` | `Idle` |

Other CV events do not change FSM state directly.

## Rule Engine

The Rule Engine correlates events using PostgreSQL time-window queries.

### `unscanned_item`

Trigger:

- CV event `item_in_bag`.

Preconditions:

- CV confidence is at least `0.80`.
- FSM state for the POS is `Scanning`.

Check:

- Look for POS event `item_scanned` in `[Tcv - 3000ms, Tcv + 1500ms]`.

Violation:

- If no scan is found, create `unscanned_item`.

### `void_without_return`

Trigger:

- POS event `receipt_cancelled` or `item_removed`.

Check:

- Resolve camera by `pos_id`.
- Look for CV events `item_return` or `hand_to_scanner` in `[Tpos - 5s, Tpos + 10s]`.
- Check customer presence in the same window.

Violation:

- If no physical return is found, create `void_without_return`.
- Confidence is lower when customer is still present.

### `loyalty_card_abuse`

Trigger:

- POS event `loyalty_card_applied`.

Check:

- Look for CV event `phone_scanned_by_cashier` in `[Tpos - 2s, Tpos + 2s]`.

Violation:

- If cashier phone scan is found, create `loyalty_card_abuse`.

### `age_verification_failed`

Trigger:

- POS event `item_scanned` with `details.age_restricted = true`.

Check:

- Wait 15 seconds.
- Look for CV event `document_presented` in `[Tpos - 5s, Tpos + 15s]`.
- Look for speech keywords in the same window:
  - `паспорт`
  - `документ`
  - `возраст`
  - `18`
  - `восемнадцать`
  - `лет`
  - `рождения`

Violation:

- If no document or speech confirmation is found, create `age_verification_failed`.

### `drawer_opened_without_sale`

Trigger:

- CV event `hand_to_drawer`.

Check:

- FSM state is `Idle` or `CustomerDetected`.

Violation:

- Create `drawer_opened_without_sale`.

### `no_cashier_on_sale`

Trigger:

- FSM transitions to `Scanning` or `Payment`.

Check:

- Resolve camera by `pos_id`.
- Find latest `no_cashier`.
- If latest `cashier_present` is newer than `no_cashier`, no violation.

Violation:

- If cashier absence is the latest presence state, create `no_cashier_on_sale`.

## AI Co-Pilot

The Co-Pilot helps cashiers offer upsells.

Flow:

1. POS sends `item_scanned`.
2. Backend parses `details.category`.
3. Backend queries `upsell_rules` with category prefix matching.
4. First matching rule creates an `UpsellCard`.
5. Card is sent to cashier UI over WebSocket.
6. Active suggestion is tracked by `receipt_id`.
7. Speech poller checks transcripts against the rule's `required_keywords`.
8. On keyword match, backend sends an `upsell_status` update with `completed`.
9. On `receipt_closed`, backend clears active tracking for the receipt.

Example `upsell_rules` row:

```sql
INSERT INTO upsell_rules (
  trigger_category,
  required_keywords,
  suggestion_text,
  suggestion_image_url
) VALUES (
  'Алкоголь/Пиво',
  ARRAY['сухарики', 'рыба', 'закуска'],
  'Предложите сухарики или рыбу к пиву',
  ''
);
```

## REST API

Base path:

```text
/api/v1
```

### `POST /api/v1/pos/event`

Receives POS events from 1C.

Authentication:

- Requires `X-API-Key: <POS_API_KEY>`.

Request:

```json
{
  "pos_event_id": "external-event-id",
  "pos_id": "pos-1",
  "receipt_id": "receipt-1001",
  "event_type": "item_scanned",
  "timestamp_ms": 1760000000000,
  "details": {
    "sku": "123456",
    "item_name": "Product name",
    "category": "Category/Subcategory",
    "price": 1000,
    "quantity": 1,
    "age_restricted": false
  }
}
```

Required fields:

- `pos_id`
- `event_type`
- `timestamp_ms`

Supported POS event types:

- `receipt_opened`
- `item_scanned`
- `item_removed`
- `receipt_cancelled`
- `loyalty_card_applied`
- `payment_started`
- `receipt_closed`

Response `201 Created`:

```json
{
  "fsm_state": "Scanning",
  "id": 1,
  "status": "accepted"
}
```

Example:

```bash
curl -i -X POST http://127.0.0.1:8080/api/v1/pos/event \
  -H 'Content-Type: application/json' \
  -d '{
    "pos_id": "pos-1",
    "receipt_id": "r-1",
    "event_type": "receipt_opened",
    "timestamp_ms": 1760000000000,
    "details": {}
  }'
```

### `GET /api/v1/violations`

Returns paginated violations.

Authentication:

- Requires `Authorization: Bearer <token>`.
- Allowed roles: `admin`, `operator`.

Query parameters:

| Parameter | Description |
| --- | --- |
| `pos_id` | Filter by POS terminal. |
| `type` | Filter by violation type. |
| `status` | Filter by status. |
| `from_ts` | Minimum `timestamp_ms`. |
| `to_ts` | Maximum `timestamp_ms`. |
| `limit` | Page size, default `50`, max `200`. |
| `offset` | Page offset, default `0`. |

Response:

```json
{
  "data": [],
  "total": 0,
  "limit": 50,
  "offset": 0
}
```

Example:

```bash
curl -i 'http://127.0.0.1:8080/api/v1/violations?limit=5'
```

### `POST /api/v1/cameras`

Creates a camera configuration.

Authentication:

- Requires `Authorization: Bearer <token>`.
- Allowed role: `admin`.

Request:

```json
{
  "id": "cam-1",
  "ip_address": "192.168.1.10",
  "username": "admin",
  "password": "password",
  "pos_id": "pos-1",
  "status": "active",
  "roi_config": {
    "bag_zone": [[0, 0], [100, 0], [100, 100], [0, 100]]
  }
}
```

Required fields:

- `id`
- `ip_address`
- `pos_id`

Defaults:

- `status = "active"` when omitted.
- `roi_config = {}` when omitted.

Response `201 Created`:

```json
{
  "id": "cam-1",
  "ip_address": "192.168.1.10",
  "username": "admin",
  "pos_id": "pos-1",
  "status": "active",
  "roi_config": {},
  "created_at": "0001-01-01T00:00:00Z"
}
```

### `GET /api/v1/cameras`

Returns all configured cameras.

Authentication:

- Requires `Authorization: Bearer <token>`.
- Allowed roles: `admin`, `operator`.

Example:

```bash
curl -i http://127.0.0.1:8080/api/v1/cameras
```

Response:

```json
[]
```

### `GET /api/v1/cameras/{id}/streams`

Returns stream metadata for a camera.

Authentication:

- Requires `Authorization: Bearer <token>`.
- Allowed roles: `admin`, `operator`.

Response:

```json
{
  "camera_id": "cam-1",
  "pos_id": "pos-1",
  "analytics_stream_url": "http://analytics.local/streams/cam-1/index.m3u8",
  "analytics_stream_type": "hls",
  "analytics_stream_status": "online",
  "analytics_stream_updated_at": "2026-07-09T13:00:00Z",
  "roi_config": {},
  "overlay_enabled": true
}
```

For `admin` users, `source_stream_url` may also be returned. Operators should use `analytics_stream_url`; browsers generally cannot play raw RTSP.

### `PATCH /api/v1/cameras/{id}/streams`

Admin endpoint to update stream metadata manually.

Authentication:

- Requires `Authorization: Bearer <token>`.
- Allowed role: `admin`.

Request:

```json
{
  "source_stream_url": "rtsp://camera/source",
  "analytics_stream_url": "http://analytics.local/streams/cam-1/index.m3u8",
  "analytics_stream_type": "hls",
  "analytics_stream_status": "online"
}
```

### `POST /api/v1/analytics/cameras/{id}/stream`

Service-to-service callback used by the analytics service to publish its browser-ready output stream with overlays.

Authentication:

- Requires `X-API-Key: <ANALYTICS_API_KEY>`.

Request:

```json
{
  "analytics_stream_url": "http://analytics.local/streams/cam-1/index.m3u8",
  "analytics_stream_type": "hls",
  "analytics_stream_status": "online"
}
```

## WebSocket API

### `GET /ws/operator`

Operator dashboard stream.

Authentication:

- Use `?token=<access_token>`.
- Allowed roles: `admin`, `operator`.

Message type: `violation_alert`

```json
{
  "type": "violation_alert",
  "payload": {
    "violation": {
      "id": 1,
      "pos_id": "pos-1",
      "violation_type": "unscanned_item",
      "timestamp_ms": 1760000000000,
      "confidence_aggregate": 0.92,
      "status": "new"
    }
  }
}
```

Message type: `task_status`

```json
{
  "type": "task_status",
  "payload": {
    "task_id": 10,
    "violation_id": 1,
    "status": "completed",
    "video_path": "/path/to/video.mp4"
  }
}
```

### `GET /ws/cashier?pos_id=XXX`

Cashier terminal stream for a specific POS.

Authentication:

- Use `?token=<access_token>`.
- Allowed roles: `admin`, `operator`, `cashier`.
- Cashier users can connect only to their assigned `pos_id`.

Message type: `upsell_card`

```json
{
  "type": "upsell_card",
  "payload": {
    "pos_id": "pos-1",
    "receipt_id": "r-1",
    "trigger_item": "Beer",
    "suggestion_text": "Offer snacks",
    "suggestion_image": "",
    "status": "pending"
  }
}
```

Message type: `upsell_status`

```json
{
  "type": "upsell_status",
  "payload": {
    "pos_id": "pos-1",
    "receipt_id": "r-1",
    "status": "completed"
  }
}
```

## PostgreSQL Task Queue

The `tasks` table is the queue between the Go backend and Python video export workers.

Go creates a task:

```json
{
  "task_type": "video_export",
  "camera_id": "cam-1",
  "violation_id": 1,
  "payload": {
    "start_timestamp_ms": 1759999990000,
    "end_timestamp_ms": 1760000010000
  },
  "status": "pending"
}
```

Expected Python worker lifecycle:

1. Select pending task.
2. Mark as `processing`.
3. Export MP4 from RAM/video buffer.
4. On success:

```sql
UPDATE tasks
SET status = 'completed',
    result_path = '/path/to/export.mp4',
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1;
```

5. On failure:

```sql
UPDATE tasks
SET status = 'failed',
    error_message = 'reason',
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1;
```

The Go backend polls:

```sql
SELECT *
FROM tasks
WHERE status = 'completed'
  AND processed_at IS NULL;
```

and:

```sql
SELECT *
FROM tasks
WHERE status = 'failed'
  AND processed_at IS NULL;
```

After handling, Go sets:

```sql
UPDATE tasks
SET processed_at = CURRENT_TIMESTAMP
WHERE id = $1;
```

## Operational Notes

- The application logs JSON through `log/slog`.
- HTTP request logging is enabled through `chi` middleware.
- Graceful shutdown listens for `SIGINT` and `SIGTERM`.
- On shutdown, the root context is cancelled, the HTTP server is stopped, and the PostgreSQL pool is closed.
- CORS currently allows all origins for development.
- WebSocket origin checks currently allow all origins for development.
- The service does not load `.env` automatically. Source it in the shell before running the server.

## Known Limitations

- Authentication is local-only: there are no refresh tokens, password reset flow, or external identity provider integration yet.
- CORS and WebSocket origin policy are development-friendly and should be restricted in production.
- There are no automated unit or integration tests yet.
- Poller offsets are in memory. After restart, old `cv_events` and `speech_transcripts` may be reprocessed unless IDs are persisted or events are otherwise deduplicated.
- `tasks.updated_at` is not automatically updated by a database trigger.
- `cameras.password` is stored as plain text.
- `POST /api/v1/cameras` uses insert-only semantics and will fail on duplicate camera IDs.
- Shutdown can log `context canceled` from pollers as errors even though shutdown succeeds.
