# Cashier Copilot

Monorepo системы управления видеоаналитикой кассовых зон: NestJS API, React SPA и PostgreSQL. Python analytics worker подключается отдельно через `/api/workers/*`.

```bash
cp .env.example .env
npm install
npm run db:generate
npm run build
docker compose up --build
```

Swagger: `http://localhost:3000/api/docs`, SPA: `http://localhost:5173`.
