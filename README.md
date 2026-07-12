# Cashier Copilot

Monorepo системы управления видеоаналитикой кассовых зон: NestJS API, Next.js web и PostgreSQL. Python analytics worker подключается отдельно через `/api/workers/*`.

## Требования

- Node.js 22+
- npm 10+
- Docker и Docker Compose

## Быстрый запуск через Docker

1. Создать `.env`:

```bash
cp .env.example .env
```

2. Проверить значения в `.env`:

```env
DATABASE_URL=postgresql://cashier:cashier@postgres:5432/cashier
JWT_ACCESS_SECRET=change-me-access
JWT_REFRESH_SECRET=change-me-refresh
ONE_C_API_KEY=change-me-1c
ONE_C_ORGANIZATION_CODE=DEMO
```

Для запуска через `docker compose` у `DATABASE_URL` должен быть host `postgres`, потому что API подключается к Postgres внутри compose-сети.

3. Собрать и запустить сервисы:

```bash
docker compose up --build
```

4. В отдельном терминале подготовить Prisma Client и создать seed-данные:

```bash
docker compose exec api npm run db:generate
docker compose exec api npm run db:seed
```

После запуска:

- web: `http://localhost:3001`
- API Swagger: `http://localhost:3000/api/docs`

## Локальный запуск для разработки

1. Создать `.env`:

```bash
cp .env.example .env
```

2. Для локального запуска без Docker API должен видеть Postgres на localhost:

```env
DATABASE_URL=postgresql://cashier:cashier@localhost:5432/cashier
```

3. Запустить только Postgres:

```bash
docker compose up postgres
```

4. Установить зависимости и подготовить Prisma:

```bash
npm install
npm run db:generate
npm run db:seed
```

5. Запустить API и web:

```bash
npm run dev -w @cashier/api
```

В другом терминале:

```bash
npm run dev -w @cashier/web
```

После запуска:

- web: `http://localhost:3001`
- API Swagger: `http://localhost:3000/api/docs`

## Полезные команды

```bash
npm run build
npm run typecheck -w @cashier/api
npm run typecheck -w @cashier/database
npm run db:generate
npm run db:seed
```

## База данных

Основная схема описана в [packages/database/prisma/schema.prisma](packages/database/prisma/schema.prisma).

SQL-миграции лежат в [packages/database/migrations](packages/database/migrations):

- `20260711_init_schema.sql` - baseline для чистой БД;
- `20260712_add_platform_tables.sql` - пользователи, worker-ы, refresh tokens, audit;
- `20260713_add_1c_transaction_tables.sql` - сканы товаров и чеки;
- `20260714_harden_database_contracts.sql` - constraints, обязательные рабочие места, статусы;
- `20260715_sales_control_tables.sql` - строки чека, sale sessions, observations, service checks, приемка;
- `20260716_transcript_links.sql` - привязка транскриптов к чеку, сессии, камере и событию.

Для чистой БД достаточно применить baseline `20260711_init_schema.sql`. Последующие миграции нужны для обновления уже существующих БД и идемпотентны для повторного применения.

Цепочка основных сущностей:

```text
organization -> store -> workplace -> camera -> camera_streams
receipt -> receipt_items
receipt -> sale_session -> service_check_results
receipt/sale_session/camera -> event_transcripts
receipt/sale_session/camera -> video_observations
analytics_event -> event_evidence
```

## Интеграции

- 1С отправляет сканы товаров и чеки в `/api/integrations/1c/*`.
- Python analytics worker отправляет события, evidence, метрики и транскрипты в `/api/workers/me/*`.
- Python analytics worker отправляет ошибки обработки в `/api/workers/me/errors`; они сохраняются в `integration_errors`.
- Транскрипты сохраняются в `event_transcripts` и могут быть привязаны к событию, камере, чеку или sale session.

## Интерфейс

Основные рабочие экраны:

- `/dashboard` - операционная панель по магазинам, рабочим местам, камерам, чекам, сессиям продаж, нарушениям и ошибкам.
- `/events` - список событий и нарушений.
- `/events/:id` - экран расследования: событие, чек, позиции чека, сканы, транскрипт, evidence и видео-наблюдения.
- `/stores`, `/workplaces`, `/cameras`, `/streams` - настройка цепочки `магазин -> рабочее место -> камера -> поток`.
- `/receipts`, `/product-scans`, `/sale-sessions`, `/transcripts`, `/integration-errors` - эксплуатационные таблицы.

## Автоматические правила

Backend автоматически создает аналитические события для базовых сценариев:

- `PRODUCT_SCANNED_WITHOUT_CUSTOMER` - товар отсканирован, но в payload указано `customerPresent: false`;
- `CUSTOMER_WITHOUT_RECEIPT` - клиент присутствует, но чек не найден;
- `PRODUCT_GIVEN_WITHOUT_PAYMENT` - товар передан, но оплата не подтверждена;
- `RECEIPT_WITHOUT_CUSTOMER` - чек пробит без клиента в кадре;
- `RECEIVING_MISMATCH` - расхождение при приемке.

Seed создает базовые `event_types` и `violation_types`, чтобы worker и 1С могли сразу отправлять эти сценарии после подготовки базы.

## Документация

- Внешний Python analytics worker: [docs/python-worker.md](docs/python-worker.md)
- Интеграция 1С: [docs/one-c-integration.md](docs/one-c-integration.md)
