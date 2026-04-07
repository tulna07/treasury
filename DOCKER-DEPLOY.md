# Docker Compose Deployment Guide — Treasury Management System

> All information verified directly from the codebase source files.

---

## Architecture Overview

| Service    | Image/Source              | Port  | Notes                                   |
|------------|---------------------------|-------|-----------------------------------------|
| `postgres` | `postgres:16-alpine`      | 5432  | Internal only                           |
| `redis`    | `redis:7-alpine`          | 6379  | Internal only, optional (rate limiting) |
| `minio`    | `minio/minio`             | 9000  | Object storage for Excel exports        |
| `migrate`  | `migrate/migrate`         | —     | One-shot, runs 13 numbered migrations   |
| `api`      | Built from `services/api` | 34080 | Go backend (`cmd/server/main.go`)       |
| `web`      | Built from `apps/web`     | 34000 | Next.js 16 frontend                     |

> **Dev `.env` discrepancies** (do not copy these to Docker):
> - `.env` has `APP_PORT=34001` — canonical port is `34080` (from `.env.example`, `ecosystem.config.cjs`, `config.go` default, `next.config.ts`)
> - `.env` has `SERVER_ENV=development` — `config.go` reads `APP_ENV`, not `SERVER_ENV`. The `SERVER_ENV` value is silently ignored.
> - `.env` has `OTEL_ENDPOINT=` — `config.go` reads `OTEL_EXPORTER_OTLP_ENDPOINT`, not `OTEL_ENDPOINT`. The `.env` value has no effect.

---

## Required Code Changes Before Building

### Change 1 — `app/apps/web/next.config.ts` — already applied

The rewrite now reads `API_INTERNAL_URL` at runtime. No action needed:

```ts
async rewrites() {
  const apiBase = process.env.API_INTERNAL_URL ?? "http://localhost:34080";
  return [{ source: "/api/:path*", destination: `${apiBase}/api/:path*` }];
},
```

`API_INTERNAL_URL=http://api:34080` is set in the web container environment (already in `docker-compose.yml`).

### Note — `output: 'standalone'` already present

`output: 'standalone'` is already in `next.config.ts`. `app/Dockerfile.web` already uses it. No action needed.

---

## Step-by-Step Deployment

### Step 1 — Build and start

All required code changes are already applied in the repo. Just build and run:

```bash
cd /Users/tulna/Downloads/treasury/app

docker compose build
docker compose up -d
docker compose logs -f api web
```

### Step 3 — Apply seed data (optional)

The `migrate` tool only runs numbered files in `migrations/`. Seed files in `migrations/seed/` must be applied manually:

```bash
# Base seed: roles, permissions, branches, currencies, admin user
docker compose exec -T postgres psql -U treasury -d treasury \
  < services/api/migrations/seed/001_seed.sql
```

Available seed files:
- `001_seed.sql` — roles, permissions, branches, currencies
- `002_fx_deals_seed.sql` — sample FX deals
- `003_bond_deals_seed.sql` — sample bond deals
- `004_credit_limits_seed.sql` — sample credit limits
- `005_ttqt_seed.sql` — sample international payments

### Step 4 — Apply export audit table (if using export feature)

The `export_audit_logs` table lives in `db/migrations/create_export_audit_logs.sql` — it is **not** in the numbered `migrations/` folder and will **not** be applied by `migrate up`. Apply it manually:

```bash
docker compose exec -T postgres psql -U treasury -d treasury \
  < services/api/db/migrations/create_export_audit_logs.sql
```

Without this, the server still starts, but any export attempt will fail at runtime with a database error.

### Step 5 — Verify

```bash
# API health check
curl http://localhost:34080/health
# Expected: {"status":"ok","service":"treasury-api"}

# Frontend
open http://localhost:34000

# Swagger docs
open http://localhost:34080/swagger/index.html

# MinIO console
open http://localhost:9001
# Credentials: minioadmin / minioadmin
```

---

## Environment Variables Reference

From `internal/config/config.go` and `internal/config/security.go`:

| Variable | Default in code | Notes |
|---|---|---|
| `APP_PORT` | `34080` | Backend listen port |
| `APP_ENV` | `development` | Note: `.env` mistakenly uses `SERVER_ENV` — that var is ignored by `config.go` |
| `SECURITY_LEVEL` | `development` | Controls token TTL, rate limits, cookie security |
| `DATABASE_URL` | `postgres://localhost:5432/treasury?sslmode=disable` | Required |
| `DATABASE_MAX_CONNS` | `25` | |
| `DATABASE_MIN_CONNS` | `5` | |
| `REDIS_URL` | `redis://localhost:6379` | Optional — rate limiting disabled if unreachable |
| `JWT_SECRET` | `treasury-dev-secret-change-in-production` | Change in production |
| `AUTH_MODE` | `standalone` | `standalone` or `zitadel` |
| `COOKIE_DOMAIN` | `""` | Set to your domain in production |
| `COOKIE_SAMESITE` | `Lax` | |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:34000,http://localhost:3000` | Comma-separated |
| `CORS_MAX_AGE` | `300` | |
| `MINIO_ENDPOINT` | — | Optional — export/attachments disabled if absent |
| `MINIO_ACCESS_KEY` | — | |
| `MINIO_SECRET_KEY` | — | |
| `MINIO_BUCKET` | `treasury-exports` | Auto-created by app on first export |
| `MINIO_USE_SSL` | `false` | |
| `LOG_LEVEL` | `info` (production) / `debug` (development) | Read by `internal/logger/logger.go` via `os.Getenv("LOG_LEVEL")`. Valid values: `debug`, `info`, `warn`, `error` |
| `EMAIL_HOST` | `localhost` | |
| `EMAIL_PORT` | `1025` | |
| `EMAIL_USE_TLS` | `false` | |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `localhost:4318` | Optional telemetry. Note: `.env` uses `OTEL_ENDPOINT` which is ignored — use this exact var name |

---

## Graceful Degradation

From `cmd/server/main.go` — the server starts successfully even when optional services are unavailable:

| Missing service | Behaviour |
|---|---|
| Redis unreachable | Logs warning, rate limiting disabled |
| MinIO endpoint not set | No warning logged — export and attachments silently disabled |
| MinIO endpoint set but unreachable | Logs warning at first export attempt, export fails |
| Database unreachable | Logs warning, runs in "mock mode" (no data) |

---

## Ports Summary

| Port | Service | Exposed externally |
|---|---|---|
| `34000` | Frontend (Next.js) | Yes |
| `34080` | Backend API (Go) | Yes |
| `9001` | MinIO Console | Yes (optional) |
| `5432` | PostgreSQL | No |
| `6379` | Redis | No |
| `9000` | MinIO API | No |
