# CLAUDE.md — Treasury Management System

## Project Overview

KienlongBank Treasury Management System — Front Office + Middle Office for FX, Bond, Money Market, Settlement, and Risk Management.

- **Backend:** Go 1.26, chi router, PostgreSQL, pgx, sqlc, JWT (HTTP-only cookies)
- **Frontend:** Next.js 15, React 19, Tailwind CSS 4, shadcn/ui, TanStack Query + Table, Zustand
- **Monorepo:** pnpm workspaces + Turborepo (`apps/web`, `packages/shared`, `packages/ui`, `services/api`)

## Code Language Rules

### English in Code
- All variable names, function names, type names, file names → English
- Use banking terminology: FX, Bond, Money Market, Settlement, Risk Management
- **NEVER** use Vietnamese abbreviations in code: ~~gtcg~~, ~~ttqt~~, ~~cctg~~, ~~qlrr~~, ~~knv~~, ~~kttc~~

### Vietnamese in UI
- All user-facing text MUST have full diacritics: "Giao dịch" not "Giao dich"
- Use i18n system (`src/lib/i18n.tsx`) — every visible string must be translated vi + en
- No exceptions: mock data, placeholder text, error messages — all must have diacritics

## Frontend Architecture

### Component Organization
- Feature components: `src/app/{feature}/components/` (e.g., `fx/components/fx-status-badge.tsx`)
- Shared components: `src/components/` (data-table, pagination, etc.)
- UI primitives: `src/components/ui/` (shadcn)
- Hooks: `src/hooks/use-{feature}.ts`

### Responsive Design
- **Desktop (md+):** TanStack Table with full columns
- **Mobile (<md):** Card view — single column layout
- **Card view rules:**
  - Single column only — NO 2-column grid on mobile (insufficient width)
  - Hide secondary info (email, last login, dates) — show only essential: name, status, key metric
  - Combine info on one line with separators: "Spot · Mua · 100,000 USD"
  - Use `<Link>` not `<button onClick>` for navigation (accessibility, right-click, SEO)

### Breadcrumb
- Desktop: show full path (parent > current)
- Mobile: hide parent, show only current page name (prevent 2-line wrapping)
- Implementation: `className="hidden md:block"` on parent breadcrumb items

### Dialog vs Page
- **Dialog OK for:** quick confirmations (delete, lock/unlock), short forms (1-2 fields + reason), assign role, revoke role, any action needing reason + confirm. Dialog works well on mobile too.
- **Dedicated page for:** full CRUD forms with 5+ fields (create user, edit user, create counterparty)
- **Rule of thumb:** 1-3 fields → Dialog. 5+ fields → Page. Dialog is preferred when it keeps user in context.

### Error Handling
- **Form validation errors:** inline field-level errors + Alert component at top of form
- **Business rule violations (API 422):** inline Alert with specific error message
- **Success actions:** toast notification (sonner)
- **Never** rely solely on toast for errors — always show inline Alert for visibility

### Data Patterns
- Use React Query (TanStack Query) for all API calls
- Hooks pattern: `use{Feature}.ts` with `useQuery` for reads, `useMutation` for writes
- API client: `src/lib/api.ts` — wrapper around fetch with cookie credentials
- Auth store: Zustand with sessionStorage persist

### Permission-Based UI
- Check `user.permissions[]` from auth store
- Hide/disable buttons based on permissions
- Use CASL ability hook (`use-ability.tsx`) for complex checks
- Admin routes: only visible in sidebar if user has `SYSTEM.MANAGE` or `AUDIT_LOG.VIEW`

### Dark Mode
- All components must work in both light and dark themes
- Use Tailwind `dark:` variants or CSS variables
- Test both modes before committing

### Number & Date Formatting
- Amounts: comma separator (1,000,000.00) — use `toLocaleString("en-US")`
- Dates: dd/mm/yyyy format — use `formatDate()` from `src/lib/utils`
- Currency: always show 3-letter code (USD, VND, EUR)

## Backend Architecture

### Module Pattern
Each module follows: `handler.go` → `service.go` → `repository.go`
- Handler: HTTP layer, parsing, validation, response formatting
- Service: business logic, authorization, audit logging
- Repository: database queries (interface-based for testing)

### Audit Logging (Banking-Grade)
- **EVERY write operation** must create an audit_log entry
- Required fields: user_id, user_full_name, action, deal_module, old_values, new_values, reason, ip_address
- Use shared audit helper: `pkg/audit/logger.go`
- Audit logs are append-only — never update or delete
- Lock/Unlock/Reset Password: always require `reason` field

### HTTP Status Codes
- 200: OK (successful read/update)
- 201: Created (successful create)
- 204: No Content (successful delete)
- 400: Validation Error (malformed request, missing fields)
- 401: Unauthorized (no auth, expired token)
- 403: Forbidden (insufficient permissions)
- 404: Not Found
- 409: Conflict (optimistic locking version mismatch, edit after approval)
- 422: Unprocessable Entity (business rule violation — invalid state transition)
- 429: Rate Limited

### Permission System
- RBAC defined in `pkg/constants/permissions.go` — single source of truth
- 10 roles × 80+ permissions
- Middleware: `RequirePermission()`, `RequireAnyPermission()`
- Self-approve prevention in service layer
- Status transition rules in `pkg/security/rbac.go`

### Database
- PostgreSQL with pgx driver
- sqlc for type-safe queries (generated in `../../internal/db/`)
- SQL Views for performance: `v_user_with_roles`, `v_audit_log_summary`
- Partitioned audit_logs table (by month)
- Seed data prefix: `SEED-`, use `ON CONFLICT DO NOTHING`

### Testing
- Integration tests for each module: `internal/{module}/*_test.go`
- Python E2E tests: `scripts/e2e_test.py` (79 test cases)
- Playwright browser tests for frontend
- Always run tests before committing

## API Conventions

### Response Format
```json
{
  "success": true,
  "data": { ... },
  "meta": { "request_id": "uuid", "timestamp": "ISO8601" }
}
```

### Paginated Response
```json
{
  "success": true,
  "data": {
    "data": [...],
    "total": 100,
    "page": 1,
    "page_size": 20,
    "total_pages": 5,
    "has_more": true
  }
}
```

### Error Response
```json
{
  "success": false,
  "error": { "code": "VALIDATION_ERROR", "message": "..." },
  "meta": { "request_id": "uuid", "timestamp": "ISO8601" }
}
```

## Git & Deployment

- Backend binary: `services/api/bin/treasury-api`
- Build: `go build -o bin/treasury-api ./cmd/server/`
- Frontend build: `pnpm build` (from app/ root)
- PM2 processes: `treasury-api` (port 34001), `treasury-web` (port 34000)
- Tunnel: `treasury.xdigi.cloud`

## Common Mistakes to Avoid

1. **Table links:** Use `<Link href>` not `<button onClick>` for row navigation
2. **Password in seed:** Verify bcrypt hash matches actual password before committing
3. **API port:** Backend runs on 34080, frontend on 34000. `.env.local` should NOT hardcode `NEXT_PUBLIC_API_URL` to localhost — use relative `/api/v1` so tunnel works
4. **CanRecall():** Allow recall from ALL pending states, not just PENDING_L2
5. **Lock/Unlock API:** Requires `{ "reason": "..." }` body — banking audit requirement
6. **Audit stats:** Requires `date_from` + `date_to` params
7. **Reset password:** Response field is `temp_password` (not `temporary_password`)
8. **Mobile cards:** Never use 2-column grid — single column only
9. **Breadcrumb mobile:** Hide parent items to prevent wrapping
10. **Test order:** Run destructive tests (lock/reset) last to avoid cascading failures

## Lessons Learned — 04/04/2026

### Frontend API Client
11. **204 No Content → don't call `response.json()`:** PUT/DELETE returning 204 has no body. Calling `.json()` on empty body throws `SyntaxError: The string did not match the expected pattern`. Fix: check `response.status === 204` → return undefined.
12. **Auto refresh token flow:** Access token hết hạn → 401 → call `POST /auth/refresh` (refresh cookie HttpOnly auto-sent) → retry original request. Use single `refreshPromise` to prevent concurrent refresh floods. Skip retry for `/auth/*` endpoints to avoid infinite loops.
13. **NEXT_PUBLIC_API_URL must be relative for tunnel:** Hardcode `http://localhost:34080` → works locally but fails via Cloudflare tunnel (browser can't reach localhost). Use relative `/api/v1` + Next.js rewrites to proxy to backend.

### base-ui Select (NOT radix)
14. **SelectValue shows UUID instead of label:** base-ui `SelectValue` resolves display text from **mounted** `SelectItem` children. When items load async (branches, counterparties), select already has value (UUID) but items haven't mounted yet → shows raw UUID. Fix: use render function `<SelectValue>{(value) => lookup(value)}</SelectValue>` + `label` prop on SelectItem.
15. **SelectTrigger default `w-fit`:** base-ui SelectTrigger defaults to `w-fit` → too narrow on mobile. Override to `w-full` in component, use `sm:w-auto` for desktop filters.
16. **SelectContent `w-(--anchor-width)`:** Dropdown same width as trigger → long items get cut off. Use `min-w-(--anchor-width) w-auto` so dropdown expands for longer content.

### base-ui Checkbox
17. **`<label>` wrapping Checkbox causes double-toggle:** base-ui Checkbox inside `<label>` → clicking label fires event twice (label propagation + native checkbox) → state toggles and toggles back → no change detected → Save button stays disabled. Fix: use `<div onClick>` with guard `if (e.target.closest('[data-slot="checkbox"]')) return` to prevent double-fire.

### Permission System
18. **Constants ≠ DB = silent bugs:** Go `constants.RolePermissions` map and DB `role_permissions` table were two separate sources of truth. `GetRolePermissions` read from constants, `UpdateRolePermissions` wrote to DB → they diverged. Fix: auto-sync on startup (`syncPermissions`) — Go constants upsert to DB, `GetRolePermissions` reads from DB.
19. **DB constraints block new resources:** Adding `MM_OMO_REPO_DEAL` permissions in Go constants but DB `CHECK (resource IN (...))` didn't include it → sync fails. Always update DB constraints when adding new resources. Better: auto-sync function handles constraint updates.
20. **Refresh cookie path too narrow:** `Path: /api/v1/auth/refresh` → logout request at `/api/v1/auth/logout` doesn't send refresh cookie → can't revoke token. Fix: set path to `/api/v1/auth` to cover all auth endpoints.

### State Management
21. **useMemo with Set state doesn't detect changes properly for isDirty:** React compares Set by reference, but `useMemo` deps work correctly since `setSelected` creates new Set each time. The real issue was double-toggle from label wrapper (see #17).
22. **Save → setInitialized(false) → stale cache race:** After save, `setInitialized(false)` triggers useEffect which runs immediately with **cached** (old) query data before refetch completes → reverts UI to old state. Fix: derive sync from `currentPermsKey` (joined string) so useEffect only runs when actual data changes from refetch.

### Content-Type Detection
23. **`http.DetectContentType` returns params:** Go's `http.DetectContentType` returns `text/plain; charset=utf-8` but allowed types map only has `text/plain` → upload rejected. Fix: strip `;` params before lookup.

### Deployment
24. **Always rebuild Go binary before PM2 restart:** `pm2 restart` only restarts the process — it runs the OLD binary. Must `go build -o bin/treasury-api ./cmd/server/` first.
25. **Playwright `waitUntil: 'networkidle'` hangs with SSE:** SSE notification stream keeps connection open forever → `networkidle` never fires → timeout. Use `waitUntil: 'domcontentloaded'` + explicit `waitForTimeout` instead.
