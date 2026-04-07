# Treasury Management System — KienlongBank

[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)](#tech-stack)
[![Next.js](https://img.shields.io/badge/Next.js-16-black?logo=next.js)](#tech-stack)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-4169E1?logo=postgresql&logoColor=white)](#tech-stack)
[![License](https://img.shields.io/badge/License-Proprietary-red)](#license)

> Full-stack Treasury Front/Middle Office system for KienlongBank, covering FX Trading, Bonds, Money Market, Credit Limits, and International Settlement.

---

## Features

- **5 Core Modules** — FX Trading, Bonds, Money Market (Interbank/OMO/Govt Repo), Credit Limits, International Settlements
- **Real-time Dashboard** — 5 SQL aggregate views with 60-second auto-refresh
- **Multi-level Approval Workflows** — Dealer → DH → TP → DIR → Risk → KTTC
- **RBAC** — Role-based access control powered by CASL
- **Auth** — Session-based authentication with Zitadel IdP integration
- **i18n** — Vietnamese and English
- **Dark Mode** — Full dark/light theme support
- **Responsive** — Mobile-friendly layout
- **SSE Notifications** — Real-time server-sent events + email outbox
- **Excel Export** — Per-module data export
- **34 Pages** — Dashboard, module CRUD views, settings, audit logs, and more
- **Swagger API Docs** — Full RESTful API documentation

## Tech Stack

| Layer | Technology |
|-------|-----------|
| **Backend** | Go 1.26, chi router, sqlc, pgx/v5 |
| **Frontend** | Next.js 16, React 19, Tailwind CSS 4, shadcn/ui |
| **Database** | PostgreSQL (13 migrations, 32+ tables) |
| **Monorepo** | pnpm workspaces |
| **Auth** | Zitadel IdP, session-based |
| **API Docs** | Swagger / OpenAPI |

## Project Structure

```
├── app/                        # Monorepo root (pnpm workspaces)
│   ├── apps/web/               # Next.js 16 frontend (138 TSX files, 34 pages)
│   ├── packages/shared/        # Shared types & utilities
│   ├── packages/ui/            # UI component library (shadcn)
│   └── services/api/           # Go backend (157 Go files)
│       ├── migrations/         # 13 SQL migration files
│       └── internal/           # Business logic & handlers
├── cmd/                        # CLI tools
├── database/                   # Database scripts
├── docs/                       # Documentation, BRD, daily logs
├── internal/                   # Go packages (db, middleware, pkg)
├── specs/                      # API specs (Swagger)
├── references/                 # Research documents
└── models/                     # Calculation models
```

## Getting Started

### Prerequisites

- **Go** ≥ 1.26
- **Node.js** ≥ 22 LTS
- **pnpm** ≥ 9
- **PostgreSQL** ≥ 16
- **sqlc** (for code generation)

### Installation

```bash
# Clone the repository
git clone https://github.com/kienlongbank/treasury-cd.git
cd treasury-cd

# Install frontend dependencies
cd app && pnpm install
```

### Environment Variables

Copy the example env files and configure:

```bash
# Backend
cp app/services/api/.env.example app/services/api/.env

# Frontend
cp app/apps/web/.env.example app/apps/web/.env.local
```

Key variables:

| Variable | Description |
|----------|-------------|
| `DATABASE_URL` | PostgreSQL connection string |
| `ZITADEL_DOMAIN` | Zitadel IdP domain |
| `NEXT_PUBLIC_API_URL` | Backend API base URL |

### Database Migration

```bash
# Run all migrations
cd app/services/api
go run cmd/migrate/main.go up
```

### Running the Application

```bash
# Start the backend (default :8080)
cd app/services/api
go run .

# Start the frontend (default :3000)
cd app/apps/web
pnpm dev
```

The application will be available at `http://localhost:3000`.

## API Overview

The backend exposes a RESTful API documented via Swagger. Key resource groups:

| Endpoint Group | Description |
|----------------|-------------|
| `/api/v1/fx` | FX Trading — create, approve, settle foreign exchange deals |
| `/api/v1/bonds` | Bonds — bond trading, inventory management |
| `/api/v1/mm` | Money Market — Interbank, OMO, and Government Repo transactions |
| `/api/v1/credit-limits` | Credit Limits — counterparty limit management |
| `/api/v1/settlements` | International Settlements |
| `/api/v1/dashboard` | Dashboard — aggregate views and statistics |
| `/api/v1/auth` | Authentication & session management |
| `/api/v1/users` | User & role management |
| `/api/v1/notifications` | SSE notifications & email outbox |
| `/api/v1/exports` | Excel export per module |

Full API documentation is available in [`specs/`](./specs/) or at `/swagger` when the server is running.

## Modules

### FX Trading
Foreign exchange deal management — create, multi-level approve, and settle FX transactions. Supports spot, forward, and swap deal types.

### Bonds
Government and corporate bond trading with inventory tracking. Full lifecycle from deal entry through settlement with real-time position management.

### Money Market
Three sub-modules covering the interbank lending market:
- **Interbank** — Interbank deposits and loans
- **OMO** — Open Market Operations
- **Government Repo** — Government bond repurchase agreements

### Credit Limits
Counterparty credit limit management with utilization tracking and breach alerts.

### International Settlements
Cross-border payment and settlement processing integrated with SWIFT messaging.

## Pages

| Module | Pages |
|--------|-------|
| **Dashboard** | Real-time overview with 5 aggregate views |
| **FX Trading** | List, Detail, Create, Edit |
| **Bonds** | List, Detail, Create, Edit, Inventory |
| **Money Market** | List + 3 sub-modules (Detail, Create each) |
| **Credit Limits** | Management view |
| **Settlements** | List, Detail |
| **Settings** | Users, Roles, Counterparties, Audit Logs |
| **Other** | Login, Notifications, Profile, Exports |

## Screenshots

> _Screenshots coming soon._

<!--
![Dashboard](docs/screenshots/dashboard.png)
![FX Trading](docs/screenshots/fx-trading.png)
![Dark Mode](docs/screenshots/dark-mode.png)
-->

## Contributing

This is a proprietary project for KienlongBank. Contributions are limited to authorized team members.

1. Create a feature branch from `master`
2. Follow existing code conventions
3. Ensure all migrations are reversible
4. Submit a pull request for review

## License

**Proprietary** — All rights reserved. This software is the property of KienlongBank. Unauthorized copying, distribution, or modification is strictly prohibited.

---

<sub>Built with Go + Next.js + PostgreSQL | Deployed on [treasury.xdigi.cloud](https://treasury.xdigi.cloud)</sub>
