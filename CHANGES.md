# Changes Report

## Files Added

| File | Purpose |
|---|---|
| `app/services/api/Dockerfile.migrate` | Dedicated migrate image — bakes migration files into `migrate/migrate:latest`, used as init container in K8s |
| `.github/workflows/build.yml` | GitHub Actions CI — builds 3 images in parallel (`treasury-api`, `treasury-migrate`, `treasury-web`) |
| `DOCKER-DEPLOY.md` | Docker Compose deployment guide |
| `K8S-DEPLOY.md` | Kubernetes (AWS) deployment guide |
| `BUILD-ISSUES.md` | Root cause analysis of all Docker build failures and fixes |

---

## Files Modified

| File | Change |
|---|---|
| `app/apps/web/next.config.ts` | 1. Added `turbopack.resolveAlias` to fix Turbopack CSS `@import` resolution for `tw-animate-css` and `shadcn/tailwind.css` in pnpm monorepo. 2. Applied `API_INTERNAL_URL` runtime env var to the rewrite proxy so the web container can reach the API container |
| `app/apps/web/package.json` | Reverted to plain `next build` (no side effects) |
| `app/.gitignore` | Added `tsconfig.tsbuildinfo`, `app/services/api/server`, `app/services/api/fx.test`, `app/db-dumps/*.zip`, `app/db-dumps/*.sql` |

---

## Files Deleted

| File | Reason |
|---|---|
| `app/apps/web/tsconfig.tsbuildinfo` | Stale TypeScript incremental build cache — causes Next.js build failures on clean CI |
| `app/services/api/server` | 52MB pre-built Go binary committed to repo — should never be in version control |
| `app/db-dumps/treasury_20260406_0935.zip` | Database dump committed to repo — should not be in version control |
| `app/.npmrc` | Added during debugging, not needed — `--shamefully-hoist` passed directly in `Dockerfile.web` |
