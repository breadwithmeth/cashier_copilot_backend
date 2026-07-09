# Video Server Integration Guide

This document describes how the video analytics and video export server must interact with the Go backend.

The integration is asynchronous through PostgreSQL. The Go backend does not call the video server directly. The video server writes detection events into `cv_events` and reads video export jobs from `tasks`.

## Responsibilities

The video server has two independent responsibilities:

1. **Real-time video analytics**
   - Read camera streams.
   - Run object/action detection.
   - Insert detection events into `cv_events`.

2. **Video export worker**
   - Poll `tasks` for `pending` video export jobs.
   - Export clips from video buffer/storage.
   - Update task status to `completed` or `failed`.

3. **Frontend analytics stream**
   - Render zones, detections, tracks, and other overlays on top of the camera frame.
   - Serve the processed output in a browser-consumable format such as HLS, WebRTC, or MJPEG.
   - Publish the stream URL to the Go backend.

## Database Connection

Use the same PostgreSQL database as the Go backend.

Required connection settings:

| Setting | Description |
| --- | --- |
| Host | PostgreSQL host. |
| Port | Usually `5432`. |
| Database | Backend database name. |
| User | Backend DB user. |
| Password | Backend DB password. |
| SSL mode | Match deployment settings. Current local backend uses `sslmode=disable`. |

Do not hardcode credentials in source code. Use environment variables.

Example environment variables for the video server:

```bash
DB_HOST=...
DB_PORT=5432
DB_NAME=cashier_copilot
DB_USER=cashier_copilot
DB_PASSWORD=...
DB_SSLMODE=disable
```

## Time Convention

All timestamps exchanged with the backend must use Unix milliseconds:

```text
timestamp_ms = int(time.time() * 1000)
```

The same clock source should be used for:

- CV detection event time.
- Video buffer indexing.
- Video export `start_timestamp_ms` and `end_timestamp_ms`.

Clock drift between POS, backend, and video server will reduce rule accuracy. Use NTP on all hosts.

## Camera Mapping

The Go backend maps CV events to POS terminals through the `cameras` table:

```text
cv_events.camera_id -> cameras.id -> cameras.pos_id
```

The video server must write `camera_id` exactly as stored in `cameras.id`.

For frontend live view, the analytics service must publish its overlay output stream against the same `camera_id`.

## Publishing Overlay Stream URL

The frontend does not consume raw RTSP. The analytics service should expose its processed output with overlays as one of:

- HLS, for example `http://analytics.local/streams/cam-1/index.m3u8`.
- MJPEG, for example `http://analytics.local/streams/cam-1.mjpeg`.
- WebRTC/WHEP URL.

After the stream is ready, call the backend:

```http
POST /api/v1/analytics/cameras/{camera_id}/stream
X-API-Key: <ANALYTICS_API_KEY>
Content-Type: application/json
```

Request:

```json
{
  "analytics_stream_url": "http://analytics.local/streams/cam-1/index.m3u8",
  "analytics_stream_type": "hls",
  "analytics_stream_status": "online"
}
```

On failure/offline:

```json
{
  "analytics_stream_status": "offline"
}
```

The Go backend stores this in the `cameras` table. The frontend reads it from:

```http
GET /api/v1/cameras/{camera_id}/streams
Authorization: Bearer <access_token>
```

Before starting detection for a camera, ensure it exists in `GET /api/v1/cameras` or in the database table:

```sql
SELECT id, pos_id, status, roi_config
FROM cameras
WHERE status = 'active';
```

## Writing CV Events

The video analytics worker writes detection rows into `cv_events`.

Required insert:

```sql
INSERT INTO cv_events (
  camera_id,
  event_type,
  timestamp_ms,
  confidence,
  model_name,
  weights_version,
  inference_time_ms,
  bbox_jsonb,
  snapshot_path
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9
);
```

Column requirements:

| Column | Requirement |
| --- | --- |
| `camera_id` | Must match `cameras.id`. |
| `event_type` | Must be one of the agreed detection types. |
| `timestamp_ms` | Unix milliseconds of the observed event. |
| `confidence` | Float from `0.0` to `1.0`. |
| `model_name` | Model identifier, for example `yolov11-cashier`. |
| `weights_version` | Version/hash of model weights. |
| `inference_time_ms` | Inference duration in milliseconds. |
| `bbox_jsonb` | JSON with bbox, class, track ID, ROI, etc. |
| `snapshot_path` | Path or URL to snapshot image. Use empty string if unavailable. |

Example:

```sql
INSERT INTO cv_events (
  camera_id,
  event_type,
  timestamp_ms,
  confidence,
  model_name,
  weights_version,
  inference_time_ms,
  bbox_jsonb,
  snapshot_path
) VALUES (
  'cam-1',
  'item_in_bag',
  1760000000000,
  0.91,
  'yolov11-cashier',
  '2026-07-09-a',
  34,
  '{"bbox":[120,80,260,220],"track_id":"t-123","roi":"bag_zone"}',
  '/snapshots/cam-1/1760000000000.jpg'
);
```

## CV Event Types Expected by Backend

The Go Rule Engine currently reacts to these event types:

| Event Type | Meaning | Used For |
| --- | --- | --- |
| `item_in_bag` | Item moved into bagging zone. | `unscanned_item` rule. |
| `hand_to_drawer` | Cashier hand near cash drawer. | `drawer_opened_without_sale` rule. |
| `phone_scanned_by_cashier` | Cashier scans their own phone/QR. | `loyalty_card_abuse` rule. |
| `document_presented` | Customer document/passport visible. | `age_verification_failed` rule. |
| `item_return` | Item moved from bag/customer area back to cashier/scanner. | `void_without_return` rule. |
| `hand_to_scanner` | Hand/item returned toward scanner/cashier area. | `void_without_return` fallback. |
| `customer_present` | Customer detected in service zone. | FSM transition. |
| `customer_left` | Customer leaves service zone. | FSM transition. |
| `no_cashier` | Cashier absent from workplace. | `no_cashier_on_sale` rule. |
| `cashier_present` | Cashier present at workplace. | `no_cashier_on_sale` rule. |

Use exactly these strings unless the backend is changed.

## Recommended Event Deduplication

Do not insert a row for every frame. Insert semantic events.

Recommended rules:

- Emit `customer_present` when customer presence starts or changes from absent to present.
- Emit `customer_left` when customer leaves after being present.
- Emit `cashier_present` and `no_cashier` only on state changes or at a low heartbeat rate.
- Emit `item_in_bag` once per tracked item entering bagging ROI.
- Emit `hand_to_drawer` once per drawer interaction, with cooldown.
- Emit `document_presented` once per age-check interaction.
- Emit `phone_scanned_by_cashier` once per suspicious QR/phone scan action.

Suggested cooldowns:

| Event Type | Cooldown |
| --- | ---: |
| `customer_present` | State change only |
| `customer_left` | State change only |
| `cashier_present` | State change or 10-30 seconds |
| `no_cashier` | State change or 10-30 seconds |
| `item_in_bag` | Per object track |
| `hand_to_drawer` | 3-5 seconds |
| `document_presented` | 5-10 seconds |
| `phone_scanned_by_cashier` | 3-5 seconds |

## `bbox_jsonb` Recommendation

Use structured JSON that can be debugged later.

Recommended shape:

```json
{
  "bbox": [120, 80, 260, 220],
  "class_name": "item",
  "track_id": "track-123",
  "roi": "bag_zone",
  "frame_id": 45678,
  "extra": {
    "direction": "scanner_to_bag",
    "source_roi": "scanner_zone",
    "target_roi": "bag_zone"
  }
}
```

## Video Export Task Worker

The Go backend creates rows in `tasks` when it needs a video proof clip.

The video server must poll tasks with:

```sql
SELECT id, task_type, camera_id, violation_id, payload
FROM tasks
WHERE status = 'pending'
  AND task_type = 'video_export'
ORDER BY created_at ASC
FOR UPDATE SKIP LOCKED
LIMIT 1;
```

Use a transaction so multiple workers do not process the same task.

Recommended claim query:

```sql
WITH next_task AS (
  SELECT id
  FROM tasks
  WHERE status = 'pending'
    AND task_type = 'video_export'
  ORDER BY created_at ASC
  FOR UPDATE SKIP LOCKED
  LIMIT 1
)
UPDATE tasks
SET status = 'processing',
    updated_at = CURRENT_TIMESTAMP
WHERE id IN (SELECT id FROM next_task)
RETURNING id, task_type, camera_id, violation_id, payload;
```

The `payload` JSON contains:

```json
{
  "start_timestamp_ms": 1759999990000,
  "end_timestamp_ms": 1760000010000
}
```

Expected behavior:

1. Claim a pending task.
2. Read `camera_id`.
3. Parse `payload.start_timestamp_ms`.
4. Parse `payload.end_timestamp_ms`.
5. Export MP4 clip for that camera and time window.
6. Store the MP4 in a location reachable by operator UI or backend.
7. Update task row.

On success:

```sql
UPDATE tasks
SET status = 'completed',
    result_path = $2,
    error_message = NULL,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1;
```

On failure:

```sql
UPDATE tasks
SET status = 'failed',
    error_message = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1;
```

Do not set `processed_at`. That field is reserved for the Go backend acknowledgement.

## Video Export Output Path

`tasks.result_path` should be one of:

- Absolute path on shared storage accessible to backend/operator UI.
- HTTP URL to a video file.
- Object storage URL.

Recommended filename:

```text
violation_<violation_id>_task_<task_id>_<camera_id>_<start_ms>_<end_ms>.mp4
```

Example:

```text
/data/video_exports/violation_123_task_456_cam-1_1759999990000_1760000010000.mp4
```

## Worker Loop Pseudocode

```python
while True:
    task = claim_pending_video_export_task()

    if task is None:
        sleep(0.5)
        continue

    try:
        payload = task["payload"]
        result_path = export_clip(
            camera_id=task["camera_id"],
            start_ms=payload["start_timestamp_ms"],
            end_ms=payload["end_timestamp_ms"],
        )
        mark_completed(task["id"], result_path)
    except Exception as exc:
        mark_failed(task["id"], str(exc))
```

## Python SQL Skeleton

This is an implementation sketch. Adapt it to your DB library and deployment.

```python
import json
import time
import psycopg


CLAIM_SQL = """
WITH next_task AS (
  SELECT id
  FROM tasks
  WHERE status = 'pending'
    AND task_type = 'video_export'
  ORDER BY created_at ASC
  FOR UPDATE SKIP LOCKED
  LIMIT 1
)
UPDATE tasks
SET status = 'processing',
    updated_at = CURRENT_TIMESTAMP
WHERE id IN (SELECT id FROM next_task)
RETURNING id, task_type, camera_id, violation_id, payload;
"""


def claim_task(conn):
    with conn.transaction():
        row = conn.execute(CLAIM_SQL).fetchone()
        if row is None:
            return None
        return {
            "id": row[0],
            "task_type": row[1],
            "camera_id": row[2],
            "violation_id": row[3],
            "payload": row[4] if isinstance(row[4], dict) else json.loads(row[4]),
        }


def mark_completed(conn, task_id, result_path):
    conn.execute(
        """
        UPDATE tasks
        SET status = 'completed',
            result_path = %s,
            error_message = NULL,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = %s
        """,
        (result_path, task_id),
    )
    conn.commit()


def mark_failed(conn, task_id, error_message):
    conn.execute(
        """
        UPDATE tasks
        SET status = 'failed',
            error_message = %s,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = %s
        """,
        (error_message[:2000], task_id),
    )
    conn.commit()


def insert_cv_event(conn, event):
    conn.execute(
        """
        INSERT INTO cv_events (
          camera_id,
          event_type,
          timestamp_ms,
          confidence,
          model_name,
          weights_version,
          inference_time_ms,
          bbox_jsonb,
          snapshot_path
        ) VALUES (%s, %s, %s, %s, %s, %s, %s, %s::jsonb, %s)
        """,
        (
            event["camera_id"],
            event["event_type"],
            event["timestamp_ms"],
            event["confidence"],
            event["model_name"],
            event["weights_version"],
            event["inference_time_ms"],
            json.dumps(event["bbox_jsonb"], ensure_ascii=False),
            event.get("snapshot_path", ""),
        ),
    )
    conn.commit()


def now_ms():
    return int(time.time() * 1000)
```

## Error Handling

The video server should mark a task as `failed` when:

- Camera buffer is unavailable.
- Requested time window is outside retention.
- Video export process fails.
- Output storage is unavailable.
- Payload is invalid.

Use clear `error_message` values, for example:

```text
buffer window not available for camera cam-1: requested 1759999990000-1760000010000
```

## Retention Requirements

The Go backend creates video tasks for a window around the violation time:

```text
event_timestamp_ms - 10000
event_timestamp_ms + 10000
```

The video server should keep at least 20 seconds around recent events. In practice, use a larger rolling buffer, for example 2-10 minutes, to handle delays.

## Startup Checklist

1. Connect to PostgreSQL.
2. Load active cameras from `cameras`.
3. Start video stream readers.
4. Start CV detection loop for each camera.
5. Start video export worker loop.
6. Periodically refresh camera config or restart when camera config changes.
7. Write `cv_events` only with valid `camera_id`.
8. Claim `tasks` using `FOR UPDATE SKIP LOCKED`.
9. Update every claimed task to `completed` or `failed`.

## Integration Test Checklist

### Check Camera Mapping

```sql
SELECT id, pos_id, status
FROM cameras;
```

### Insert Test CV Event

Only run this in a test environment or with an agreed test camera:

```sql
INSERT INTO cv_events (
  camera_id,
  event_type,
  timestamp_ms,
  confidence,
  model_name,
  weights_version,
  inference_time_ms,
  bbox_jsonb,
  snapshot_path
) VALUES (
  'cam-test',
  'customer_present',
  EXTRACT(EPOCH FROM NOW())::bigint * 1000,
  0.95,
  'manual-test',
  'manual',
  0,
  '{}'::jsonb,
  ''
);
```

### Check Pending Tasks

```sql
SELECT id, task_type, camera_id, violation_id, payload, status, created_at
FROM tasks
WHERE status = 'pending'
ORDER BY created_at ASC;
```

### Check Completed Task Handling

After the video server marks a task `completed`, the Go backend should:

- Update `violations.proof_video_path`.
- Set `tasks.processed_at`.
- Send WebSocket `task_status` to operators.

## Production Notes

- Use a DB connection pool.
- Use retry with backoff for transient DB errors.
- Use `FOR UPDATE SKIP LOCKED` for horizontal worker scaling.
- Keep video export idempotent where possible.
- Log task ID, camera ID, violation ID, and requested time window for every export.
- Do not emit CV events at frame rate.
- Keep server clocks synchronized through NTP.
- Protect output video storage from public access unless URLs are signed.
- Avoid storing camera credentials in logs.
