# Treasury System — KienlongBank

Hệ thống Treasury Front Office cho KienlongBank. Monorepo quản lý các module nghiệp vụ ngân quỹ:

- **FX** — Kinh doanh Ngoại tệ
- **GTCG** — Giấy tờ có giá
- **MM** — Money Market (Thị trường tiền tệ)
- **Limits** — Quản lý Hạn mức
- **TTQT** — Thanh toán Quốc tế

## Cấu trúc

```
apps/web/        → Next.js 15 — Treasury Front Office UI
packages/ui/     → Shared UI components
packages/shared/ → Shared types & utilities
services/api/    → Go API server (chi router, PostgreSQL)
```

## Yêu cầu

- Node.js >= 20
- pnpm >= 9.15
- Go >= 1.24
- PostgreSQL >= 16

## Phát triển

```bash
# Cài đặt dependencies
pnpm install

# Chạy toàn bộ stack
make dev

# Chỉ chạy web
pnpm --filter @treasury/web dev

# Chỉ chạy API
make api-dev
```

## Build

```bash
make build
```
