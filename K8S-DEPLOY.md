# Kubernetes Deployment Guide — Treasury Management System (AWS)

> Uses AWS RDS Aurora PostgreSQL, AWS ElastiCache Redis, and AWS S3 (via minio-go S3-compatible client).
> All env var names verified from `internal/config/config.go`, `internal/config/security.go`, and `cmd/server/main.go`.

---

## Prerequisites

- Kubernetes cluster (EKS or any k8s 1.25+)
- `kubectl` configured
- Docker images built and pushed to ECR or your registry
- AWS RDS Aurora PostgreSQL cluster endpoint
- AWS ElastiCache Redis endpoint
- AWS S3 bucket created

---

## Image Build & Push

Apply Change 1 to `next.config.ts` first (see `DOCKER-DEPLOY.md` — the rewrite fix), then:

```bash
cd /Users/tulna/Downloads/treasury/app

# API server image
docker build -t your-registry/treasury-api:latest ./services/api

# Dedicated migrate image — bakes migration files in, no ConfigMap needed
docker build -t your-registry/treasury-migrate:latest \
  -f services/api/Dockerfile.migrate ./services/api

# Web image
docker build -t your-registry/treasury-web:latest -f Dockerfile.web .

docker push your-registry/treasury-api:latest
docker push your-registry/treasury-migrate:latest
docker push your-registry/treasury-web:latest
```

**`app/services/api/Dockerfile.migrate`** (create this file):

```dockerfile
FROM migrate/migrate:latest
COPY migrations /migrations
ENTRYPOINT ["/migrate"]
```

> Migration files are baked into the image at build time. When new migration files are added (014, 015, ...), rebuild and push `treasury-migrate` — no ConfigMap or Job changes needed.

---

## Namespace

```bash
kubectl create namespace treasury
```

---

## Secrets

```bash
kubectl create secret generic treasury-secrets -n treasury \
  --from-literal=DATABASE_URL="postgres://treasury:<password>@<rds-cluster-endpoint>:5432/treasury?sslmode=require" \
  --from-literal=REDIS_URL="redis://<elasticache-endpoint>:6379" \
  --from-literal=JWT_SECRET="change_me_in_production" \
  --from-literal=MINIO_ACCESS_KEY="<aws-iam-access-key-id>" \
  --from-literal=MINIO_SECRET_KEY="<aws-iam-secret-access-key>"
```

> `DATABASE_URL` is read directly by `internal/database/database.go` via `pgxpool.ParseConfig`.
> `REDIS_URL` is read by `internal/config/security.go` as `env("REDIS_URL", "redis://localhost:6379")`.
> `MINIO_ACCESS_KEY` / `MINIO_SECRET_KEY` are read by `cmd/server/main.go` via `os.Getenv`.

---

## ConfigMap

```yaml
# configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: treasury-config
  namespace: treasury
data:
  APP_ENV: "production"
  APP_PORT: "34080"
  SECURITY_LEVEL: "production"
  AUTH_MODE: "standalone"
  COOKIE_SAMESITE: "Lax"
  COOKIE_DOMAIN: "your-domain.com"
  CORS_ALLOWED_ORIGINS: "https://your-domain.com"
  LOG_LEVEL: "info"
  # S3 — minio-go is S3-compatible, set endpoint to AWS S3 regional endpoint
  MINIO_ENDPOINT: "s3.<aws-region>.amazonaws.com"
  MINIO_BUCKET: "treasury-exports"
  MINIO_USE_SSL: "true"
  # Optional OTLP telemetry (correct var name — not OTEL_ENDPOINT)
  # OTEL_EXPORTER_OTLP_ENDPOINT: "http://otel-collector:4318"
```

> **S3 note:** The app uses `minio-go/v7` with `credentials.NewStaticV4(accessKey, secretKey, "")` and `minio.New(endpoint, ...)`. This is fully S3-compatible. Set `MINIO_ENDPOINT` to the regional S3 endpoint (e.g. `s3.ap-southeast-1.amazonaws.com`), `MINIO_USE_SSL=true`, and provide IAM credentials via `MINIO_ACCESS_KEY` / `MINIO_SECRET_KEY`.

> **`ensureBucket` on S3:** The app calls `client.BucketExists` + `client.MakeBucket` at export time (inside `Engine.Execute`), not at startup. If the IAM user lacks `s3:CreateBucket`, pre-create the bucket manually and ensure the IAM policy includes `s3:GetBucketLocation`, `s3:ListBucket`, `s3:PutObject`, `s3:GetObject`.

---

## Migrations — Init Container (production-grade)

Migration files are baked into `treasury-migrate` image. The init container runs `migrate up` before the API container starts on every deploy. It is idempotent — `migrate` tracks applied versions in the `schema_migrations` table and only runs new files.

The migrate init container is defined directly in `api.yaml` — see the API Deployment section below. No separate Job or ConfigMap required.

---

## API Deployment

The init container runs `migrate up` before the API starts. If migration fails, the API pod never starts.

```yaml
# api.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: treasury-api
  namespace: treasury
spec:
  replicas: 2
  selector:
    matchLabels:
      app: treasury-api
  template:
    metadata:
      labels:
        app: treasury-api
    spec:
      initContainers:
        - name: migrate
          image: your-registry/treasury-migrate:latest
          args:
            - "-path=/migrations"
            - "-database=$(DATABASE_URL)"
            - "up"
          env:
            - name: DATABASE_URL
              valueFrom:
                secretKeyRef:
                  name: treasury-secrets
                  key: DATABASE_URL
      containers:
        - name: api
          image: your-registry/treasury-api:latest
          ports:
            - containerPort: 34080
          envFrom:
            - configMapRef:
                name: treasury-config
          env:
            - name: DATABASE_URL
              valueFrom:
                secretKeyRef:
                  name: treasury-secrets
                  key: DATABASE_URL
            - name: REDIS_URL
              valueFrom:
                secretKeyRef:
                  name: treasury-secrets
                  key: REDIS_URL
            - name: JWT_SECRET
              valueFrom:
                secretKeyRef:
                  name: treasury-secrets
                  key: JWT_SECRET
            - name: MINIO_ACCESS_KEY
              valueFrom:
                secretKeyRef:
                  name: treasury-secrets
                  key: MINIO_ACCESS_KEY
            - name: MINIO_SECRET_KEY
              valueFrom:
                secretKeyRef:
                  name: treasury-secrets
                  key: MINIO_SECRET_KEY
          readinessProbe:
            httpGet:
              path: /health
              port: 34080
            initialDelaySeconds: 5
            periodSeconds: 10
          livenessProbe:
            httpGet:
              path: /health
              port: 34080
            initialDelaySeconds: 15
            periodSeconds: 20
---
apiVersion: v1
kind: Service
metadata:
  name: treasury-api
  namespace: treasury
spec:
  selector:
    app: treasury-api
  ports:
    - port: 34080
      targetPort: 34080
```

---

## Web Deployment

```yaml
# web.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: treasury-web
  namespace: treasury
spec:
  replicas: 2
  selector:
    matchLabels:
      app: treasury-web
  template:
    metadata:
      labels:
        app: treasury-web
    spec:
      containers:
        - name: web
          image: your-registry/treasury-web:latest
          ports:
            - containerPort: 34000
          env:
            - name: NODE_ENV
              value: production
            # Server-side proxy destination — requires next.config.ts fix (DOCKER-DEPLOY.md Change 1)
            - name: API_INTERNAL_URL
              value: http://treasury-api:34080
          readinessProbe:
            httpGet:
              path: /
              port: 34000
            initialDelaySeconds: 10
            periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: treasury-web
  namespace: treasury
spec:
  selector:
    app: treasury-web
  ports:
    - port: 34000
      targetPort: 34000
```

---

## Ingress

```yaml
# ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: treasury-ingress
  namespace: treasury
  annotations:
    nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
    # Required for SSE (/api/v1/notifications/stream keeps connection open)
    nginx.ingress.kubernetes.io/proxy-buffering: "off"
spec:
  rules:
    - host: your-domain.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: treasury-web
                port:
                  number: 34000
```

---

## Apply Order

```bash
kubectl apply -f configmap.yaml
kubectl apply -f api.yaml
kubectl apply -f web.yaml
kubectl apply -f ingress.yaml
```

> No separate migration Job or `kubectl wait` needed. The init container in `api.yaml` handles migrations automatically before the API starts.

---

## Post-Deploy: Seed Data & Export Table

This guide targets AWS RDS — there is no postgres pod in the cluster. Run these from a bastion host or a temporary pod that has `psql` and network access to the RDS endpoint.

**Option A — temporary psql pod:**

```bash
kubectl run psql-seed --rm -it --restart=Never -n treasury \
  --image=postgres:16-alpine \
  --env="PGPASSWORD=<your-db-password>" \
  -- psql -h <rds-endpoint> -U treasury -d treasury
```

Then paste the contents of the seed files interactively, or use `--command` with a heredoc.

**Option B — from a bastion host with psql installed:**

```bash
export PGPASSWORD=<your-db-password>

# Apply base seed (roles, permissions, branches, currencies, admin user)
psql -h <rds-endpoint> -U treasury -d treasury \
  < app/services/api/migrations/seed/001_seed.sql

# Apply export_audit_logs table — NOT in numbered migrations, must be manual
psql -h <rds-endpoint> -U treasury -d treasury \
  < app/services/api/db/migrations/create_export_audit_logs.sql
```

> Without `export_audit_logs`, the server starts fine but any export attempt will fail at runtime with a database error.

---

## Key Notes

| Topic | Detail |
|---|---|
| Migrations | Baked into `treasury-migrate` image. Init container runs `migrate up` before API starts on every deploy. Idempotent — only new files run. Adding 014, 015... just requires rebuilding the image |
| RDS Aurora | Use `sslmode=require` in `DATABASE_URL`. Aurora PostgreSQL is fully compatible with `pgx/v5` |
| ElastiCache Redis | App reads `REDIS_URL` from `security.go`. If ElastiCache has no auth, use `redis://<endpoint>:6379`. With auth token: `redis://:<token>@<endpoint>:6379` |
| S3 as MinIO | App uses `minio-go/v7` with `credentials.NewStaticV4` — fully S3-compatible. Set `MINIO_ENDPOINT=s3.<region>.amazonaws.com` and `MINIO_USE_SSL=true` |
| S3 IAM permissions | Minimum: `s3:GetBucketLocation`, `s3:ListBucket`, `s3:PutObject`, `s3:GetObject`, `s3:DeleteObject`. Add `s3:CreateBucket` or pre-create the bucket |
| `export_audit_logs` | Not in numbered migrations — apply `db/migrations/create_export_audit_logs.sql` manually once on first deploy. Server starts without it but exports fail at runtime |
| Seed data | Not applied by migrate — apply `migrations/seed/001_seed.sql` manually once on first deploy |
| SSE | Requires `proxy-buffering: off` on ingress — `/api/v1/notifications/stream` keeps connection open |
| `API_INTERNAL_URL` | Must be set on web pod — requires `next.config.ts` fix (see `DOCKER-DEPLOY.md`) |
| Health endpoint | `GET /health` → `{"status":"ok","service":"treasury-api"}` |
| `SECURITY_LEVEL=production` | Sets `CookieSecure: true`, 15m access token TTL, 7d refresh token TTL, strict rate limits |
| `APP_ENV` not `SERVER_ENV` | `config.go` reads `APP_ENV`. Dev `.env` has `SERVER_ENV` which is silently ignored |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Correct var name per `config.go`. Dev `.env` has `OTEL_ENDPOINT` which is ignored |
