# Coolify Nixpacks deployment

Этот вариант нужен, если API и web деплоятся как два отдельных Coolify resource через Nixpacks.

## Общая схема

Создайте в Coolify три ресурса:

- PostgreSQL database.
- API application.
- Web application.

Оба application-ресурса можно создать из одного Git repository. Build directory оставьте корнем репозитория, потому что проект использует npm workspaces и общий `package-lock.json`.

## API resource

Настройки:

- Build Pack: `Nixpacks`
- Base Directory: `/`
- Nixpacks Config Path: `apps/api/nixpacks.toml`
- Port: `3000`

Environment variables:

```env
DATABASE_URL=postgresql://USER:PASSWORD@HOST:5432/DB
JWT_ACCESS_SECRET=strong-random-access-secret
JWT_REFRESH_SECRET=strong-random-refresh-secret
ACCESS_TOKEN_TTL=15m
REFRESH_TOKEN_DAYS=30
PORT=3000
WEB_ORIGIN=https://copilot.naliv.kz
WORKER_HEARTBEAT_TIMEOUT_SECONDS=90
ONE_C_API_KEY=strong-1c-api-key
ONE_C_ORGANIZATION_CODE=NALIV
```

После первого deploy выполните в terminal API-ресурса:

```bash
npm run db:generate
npm run db:seed
```

Если база пустая и SQL baseline еще не применен, примените миграции из `packages/database/migrations`.

## Web resource

Настройки:

- Build Pack: `Nixpacks`
- Base Directory: `/`
- Nixpacks Config Path: `apps/web/nixpacks.toml`
- Port: `3001`

Environment variables:

```env
NEXT_PUBLIC_API_URL=https://api.naliv.kz/api
NEXT_PUBLIC_WS_URL=https://api.naliv.kz
PORT=3001
```

Важно: `NEXT_PUBLIC_API_URL` и `NEXT_PUBLIC_WS_URL` должны быть доступны на этапе build, потому что Next.js вшивает эти значения в frontend bundle.

## Домены

Самый простой вариант для двух Nixpacks-проектов:

- API: `https://api.naliv.kz`
- Web: `https://copilot.naliv.kz`

Если нужен строго один публичный домен, используйте Docker Compose вариант из корневого `docker-compose.yml`: там есть `proxy`, который маршрутизирует `/api/*` в API, а остальные пути в web.

Для одного домена с двумя отдельными Nixpacks-ресурсами нужен отдельный reverse proxy resource или ручные route rules в инфраструктуре Coolify:

- `https://copilot.naliv.kz/api/*` -> API resource
- `https://copilot.naliv.kz/socket.io/*` -> API resource
- `https://copilot.naliv.kz/*` -> Web resource

В таком случае для web задайте:

```env
NEXT_PUBLIC_API_URL=/api
NEXT_PUBLIC_WS_URL=/
```
