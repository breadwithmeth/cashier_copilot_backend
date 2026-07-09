# Frontend Integration Guide

This document describes how the frontend should integrate with the Cashier Copilot backend.

The backend exposes:

- REST API for cameras, POS events, and violations.
- WebSocket stream for operator dashboard alerts.
- WebSocket stream for cashier terminal upsell prompts.

The backend uses local username/password login and Bearer access tokens.

## Frontend Applications

The system has two expected frontend surfaces:

1. **Operator Dashboard**
   - Shows violations journal.
   - Receives real-time violation alerts.
   - Receives video export task status updates.
   - Displays proof video path when ready.
   - Manages camera configuration.

2. **Cashier Terminal UI**
   - Connects by `pos_id`.
   - Receives AI Co-Pilot upsell cards.
   - Receives upsell completion status updates.

## Backend Base URLs

Recommended frontend environment variables:

```bash
VITE_API_BASE_URL=http://127.0.0.1:8080
VITE_WS_BASE_URL=ws://127.0.0.1:8080
```

For production:

```bash
VITE_API_BASE_URL=https://api.example.com
VITE_WS_BASE_URL=wss://api.example.com
```

Do not hardcode backend URLs in components. Keep them in configuration.

## Health Check

Use this to verify backend availability:

```http
GET /health
```

Expected response:

```text
.
```

Status:

```text
200 OK
```

## Authentication

Login endpoint:

```http
POST /api/v1/auth/login
```

Request:

```json
{
  "username": "admin",
  "password": "password"
}
```

Response:

```json
{
  "access_token": "eyJ...",
  "token_type": "Bearer",
  "expires_at": 1760000000,
  "user": {
    "id": 1,
    "username": "admin",
    "role": "admin"
  }
}
```

Store the token in frontend auth state and send it on protected REST requests:

```http
Authorization: Bearer <access_token>
```

Current roles:

| Role | Frontend Access |
| --- | --- |
| `admin` | Operator dashboard, camera management, user management, cashier stream. |
| `operator` | Operator dashboard, camera list, cashier stream. |
| `cashier` | Cashier terminal for assigned `pos_id`. |

Current user:

```http
GET /api/v1/auth/me
Authorization: Bearer <access_token>
```

## REST API Summary

Base path:

```text
/api/v1
```

| Method | Path | Frontend Use |
| --- | --- | --- |
| `POST` | `/api/v1/auth/login` | User login. |
| `GET` | `/api/v1/auth/me` | Current user. |
| `GET` | `/api/v1/users` | Admin user list. |
| `POST` | `/api/v1/users` | Admin user creation. |
| `GET` | `/api/v1/violations` | Operator violations journal. |
| `GET` | `/api/v1/cameras` | Camera list/settings page. |
| `POST` | `/api/v1/cameras` | Add camera configuration. |
| `POST` | `/api/v1/pos/event` | Development/test event injection. Usually sent by 1C, not frontend. |

## Common API Rules

### JSON Headers

Use:

```http
Content-Type: application/json
Accept: application/json
```

For protected endpoints, also send:

```http
Authorization: Bearer <access_token>
```

### Error Shape

Backend errors use:

```json
{
  "error": "message",
  "details": "optional details"
}
```

Frontend should display `error` and log `details` for debugging.

### Timestamp Format

The backend uses Unix milliseconds:

```ts
type TimestampMs = number;
```

Convert for display:

```ts
const date = new Date(timestamp_ms);
```

## Operator Dashboard

### Main Views

Recommended views:

- **Live Alerts**
  - Real-time WebSocket alerts.
  - Current task/video status.
  - Alert details.

- **Violations Journal**
  - Paginated table.
  - Filters by POS, type, status, date range.
  - Link/open proof video when available.

- **Camera Settings**
  - List cameras.
  - Add camera.
  - Show POS mapping.
  - Show ROI config as JSON or visual editor later.

### `GET /api/v1/violations`

Fetch paginated violation history.

Query parameters:

| Parameter | Type | Description |
| --- | --- | --- |
| `pos_id` | `string` | Optional POS terminal filter. |
| `type` | `string` | Optional violation type filter. |
| `status` | `string` | Optional status filter. |
| `from_ts` | `number` | Optional start time in Unix ms. |
| `to_ts` | `number` | Optional end time in Unix ms. |
| `limit` | `number` | Page size, default `50`, max `200`. |
| `offset` | `number` | Offset, default `0`. |

Example:

```http
GET /api/v1/violations?limit=20&offset=0
```

Response:

```json
{
  "data": [
    {
      "id": 1,
      "pos_id": "pos-1",
      "violation_type": "unscanned_item",
      "timestamp_ms": 1760000000000,
      "proof_video_path": "/data/video_exports/violation_1.mp4",
      "proof_image_path": null,
      "cv_event_id": 10,
      "pos_event_id": null,
      "speech_transcript_id": null,
      "confidence_aggregate": 0.91,
      "status": "new"
    }
  ],
  "total": 1,
  "limit": 20,
  "offset": 0
}
```

TypeScript types:

```ts
export type ViolationStatus = "new" | "auto_filtered" | string;

export type ViolationType =
  | "unscanned_item"
  | "void_without_return"
  | "loyalty_card_abuse"
  | "age_verification_failed"
  | "drawer_opened_without_sale"
  | "no_cashier_on_sale"
  | string;

export interface Violation {
  id: number;
  pos_id: string;
  violation_type: ViolationType;
  timestamp_ms: number;
  proof_video_path?: string | null;
  proof_image_path?: string | null;
  cv_event_id?: number | null;
  pos_event_id?: number | null;
  speech_transcript_id?: number | null;
  confidence_aggregate: number;
  status: ViolationStatus;
}

export interface ViolationListResponse {
  data: Violation[];
  total: number;
  limit: number;
  offset: number;
}
```

Recommended UI columns:

- Time.
- POS ID.
- Violation type.
- Confidence.
- Status.
- Proof video.
- Linked event IDs.

Recommended actions:

- Open violation details drawer/page.
- Open proof video when `proof_video_path` is present.
- Refresh list.
- Apply filters.

### Violation Type Labels

Recommended display labels:

| Type | Label |
| --- | --- |
| `unscanned_item` | Unscanned item |
| `void_without_return` | Void without return |
| `loyalty_card_abuse` | Loyalty card abuse |
| `age_verification_failed` | Age verification failed |
| `drawer_opened_without_sale` | Drawer opened without sale |
| `no_cashier_on_sale` | No cashier on sale |

Use local language labels in the product UI if needed, but keep API values unchanged.

### Confidence Display

The backend returns `confidence_aggregate` from `0.0` to `1.0`.

Display as percent:

```ts
const percent = Math.round(confidence_aggregate * 100);
```

Suggested visual severity:

| Confidence | Severity |
| ---: | --- |
| `>= 0.90` | High |
| `>= 0.75` | Medium |
| `< 0.75` | Low or filtered |

## Camera Settings

### `GET /api/v1/cameras`

Fetch camera list.

Example:

```http
GET /api/v1/cameras
```

Response:

```json
[
  {
    "id": "cam-1",
    "ip_address": "192.168.1.10",
    "username": "admin",
    "pos_id": "pos-1",
    "status": "active",
    "roi_config": {
      "bag_zone": [[0, 0], [100, 0], [100, 100], [0, 100]]
    },
    "created_at": "2026-07-09T12:00:00Z"
  }
]
```

TypeScript type:

```ts
export interface Camera {
  id: string;
  ip_address: string;
  username: string;
  password?: string;
  pos_id: string;
  status: string;
  roi_config: unknown;
  created_at: string;
}
```

Note: `password` is omitted in JSON responses when empty, but may be returned if stored. Frontend should avoid displaying stored passwords.

### `POST /api/v1/cameras`

Create a new camera.

Request:

```json
{
  "id": "cam-1",
  "ip_address": "192.168.1.10",
  "username": "admin",
  "password": "camera-password",
  "pos_id": "pos-1",
  "status": "active",
  "roi_config": {
    "bag_zone": [[0, 0], [100, 0], [100, 100], [0, 100]]
  }
}
```

Required:

- `id`
- `ip_address`
- `pos_id`

Optional:

- `username`
- `password`
- `status`
- `roi_config`

Backend defaults:

- `status = "active"` if omitted.
- `roi_config = {}` if omitted.

Recommended form validation:

- Camera ID is non-empty.
- IP address is non-empty.
- POS ID is non-empty.
- ROI config is valid JSON.

Current backend behavior:

- Create only.
- Duplicate `id` returns server/database error.
- No update or delete endpoint yet.

## WebSocket Integration

### Operator WebSocket

Endpoint:

```text
GET /ws/operator
```

Browser URL:

```ts
const ws = new WebSocket(`${WS_BASE_URL}/ws/operator?token=${encodeURIComponent(accessToken)}`);
```

Incoming envelope:

```ts
export interface WSMessage<T = unknown> {
  type: string;
  payload: T;
}
```

### Operator Message: `violation_alert`

Sent when a new violation is detected and passes the confidence threshold.

Payload:

```ts
export interface ViolationAlert {
  violation: Violation;
  pos_event?: PosEvent;
  cv_event?: CvEvent;
}
```

Example:

```json
{
  "type": "violation_alert",
  "payload": {
    "violation": {
      "id": 1,
      "pos_id": "pos-1",
      "violation_type": "unscanned_item",
      "timestamp_ms": 1760000000000,
      "cv_event_id": 10,
      "confidence_aggregate": 0.91,
      "status": "new"
    }
  }
}
```

Recommended UI behavior:

- Add alert to top of live feed.
- Show visual indication for new alert.
- Optionally refetch `/api/v1/violations`.
- Show video state as pending until `task_status` arrives.

### Operator Message: `task_status`

Sent when video export task is completed or failed.

Payload:

```ts
export interface TaskStatusUpdate {
  task_id: number;
  violation_id: number;
  status: "completed" | "failed" | string;
  video_path?: string;
}
```

Example completed:

```json
{
  "type": "task_status",
  "payload": {
    "task_id": 10,
    "violation_id": 1,
    "status": "completed",
    "video_path": "/data/video_exports/violation_1.mp4"
  }
}
```

Example failed:

```json
{
  "type": "task_status",
  "payload": {
    "task_id": 10,
    "violation_id": 1,
    "status": "failed"
  }
}
```

Recommended UI behavior:

- Find matching violation by `violation_id`.
- If `completed`, attach `video_path` and enable proof video action.
- If `failed`, show video export failed state.
- Optionally refetch violation details/list.

### Cashier WebSocket

Endpoint:

```text
GET /ws/cashier?pos_id=XXX
```

Browser URL:

```ts
const ws = new WebSocket(
  `${WS_BASE_URL}/ws/cashier?pos_id=${encodeURIComponent(posId)}&token=${encodeURIComponent(accessToken)}`,
);
```

The `pos_id` is required.

### Cashier Message: `upsell_card`

Sent when a scanned item matches an upsell rule.

Payload:

```ts
export interface UpsellCard {
  pos_id: string;
  receipt_id: string;
  trigger_item: string;
  suggestion_text: string;
  suggestion_image?: string;
  status: "pending" | "completed" | string;
}
```

Example:

```json
{
  "type": "upsell_card",
  "payload": {
    "pos_id": "pos-1",
    "receipt_id": "receipt-1",
    "trigger_item": "Beer",
    "suggestion_text": "Offer snacks",
    "suggestion_image": "",
    "status": "pending"
  }
}
```

Recommended UI behavior:

- Show compact recommendation card.
- Do not block cashier workflow.
- Replace existing card for same `receipt_id`.
- Hide card on completed status or receipt change if known.

### Cashier Message: `upsell_status`

Sent when speech transcript indicates cashier completed the upsell.

Payload:

```ts
export interface UpsellStatusUpdate {
  pos_id: string;
  receipt_id: string;
  status: "completed" | string;
}
```

Example:

```json
{
  "type": "upsell_status",
  "payload": {
    "pos_id": "pos-1",
    "receipt_id": "receipt-1",
    "status": "completed"
  }
}
```

Recommended UI behavior:

- Mark current upsell card as completed.
- Hide after a short delay.
- Do not send acknowledgement; backend does not expect one.

## WebSocket Client Behavior

Implement reconnect with backoff.

Recommended behavior:

- Connect on page load.
- Show connection state: connecting, connected, disconnected.
- Reconnect after close.
- Use exponential backoff with max delay, for example 1s, 2s, 5s, 10s, 30s.
- Reset backoff after successful connection.
- Ignore unknown message types but log them.
- Validate message shape before updating UI.

Example TypeScript helper:

```ts
type MessageHandler = (message: WSMessage) => void;

export function connectWithReconnect(
  url: string,
  onMessage: MessageHandler,
  onState?: (state: "connecting" | "connected" | "disconnected") => void,
) {
  let socket: WebSocket | null = null;
  let stopped = false;
  let retryMs = 1000;

  const connect = () => {
    if (stopped) return;

    onState?.("connecting");
    socket = new WebSocket(url);

    socket.onopen = () => {
      retryMs = 1000;
      onState?.("connected");
    };

    socket.onmessage = (event) => {
      try {
        onMessage(JSON.parse(event.data));
      } catch (error) {
        console.error("Invalid WebSocket message", error);
      }
    };

    socket.onclose = () => {
      onState?.("disconnected");
      if (stopped) return;

      const delay = retryMs;
      retryMs = Math.min(retryMs * 2, 30000);
      window.setTimeout(connect, delay);
    };

    socket.onerror = () => {
      socket?.close();
    };
  };

  connect();

  return () => {
    stopped = true;
    socket?.close();
  };
}
```

## POS Event Injection for Development

Normally POS events come from 1C, not from frontend.

For development/test panels, frontend can call:

```http
POST /api/v1/pos/event
```

This endpoint uses service API key auth, not user Bearer auth:

```http
X-API-Key: <POS_API_KEY>
```

Request:

```json
{
  "pos_id": "pos-1",
  "receipt_id": "receipt-1",
  "event_type": "item_scanned",
  "timestamp_ms": 1760000000000,
  "details": {
    "sku": "sku-1",
    "item_name": "Beer",
    "category": "Alcohol/Beer",
    "price": 1200,
    "quantity": 1,
    "age_restricted": true
  }
}
```

Response:

```json
{
  "id": 1,
  "status": "accepted",
  "fsm_state": "Scanning"
}
```

Supported event types:

- `receipt_opened`
- `item_scanned`
- `item_removed`
- `receipt_cancelled`
- `loyalty_card_applied`
- `payment_started`
- `receipt_closed`

## Recommended Frontend State Model

### Operator Store

Suggested state:

```ts
interface OperatorState {
  violations: Violation[];
  total: number;
  limit: number;
  offset: number;
  filters: {
    pos_id?: string;
    type?: string;
    status?: string;
    from_ts?: number;
    to_ts?: number;
  };
  liveAlerts: ViolationAlert[];
  taskStatusByViolationId: Record<number, TaskStatusUpdate>;
  wsState: "connecting" | "connected" | "disconnected";
}
```

### Cashier Store

Suggested state:

```ts
interface CashierState {
  posId: string;
  currentUpsell?: UpsellCard;
  upsellHistory: UpsellCard[];
  wsState: "connecting" | "connected" | "disconnected";
}
```

## Proof Video Handling

The backend currently returns `proof_video_path` or WebSocket `video_path` exactly as provided by the video export worker.

Frontend must support these possibilities:

- Absolute server file path.
- Relative path.
- HTTP URL.
- Object storage URL.

Recommended handling:

```ts
function resolveVideoUrl(path: string): string {
  if (path.startsWith("http://") || path.startsWith("https://")) {
    return path;
  }

  // Current backend does not serve local video files.
  // Replace this with the real media gateway when available.
  return path;
}
```

Important: the current Go backend does not expose a static file route for videos. If `result_path` is a local filesystem path, the frontend cannot play it directly in the browser until a media-serving endpoint or object storage URL is added.

## CORS and Local Development

The backend currently allows all origins for development.

Expected local setup:

```text
Frontend: http://127.0.0.1:5173
Backend:  http://127.0.0.1:8080
```

Use `VITE_API_BASE_URL` and `VITE_WS_BASE_URL` to avoid hardcoding.

## UI States to Handle

### REST Requests

For every REST view, handle:

- Loading.
- Empty state.
- Error state.
- Success state.
- Retry action.

### WebSocket

For WebSocket views, handle:

- Connecting.
- Connected.
- Disconnected/reconnecting.
- Unknown message type.
- Invalid JSON.

### Video Proof

For proof video:

- Not requested yet.
- Export pending.
- Export completed.
- Export failed.
- URL/path unavailable.

## Manual Test Plan

### 1. Backend Health

```bash
curl -i http://127.0.0.1:8080/health
```

Expected:

```text
200 OK
```

### 2. Login

```bash
curl -i -X POST http://127.0.0.1:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"password"}'
```

Expected:

```text
200 OK
```

Use `access_token` from the response in the following requests.

### 3. Fetch Cameras

```bash
curl -i http://127.0.0.1:8080/api/v1/cameras \
  -H "Authorization: Bearer ACCESS_TOKEN"
```

Expected:

```json
[]
```

or camera list.

### 4. Fetch Violations

```bash
curl -i 'http://127.0.0.1:8080/api/v1/violations?limit=5' \
  -H "Authorization: Bearer ACCESS_TOKEN"
```

Expected:

```json
{
  "data": [],
  "total": 0,
  "limit": 5,
  "offset": 0
}
```

### 5. Connect Operator WebSocket

Use browser code:

```js
const ws = new WebSocket("ws://127.0.0.1:8080/ws/operator?token=ACCESS_TOKEN");
ws.onopen = () => console.log("connected");
ws.onmessage = (event) => console.log(JSON.parse(event.data));
ws.onclose = () => console.log("closed");
```

### 6. Connect Cashier WebSocket

```js
const ws = new WebSocket("ws://127.0.0.1:8080/ws/cashier?pos_id=pos-1&token=ACCESS_TOKEN");
ws.onopen = () => console.log("connected");
ws.onmessage = (event) => console.log(JSON.parse(event.data));
ws.onclose = () => console.log("closed");
```

## Missing Backend Features Frontend Should Not Assume

The frontend should not assume these features exist yet:

- Violation status update endpoint.
- Camera update/delete endpoints.
- Static serving of proof videos.
- REST endpoint for task list.
- REST endpoint for live POS state.
- REST endpoint for raw CV events.
- REST endpoint for raw speech transcripts.
- WebSocket acknowledgement messages.

If the frontend needs any of these, they must be added to the backend contract first.

## Production Requirements To Add Later

Before production, define:

- Refresh token rotation.
- Password reset flow.
- External identity provider integration.
- Restricted CORS origins.
- Media serving strategy for proof videos.
- Camera password handling policy.
- Pagination and search UX for large violation history.
- Audit log for operator actions.
- Localization.
