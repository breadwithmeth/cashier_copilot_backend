# Python Analytics Worker

Документ описывает внешний Python-сервис видеоаналитики, который подключается к Cashier Copilot Backend и отправляет результаты обработки кассовых камер.

Backend предоставляет worker API по адресу:

```text
http://localhost:3000/api
```

В production вместо `localhost` используется публичный URL API.

## Назначение сервиса

Python worker отвечает за:

- получение своей конфигурации и списка назначенных камер;
- периодический heartbeat;
- запуск processing session для обработки камеры или потока;
- отправку метрик по камерам;
- отправку аналитических событий: нарушения, подозрительные действия, технические события;
- отправку evidence: изображения, видеофрагменты, ссылки на файлы;
- отправку транскриптов речи с привязкой к камере, чеку или sale session;
- отправку ошибок обработки в `integration_errors`;
- отправку логов обработки.

Worker не использует пользовательский JWT. Авторизация выполняется по заголовку `X-Worker-Key`.

## Переменные окружения

Минимальный набор для Python-сервиса:

```env
CASHIER_API_URL=http://localhost:3000/api
CASHIER_WORKER_KEY=replace-with-worker-api-key
WORKER_STATUS=ONLINE
HEARTBEAT_INTERVAL_SECONDS=30
REQUEST_TIMEOUT_SECONDS=10
```

Рекомендуемые дополнительные переменные:

```env
WORKER_NAME=checkout-vision-01
LOG_LEVEL=INFO
MEDIA_BASE_PATH=/var/lib/cashier-worker/evidence
```

`CASHIER_WORKER_KEY` должен соответствовать записи в таблице `analytics_workers`. В базе хранится не сам ключ, а argon2 hash в поле `api_key_hash`.

## Авторизация

Каждый запрос worker-а должен содержать:

```http
X-Worker-Key: <CASHIER_WORKER_KEY>
Content-Type: application/json
```

Если ключ неверный, backend вернет `401 Unauthorized`.

## Endpoint-ы

### Heartbeat

```http
POST /api/workers/heartbeat
```

Сообщает backend, что worker жив, и обновляет статус.

Тело запроса:

```json
{
  "status": "ONLINE",
  "metadata": {
    "pid": 12345,
    "gpu": "cuda:0",
    "queueSize": 0
  }
}
```

Допустимые практические статусы:

- `ONLINE` - worker готов обрабатывать камеры;
- `BUSY` - worker работает под нагрузкой;
- `OFFLINE` - worker штатно выключается;
- `ERROR` - worker запущен, но часть обработки недоступна.

### Конфигурация worker-а

```http
GET /api/workers/me/config
```

Пример ответа:

```json
{
  "worker": {
    "id": "1",
    "organization_id": "1",
    "name": "checkout-vision-01",
    "host": "worker-host",
    "version": "1.0.0",
    "status": "ONLINE",
    "capabilities": {},
    "metadata": {}
  },
  "heartbeatIntervalSeconds": 30
}
```

### Назначенные камеры

```http
GET /api/workers/me/cameras
```

Возвращает камеры, назначенные текущему worker-у. В ответе есть объект `camera`, его потоки, ROI, модели и версии моделей.

Backend маскирует логин и пароль в `stream_url`. Если worker должен реально подключаться к RTSP-потоку, храните рабочий URL в локальном secure-хранилище worker-а или передавайте его через отдельный безопасный канал.

Типы потоков в `camera.camera_streams`:

- `RTSP_VIDEO` - основной видео RTSP-поток камеры;
- `RTSP_AUDIO` - отдельный аудио RTSP-поток камеры;
- `RTSP` - legacy-тип, если поток был создан старой версией интерфейса.

Python worker должен выбирать `RTSP_VIDEO` для видеоаналитики и `RTSP_AUDIO` для распознавания речи/аудиособытий, если такой поток задан.

### Processing session

```http
POST /api/workers/me/sessions
```

Создает сессию обработки.

Пример:

```json
{
  "camera_id": "1",
  "session_type": "STREAM",
  "startedAt": "2026-07-12T13:00:00.000Z",
  "status": "RUNNING",
  "metadata": {
    "model": "cashier-action-detector",
    "modelVersion": "2026.07.1"
  }
}
```

Точные поля сохраняются в таблицу `processing_sessions`; дополнительные данные можно класть в `metadata`.

### Метрики камеры

```http
POST /api/workers/me/metrics
```

Backend принимает не чаще одного значения на камеру за 30 секунд. При превышении лимита вернет:

```json
{
  "accepted": false,
  "reason": "rate_limited"
}
```

Пример:

```json
{
  "camera_id": "1",
  "recordedAt": "2026-07-12T13:00:00.000Z",
  "fps": 24.8,
  "latency_ms": 120,
  "status": "ONLINE",
  "metadata": {
    "droppedFrames": 3,
    "gpuUtilization": 0.72
  }
}
```

### Отправка одного события

```http
POST /api/workers/me/events
```

Минимальное тело:

```json
{
  "cameraId": "1",
  "eventTypeCode": "NO_RECEIPT",
  "startedAt": "2026-07-12T13:00:00.000Z",
  "title": "Продажа без чека"
}
```

Полное тело:

```json
{
  "cameraId": "1",
  "eventTypeCode": "NO_RECEIPT",
  "startedAt": "2026-07-12T13:00:00.000Z",
  "deduplicationKey": "camera-1:no-receipt:20260712T130000Z",
  "title": "Продажа без чека",
  "description": "Модель обнаружила передачу товара без фискального события.",
  "severity": "WARNING",
  "confidence": 0.91,
  "metadata": {
    "model": "cashier-action-detector",
    "modelVersion": "2026.07.1",
    "rule": "no_receipt_after_payment"
  },
  "objects": [
    {
      "object_type": "person",
      "label": "cashier",
      "confidence": 0.98,
      "bbox": { "x": 120, "y": 80, "w": 180, "h": 420 },
      "metadata": {}
    }
  ],
  "transcripts": [
    {
      "startedAt": "2026-07-12T13:00:01.000Z",
      "finishedAt": "2026-07-12T13:00:04.000Z",
      "speaker": "CASHIER",
      "text": "Оплата наличными?",
      "language": "ru",
      "confidence": 0.86,
      "words": [],
      "metadata": {}
    }
  ],
  "evidence": [
    {
      "evidence_type": "IMAGE",
      "file_path": "/var/lib/cashier-worker/evidence/event-123.jpg",
      "public_url": "https://storage.example.com/events/event-123.jpg",
      "capturedAt": "2026-07-12T13:00:02.000Z",
      "videoStartedAt": "2026-07-12T12:59:52.000Z",
      "videoFinishedAt": "2026-07-12T13:00:12.000Z",
      "preSeconds": 10,
      "postSeconds": 10,
      "expiresAt": "2026-08-12T13:00:02.000Z",
      "metadata": {
        "width": 1280,
        "height": 720
      }
    }
  ]
}
```

`eventTypeCode` должен существовать в справочнике `event_types`. Если код неизвестен, backend вернет `400 Bad Request`.

`cameraId` должен принадлежать той же организации, что и worker. Если камера недоступна worker-у, backend вернет `400 Bad Request`.

`deduplicationKey` нужен для идемпотентности. Если событие с таким ключом уже есть, backend вернет существующую запись вместо создания дубля.

Backend умеет сам определить базовые нарушения по `metadata`:

- `eventTypeCode: "PRODUCT_SCANNED"` и `metadata.customerPresent: false` -> `PRODUCT_SCANNED_WITHOUT_CUSTOMER`;
- `metadata.customerPresent: true` и `metadata.receiptPresent: false` -> `CUSTOMER_WITHOUT_RECEIPT`;
- `metadata.productGiven: true` и `metadata.paid: false` -> `PRODUCT_GIVEN_WITHOUT_PAYMENT`;
- `metadata.receiptPresent: true` и `metadata.customerPresent: false` -> `RECEIPT_WITHOUT_CUSTOMER`;
- `metadata.receivingMismatch: true` -> `RECEIVING_MISMATCH`.

### Отправка пачки событий

```http
POST /api/workers/me/events/batch
```

Тело запроса - массив объектов того же формата, что и для одиночного события.

```json
[
  {
    "cameraId": "1",
    "eventTypeCode": "NO_RECEIPT",
    "startedAt": "2026-07-12T13:00:00.000Z",
    "deduplicationKey": "camera-1:no-receipt:20260712T130000Z",
    "title": "Продажа без чека"
  }
]
```

### Отправка транскрипта

```http
POST /api/workers/me/transcripts
```

Этот endpoint нужен для Python-сервиса распознавания речи. Транскрипт можно
привязать к событию, камере, чеку или sale session. Минимально backend должен
суметь определить `store_id` и `workplace_id` по одной из переданных связей.

Пример с привязкой к чеку:

```json
{
  "externalTranscriptId": "asr-000001",
  "sourceService": "whisper-worker-01",
  "cameraId": "1",
  "externalReceiptId": "receipt-000001",
  "startedAt": "2026-07-13T10:15:31.000Z",
  "finishedAt": "2026-07-13T10:15:36.000Z",
  "speaker": "CASHIER",
  "text": "С вас шестьсот пятьдесят тенге",
  "language": "ru",
  "confidence": 0.91,
  "audioUrl": "https://storage.example.com/audio/asr-000001.wav",
  "words": [
    { "word": "С", "start": 0.0, "end": 0.1 },
    { "word": "вас", "start": 0.1, "end": 0.3 }
  ],
  "metadata": {
    "model": "whisper-large-v3",
    "amountSpoken": true
  }
}
```

Пример с привязкой к событию:

```json
{
  "externalTranscriptId": "asr-000002",
  "sourceService": "whisper-worker-01",
  "eventId": "123",
  "startedAt": "2026-07-13T10:15:31.000Z",
  "text": "Оплата наличными?",
  "speaker": "CASHIER",
  "language": "ru"
}
```

Поля:

- `externalTranscriptId` + `sourceService` дают идемпотентность.
- `eventId` связывает транскрипт с `analytics_events`.
- `cameraId` связывает транскрипт с камерой.
- `externalReceiptId` связывает транскрипт с `receipts`.
- `saleSessionId` связывает транскрипт с `sale_sessions`.
- `audioUrl` хранит ссылку на аудиофрагмент.
- `words` хранит word-level timestamps, если ASR их умеет отдавать.

Транскрипты сохраняются в `event_transcripts`.

### Отправка пачки транскриптов

```http
POST /api/workers/me/transcripts/batch
```

Тело запроса - массив объектов того же формата:

```json
[
  {
    "externalTranscriptId": "asr-000001",
    "sourceService": "whisper-worker-01",
    "cameraId": "1",
    "externalReceiptId": "receipt-000001",
    "startedAt": "2026-07-13T10:15:31.000Z",
    "text": "Здравствуйте",
    "speaker": "CASHIER",
    "language": "ru"
  }
]
```

### Ошибки обработки

```http
POST /api/workers/me/errors
```

Endpoint сохраняет ошибку в `integration_errors`, чтобы ее было видно в интерфейсе
`/integration-errors` и на дашборде.

Пример:

```json
{
  "cameraId": "1",
  "sourceSystem": "PYTHON_WORKER",
  "entityType": "CAMERA_STREAM",
  "externalId": "camera-1-main",
  "errorCode": "RTSP_TIMEOUT",
  "errorMessage": "RTSP stream did not respond within timeout",
  "occurredAt": "2026-07-13T10:20:00.000Z",
  "payload": {
    "attempt": 3,
    "streamType": "RTSP_VIDEO"
  }
}
```

### Логи

```http
POST /api/workers/me/logs
```

Backend сейчас подтверждает прием логов и возвращает количество записей.

Пример:

```json
[
  {
    "level": "INFO",
    "message": "Camera processing started",
    "cameraId": "1",
    "timestamp": "2026-07-12T13:00:00.000Z"
  }
]
```

## Минимальный Python-клиент

Установка зависимостей:

```bash
python -m venv .venv
source .venv/bin/activate
pip install requests python-dotenv
```

Файл `.env` для worker-а:

```env
CASHIER_API_URL=http://localhost:3000/api
CASHIER_WORKER_KEY=replace-with-worker-api-key
```

Пример `worker_client.py`:

```python
import os
from datetime import datetime, timezone

import requests
from dotenv import load_dotenv

load_dotenv()


class CashierWorkerClient:
    def __init__(self) -> None:
        self.base_url = os.environ["CASHIER_API_URL"].rstrip("/")
        self.session = requests.Session()
        self.session.headers.update({
            "X-Worker-Key": os.environ["CASHIER_WORKER_KEY"],
            "Content-Type": "application/json",
        })
        self.timeout = float(os.getenv("REQUEST_TIMEOUT_SECONDS", "10"))

    def request(self, method: str, path: str, **kwargs):
        response = self.session.request(
            method,
            f"{self.base_url}{path}",
            timeout=self.timeout,
            **kwargs,
        )
        response.raise_for_status()
        return response.json()

    def heartbeat(self, status: str = "ONLINE", metadata: dict | None = None):
        return self.request("POST", "/workers/heartbeat", json={
            "status": status,
            "metadata": metadata or {},
        })

    def config(self):
        return self.request("GET", "/workers/me/config")

    def cameras(self):
        return self.request("GET", "/workers/me/cameras")

    def send_event(self, payload: dict):
        return self.request("POST", "/workers/me/events", json=payload)

    def send_transcript(self, payload: dict):
        return self.request("POST", "/workers/me/transcripts", json=payload)

    def send_error(self, payload: dict):
        return self.request("POST", "/workers/me/errors", json=payload)


def utc_now() -> str:
    return datetime.now(timezone.utc).isoformat()


if __name__ == "__main__":
    client = CashierWorkerClient()
    client.heartbeat(metadata={"source": "manual-test"})

    cameras = client.cameras()
    if not cameras:
        raise RuntimeError("No cameras assigned to this worker")

    camera_id = str(cameras[0]["camera"]["id"])
    event = client.send_event({
        "cameraId": camera_id,
        "eventTypeCode": "NO_RECEIPT",
        "startedAt": utc_now(),
        "deduplicationKey": f"{camera_id}:manual-test:{utc_now()}",
        "title": "Тестовое событие от Python worker",
        "severity": "INFO",
        "confidence": 0.99,
        "metadata": {
            "test": True,
            "client": "worker_client.py",
        },
    })
    print(event)

    transcript = client.send_transcript({
        "externalTranscriptId": f"manual-asr-{camera_id}-{utc_now()}",
        "sourceService": "manual-test",
        "cameraId": camera_id,
        "startedAt": utc_now(),
        "speaker": "CASHIER",
        "text": "Тестовый транскрипт от Python worker",
        "language": "ru",
        "confidence": 0.99,
    })
    print(transcript)
```

Запуск:

```bash
python worker_client.py
```

## Рекомендуемый цикл работы worker-а

1. Загрузить переменные окружения.
2. Отправить `POST /workers/heartbeat` со статусом `ONLINE`.
3. Получить `GET /workers/me/config`.
4. Получить `GET /workers/me/cameras`.
5. Запустить обработчики потоков для назначенных камер.
6. Отправлять heartbeat каждые `heartbeatIntervalSeconds`.
7. Отправлять metrics не чаще одного раза в 30 секунд на камеру.
8. При обнаружении события отправлять `POST /workers/me/events` с `deduplicationKey`.
9. При распознавании речи отправлять `POST /workers/me/transcripts`.
10. При ошибках RTSP, ASR, модели или storage отправлять `POST /workers/me/errors`.
11. При штатном завершении отправить heartbeat со статусом `OFFLINE`.

## Обработка ошибок

Рекомендуемая политика:

- `401` - неверный `CASHIER_WORKER_KEY`, остановить сервис и проверить секрет;
- `400` - ошибка payload, залогировать тело события и не повторять без исправления;
- `429` - throttling, повторить позже с backoff;
- `5xx` - временная ошибка backend, повторить с exponential backoff;
- network timeout - повторить запрос, но события отправлять с `deduplicationKey`.

Для событий используйте очередь на стороне worker-а. Если backend временно недоступен, событие должно оставаться в локальной очереди до успешной отправки или до истечения TTL.

## Требования к времени

Все даты передаются в ISO 8601:

```text
2026-07-12T13:00:00.000Z
```

Worker должен использовать UTC для `startedAt`, `recordedAt`, `capturedAt`, `expiresAt`.

## Безопасность

- Не логируйте `CASHIER_WORKER_KEY`.
- Не храните ключ в репозитории.
- Передавайте evidence через защищенное хранилище, если `public_url` доступен извне.
- Используйте HTTPS в production.
- Ограничивайте сетевой доступ worker-а к backend и хранилищу evidence.
