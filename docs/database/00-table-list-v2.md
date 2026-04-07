# DATABASE DESIGN — TABLE LIST

# TREASURY MANAGEMENT SYSTEM — KIENLONGBANK

| Info | Detail |
|------|--------|
| **Version** | 3.0 |
| **Date** | 03/04/2026 |
| **Based on** | BRD v3.0 (02/04/2026) + Test Cases v1 + DBA Review |
| **Author** | KAI (AI Banking Assistant) |
| **Status** | Draft — Post-DBA review |

---

## DESIGN PRINCIPLES

### Banking Standards — Secure by Design

1. **Soft delete** — No hard deletes. All tables have `deleted_at` (nullable timestamp)
2. **Audit columns** — All tables: `created_at`, `created_by`, `updated_at`, `updated_by`
3. **UUID v7 primary key** — Time-sortable, prevents enumeration attacks
4. **Immutable transactions** — Deals locked after approval; changes via new records or audit log
5. **Decimal precision** — Convention table below; no floating-point
6. **Row-Level Security (RLS)** — PostgreSQL RLS policies per role
7. **Encryption at rest** — Sensitive fields flagged for column-level encryption
8. **Foreign key constraints** — Referential integrity at DB level
9. **Check constraints** — Business rules enforced at DB level (amount > 0, rate > 0)
10. **Index strategy** — Optimized for query patterns: status, date range, counterparty, module

### Naming Convention

- **Table names:** snake_case, English, standard banking/financial terminology
- **Column names:** snake_case, English — no Vietnamese abbreviations
- **Enums:** UPPER_SNAKE_CASE
- **No prefixes:** no `tbl_`, `t_`, `fk_` prefixes on table names

### User & Auth — Standalone + Zitadel Ready

- **Standalone mode:** `users` + `roles` + `user_roles` — full auth, dev can run without IdP
- **Zitadel mode:** Toggle via `auth_config`. Login via OIDC → JWT → match `users.external_id`
- **Switch:** `AUTH_MODE=standalone|zitadel` in env/config — no schema change

### Document Management — MinIO/S3 Centralized Storage

- All documents stored in **MinIO/S3** — not local filesystem
- `documents` table tracks metadata + S3 object key
- Bucket structure: `treasury-docs/{module}/{deal_id}/{filename}`
- Versioning enabled — overwrite creates new version
- Pre-signed URLs for secure download (time-limited)

### Decimal Precision Convention

| Data Type | PostgreSQL Type | Usage |
|-----------|----------------|-------|
| Amount VND | NUMERIC(20,0) | Vietnamese Dong (no decimals) |
| Amount FCY | NUMERIC(20,2) | Foreign currency amounts |
| Amount flexible | NUMERIC(20,4) | Cross-currency converted amounts |
| Interest rate | NUMERIC(10,6) | Interest rates (%/year) |
| Exchange rate | NUMERIC(20,6) | FX rates |
| Percentage | NUMERIC(5,2) | Haircut, ratio |
| Quantity | BIGINT | Bond quantity (integer) |

---

## OVERVIEW — 32 TABLES

```
📦 Treasury Database
├── 🔐 AUTH & USER (8 tables)
│   ├── users
│   ├── roles
│   ├── permissions
│   ├── role_permissions
│   ├── user_roles
│   ├── auth_configs
│   ├── external_role_mappings
│   └── user_sessions
│
├── 🏦 ORGANIZATION (1 table)
│   └── branches
│
├── 📋 MASTER DATA (7 tables)
│   ├── counterparties
│   ├── currencies
│   ├── currency_pairs
│   ├── bond_catalog
│   ├── settlement_instructions
│   ├── exchange_rates
│   └── business_calendar
│
├── 💱 MODULE 1: FX (2 tables)
│   ├── fx_deals
│   └── fx_deal_legs
│
├── 📜 MODULE 2: BOND — GTCG (2 tables)
│   ├── bond_deals
│   └── bond_inventory
│
├── 💰 MODULE 3: MONEY MARKET (2 tables)
│   ├── mm_interbank_deals
│   └── mm_omo_repo_deals
│
├── 📊 MODULE 4: CREDIT LIMIT (3 tables)
│   ├── credit_limits
│   ├── limit_utilization_snapshots
│   └── limit_approval_records
│
├── 🌐 MODULE 5: INTERNATIONAL PAYMENT (1 table)
│   └── international_payments
│
├── ✅ WORKFLOW (3 tables)
│   ├── approval_actions
│   ├── status_transition_rules
│   └── deal_sequences
│
├── 📂 DOCUMENT MANAGEMENT (1 table)
│   └── documents
│
├── 🔔 NOTIFICATION (1 table)
│   └── notifications
│
└── 📝 AUDIT (1 table)
    └── audit_logs
```

---

## TABLE DETAILS

### 🔐 AUTH & USER (8 tables)

#### 1. `users`

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID PK | Internal ID |
| `external_id` | VARCHAR(255) NULL | Zitadel user ID (when AUTH_MODE=zitadel) |
| `username` | VARCHAR(100) UNIQUE | Login username (standalone mode) |
| `password_hash` | VARCHAR(255) NULL | BCrypt hash (standalone only) |
| `full_name` | VARCHAR(255) | Full name |
| `email` | VARCHAR(255) | Corporate email |
| `branch_id` | UUID FK → branches | Branch/office assignment |
| `department` | VARCHAR(100) | Department/Division |
| `position` | VARCHAR(100) | Job title |
| `is_active` | BOOLEAN DEFAULT true | Active status |
| `last_login_at` | TIMESTAMPTZ NULL | Last login timestamp |
| `created_at` | TIMESTAMPTZ | |
| `updated_at` | TIMESTAMPTZ | |
| `deleted_at` | TIMESTAMPTZ NULL | Soft delete |

**Index:** `username`, `external_id`, `department`

#### 2. `roles`
> 10 roles per BRD v3 section 2.4.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID PK | |
| `code` | VARCHAR(50) UNIQUE | `DEALER`, `DESK_HEAD`, `CENTER_DIRECTOR`, `DIVISION_HEAD`, `RISK_OFFICER`, `RISK_HEAD`, `ACCOUNTANT`, `CHIEF_ACCOUNTANT`, `SETTLEMENT_OFFICER`, `ADMIN` |
| `name` | VARCHAR(255) | Display name |
| `description` | TEXT NULL | Permission description |
| `scope` | VARCHAR(50) | Data scope: `ALL`, `MODULE_SPECIFIC`, `STEP_SPECIFIC` |
| `created_at` | TIMESTAMPTZ | |

**Role mapping (BRD → DB):**

| BRD Role | DB Code | Description |
|----------|---------|-------------|
| CV K.NV | `DEALER` | Dealer / Trade Support |
| TP K.NV | `DESK_HEAD` | Desk Head — Level 1 approver |
| GĐ TT KDV/QLV | `CENTER_DIRECTOR` | Center Director — Level 2 approver |
| GĐ/PGĐ Khối | `DIVISION_HEAD` | Division Head — Level 2 approver + Cancel approver |
| CV QLRR | `RISK_OFFICER` | Market Risk Officer — Limit approval L1 |
| TPB QLRR | `RISK_HEAD` | Risk Department Head — Limit approval L2 |
| CV P.KTTC | `ACCOUNTANT` | Accountant — Booking L1 |
| LĐ P.KTTC | `CHIEF_ACCOUNTANT` | Chief Accountant — Booking L2 |
| BP.TTQT | `SETTLEMENT_OFFICER` | International Settlement Officer |
| Admin | `ADMIN` | System Administrator |

#### 3. `permissions`
> Granular permission definitions. Resource + action pattern.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID PK | |
| `code` | VARCHAR(100) UNIQUE | Permission code (see matrix below) |
| `resource` | VARCHAR(50) | Resource: `FX_DEAL`, `BOND_DEAL`, `MM_INTERBANK_DEAL`, `MM_OMO_REPO_DEAL`, `CREDIT_LIMIT`, `INTERNATIONAL_PAYMENT`, `MASTER_DATA`, `SYSTEM` |
| `action` | VARCHAR(30) | Action: `VIEW`, `CREATE`, `EDIT`, `APPROVE_L1`, `APPROVE_L2`, `APPROVE_RISK_L1`, `APPROVE_RISK_L2`, `BOOK_L1`, `BOOK_L2`, `SETTLE`, `RECALL`, `CANCEL_REQUEST`, `CANCEL_APPROVE_L1`, `CANCEL_APPROVE_L2`, `CLONE`, `EXPORT`, `MANAGE` |
| `description` | TEXT NULL | Human-readable description |
| `created_at` | TIMESTAMPTZ | |

**Permission matrix (seed data):**

| Permission Code | Resource | Action | Roles |
|----------------|----------|--------|-------|
| `FX_DEAL.VIEW` | FX_DEAL | VIEW | DEALER, DESK_HEAD, CENTER_DIRECTOR, DIVISION_HEAD, ADMIN |
| `FX_DEAL.CREATE` | FX_DEAL | CREATE | DEALER |
| `FX_DEAL.EDIT` | FX_DEAL | EDIT | DEALER (only when OPEN) |
| `FX_DEAL.APPROVE_L1` | FX_DEAL | APPROVE_L1 | DESK_HEAD |
| `FX_DEAL.APPROVE_L2` | FX_DEAL | APPROVE_L2 | CENTER_DIRECTOR, DIVISION_HEAD |
| `FX_DEAL.BOOK_L1` | FX_DEAL | BOOK_L1 | ACCOUNTANT |
| `FX_DEAL.BOOK_L2` | FX_DEAL | BOOK_L2 | CHIEF_ACCOUNTANT |
| `FX_DEAL.RECALL` | FX_DEAL | RECALL | DEALER, DESK_HEAD |
| `FX_DEAL.CANCEL_REQUEST` | FX_DEAL | CANCEL_REQUEST | DEALER |
| `FX_DEAL.CANCEL_APPROVE_L1` | FX_DEAL | CANCEL_APPROVE_L1 | DESK_HEAD |
| `FX_DEAL.CANCEL_APPROVE_L2` | FX_DEAL | CANCEL_APPROVE_L2 | DIVISION_HEAD |
| `FX_DEAL.CLONE` | FX_DEAL | CLONE | DEALER |
| `BOND_DEAL.VIEW` | BOND_DEAL | VIEW | DEALER, DESK_HEAD, CENTER_DIRECTOR, DIVISION_HEAD, ADMIN |
| `BOND_DEAL.CREATE` | BOND_DEAL | CREATE | DEALER |
| `BOND_DEAL.EDIT` | BOND_DEAL | EDIT | DEALER |
| `BOND_DEAL.APPROVE_L1` | BOND_DEAL | APPROVE_L1 | DESK_HEAD |
| `BOND_DEAL.APPROVE_L2` | BOND_DEAL | APPROVE_L2 | CENTER_DIRECTOR, DIVISION_HEAD |
| `BOND_DEAL.BOOK_L1` | BOND_DEAL | BOOK_L1 | ACCOUNTANT |
| `BOND_DEAL.BOOK_L2` | BOND_DEAL | BOOK_L2 | CHIEF_ACCOUNTANT |
| `BOND_DEAL.RECALL` | BOND_DEAL | RECALL | DEALER, DESK_HEAD |
| `BOND_DEAL.CANCEL_REQUEST` | BOND_DEAL | CANCEL_REQUEST | DEALER |
| `BOND_DEAL.CANCEL_APPROVE_L1` | BOND_DEAL | CANCEL_APPROVE_L1 | DESK_HEAD |
| `BOND_DEAL.CANCEL_APPROVE_L2` | BOND_DEAL | CANCEL_APPROVE_L2 | DIVISION_HEAD |
| `BOND_DEAL.CLONE` | BOND_DEAL | CLONE | DEALER |
| `MM_INTERBANK_DEAL.VIEW` | MM_INTERBANK_DEAL | VIEW | DEALER, DESK_HEAD, CENTER_DIRECTOR, DIVISION_HEAD, ADMIN |
| `MM_INTERBANK_DEAL.CREATE` | MM_INTERBANK_DEAL | CREATE | DEALER |
| `MM_INTERBANK_DEAL.EDIT` | MM_INTERBANK_DEAL | EDIT | DEALER |
| `MM_INTERBANK_DEAL.APPROVE_L1` | MM_INTERBANK_DEAL | APPROVE_L1 | DESK_HEAD |
| `MM_INTERBANK_DEAL.APPROVE_L2` | MM_INTERBANK_DEAL | APPROVE_L2 | CENTER_DIRECTOR, DIVISION_HEAD |
| `MM_INTERBANK_DEAL.APPROVE_RISK_L1` | MM_INTERBANK_DEAL | APPROVE_RISK_L1 | RISK_OFFICER |
| `MM_INTERBANK_DEAL.APPROVE_RISK_L2` | MM_INTERBANK_DEAL | APPROVE_RISK_L2 | RISK_HEAD |
| `MM_INTERBANK_DEAL.BOOK_L1` | MM_INTERBANK_DEAL | BOOK_L1 | ACCOUNTANT |
| `MM_INTERBANK_DEAL.BOOK_L2` | MM_INTERBANK_DEAL | BOOK_L2 | CHIEF_ACCOUNTANT |
| `MM_INTERBANK_DEAL.SETTLE` | MM_INTERBANK_DEAL | SETTLE | SETTLEMENT_OFFICER |
| `MM_INTERBANK_DEAL.RECALL` | MM_INTERBANK_DEAL | RECALL | DEALER, DESK_HEAD |
| `MM_INTERBANK_DEAL.CANCEL_REQUEST` | MM_INTERBANK_DEAL | CANCEL_REQUEST | DEALER |
| `MM_INTERBANK_DEAL.CANCEL_APPROVE_L1` | MM_INTERBANK_DEAL | CANCEL_APPROVE_L1 | DESK_HEAD |
| `MM_INTERBANK_DEAL.CANCEL_APPROVE_L2` | MM_INTERBANK_DEAL | CANCEL_APPROVE_L2 | DIVISION_HEAD |
| `MM_INTERBANK_DEAL.CLONE` | MM_INTERBANK_DEAL | CLONE | DEALER |
| `MM_OMO_REPO_DEAL.VIEW` | MM_OMO_REPO_DEAL | VIEW | DEALER, DESK_HEAD, CENTER_DIRECTOR, DIVISION_HEAD, ADMIN |
| `MM_OMO_REPO_DEAL.CREATE` | MM_OMO_REPO_DEAL | CREATE | DEALER |
| `MM_OMO_REPO_DEAL.APPROVE_L1` | MM_OMO_REPO_DEAL | APPROVE_L1 | DESK_HEAD |
| `MM_OMO_REPO_DEAL.APPROVE_L2` | MM_OMO_REPO_DEAL | APPROVE_L2 | CENTER_DIRECTOR, DIVISION_HEAD |
| `MM_OMO_REPO_DEAL.BOOK_L1` | MM_OMO_REPO_DEAL | BOOK_L1 | ACCOUNTANT |
| `MM_OMO_REPO_DEAL.BOOK_L2` | MM_OMO_REPO_DEAL | BOOK_L2 | CHIEF_ACCOUNTANT |
| `CREDIT_LIMIT.VIEW` | CREDIT_LIMIT | VIEW | RISK_OFFICER, RISK_HEAD, ADMIN |
| `CREDIT_LIMIT.APPROVE_RISK_L1` | CREDIT_LIMIT | APPROVE_RISK_L1 | RISK_OFFICER |
| `CREDIT_LIMIT.APPROVE_RISK_L2` | CREDIT_LIMIT | APPROVE_RISK_L2 | RISK_HEAD |
| `CREDIT_LIMIT.MANAGE` | CREDIT_LIMIT | MANAGE | ADMIN |
| `CREDIT_LIMIT.EXPORT` | CREDIT_LIMIT | EXPORT | RISK_OFFICER, RISK_HEAD, DEALER, DESK_HEAD |
| `INTERNATIONAL_PAYMENT.VIEW` | INTERNATIONAL_PAYMENT | VIEW | SETTLEMENT_OFFICER, ADMIN |
| `INTERNATIONAL_PAYMENT.SETTLE` | INTERNATIONAL_PAYMENT | SETTLE | SETTLEMENT_OFFICER |
| `MASTER_DATA.VIEW` | MASTER_DATA | VIEW | ALL ROLES |
| `MASTER_DATA.MANAGE` | MASTER_DATA | MANAGE | ADMIN |
| `AUDIT_LOG.VIEW` | SYSTEM | VIEW | DESK_HEAD, CENTER_DIRECTOR, DIVISION_HEAD, RISK_OFFICER, RISK_HEAD, ACCOUNTANT, CHIEF_ACCOUNTANT, ADMIN |

**Index:** `code`, `resource`, `action`

#### 4. `role_permissions`
> Maps roles to permissions. Granular RBAC.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID PK | |
| `role_id` | UUID FK → roles | |
| `permission_id` | UUID FK → permissions | |
| `created_at` | TIMESTAMPTZ | |
| `created_by` | UUID FK → users | |

**Unique:** `(role_id, permission_id)`
**Index:** `role_id`, `permission_id`

#### 5. `user_roles`

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID PK | |
| `user_id` | UUID FK → users | |
| `role_id` | UUID FK → roles | |
| `granted_at` | TIMESTAMPTZ | |
| `granted_by` | UUID FK → users | |

**Unique:** `(user_id, role_id)`

#### 6. `auth_configs`

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID PK | |
| `auth_mode` | VARCHAR(20) | `standalone` or `zitadel` |
| `issuer_url` | VARCHAR(500) NULL | Zitadel issuer URL |
| `client_id` | VARCHAR(255) NULL | OIDC Client ID |
| `client_secret_encrypted` | TEXT NULL | Encrypted client secret |
| `scopes` | VARCHAR(500) NULL | OIDC scopes |
| `auto_create_user` | BOOLEAN DEFAULT true | Auto-create user on first Zitadel login |
| `sync_user_info` | BOOLEAN DEFAULT true | Sync name/email from JWT claims |
| `is_active` | BOOLEAN DEFAULT true | |
| `updated_at` | TIMESTAMPTZ | |
| `updated_by` | UUID FK → users | |

**Constraint:** Application enforces single active config row

#### 7. `external_role_mappings`

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID PK | |
| `external_group` | VARCHAR(255) | Zitadel group/role name |
| `role_id` | UUID FK → roles | Internal role |
| `created_at` | TIMESTAMPTZ | |

**Unique:** `(external_group, role_id)`

#### 8. `user_sessions`
> Session management for standalone auth mode. Zitadel mode uses IdP sessions.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID PK | Session ID |
| `user_id` | UUID FK → users | |
| `token_hash` | VARCHAR(64) | SHA-256 hash of session/refresh token |
| `ip_address` | INET NULL | Client IP at login |
| `user_agent` | TEXT NULL | Browser/client info |
| `expires_at` | TIMESTAMPTZ | Token expiry |
| `revoked_at` | TIMESTAMPTZ NULL | Revoked timestamp (logout/force-expire) |
| `created_at` | TIMESTAMPTZ | |

**Index:** `user_id`, `token_hash`, `expires_at`
**Cleanup:** Periodic purge of expired sessions (> 30 days)

---

### 🏦 ORGANIZATION (1 table)

#### 9. `branches`
> Branch/office hierarchy. Phase 1: seed HEAD_OFFICE only. Ready for multi-branch expansion.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID PK | |
| `code` | VARCHAR(20) UNIQUE | Branch code (e.g. `HO`, `HCM01`, `HN01`, `KG01`) |
| `name` | VARCHAR(255) | Branch name |
| `branch_type` | VARCHAR(20) | `HEAD_OFFICE`, `BRANCH`, `SUB_BRANCH`, `TRANSACTION_OFFICE` |
| `parent_branch_id` | UUID NULL FK → branches | Parent branch (hierarchy) |
| `flexcube_branch_code` | VARCHAR(20) NULL | Flexcube Core Banking branch code (Phase 2 integration) |
| `swift_branch_code` | VARCHAR(5) NULL | SWIFT branch identifier (if applicable) |
| `address` | TEXT NULL | Branch address |
| `is_active` | BOOLEAN DEFAULT true | |
| `created_at` | TIMESTAMPTZ | |
| `created_by` | UUID FK → users | |
| `updated_at` | TIMESTAMPTZ | |
| `updated_by` | UUID FK → users | |

**Index:** `code`, `branch_type`, `parent_branch_id`
**Phase 1 seed:** `INSERT INTO branches (code, name, branch_type) VALUES ('HO', 'Hội sở KienlongBank', 'HEAD_OFFICE');`

---

### 📋 MASTER DATA (7 tables)

#### 10. `counterparties`

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID PK | |
| `code` | VARCHAR(20) UNIQUE | Internal code (e.g. MSBI, ACB) |
| `full_name` | VARCHAR(500) | Official name |
| `short_name` | VARCHAR(255) NULL | Abbreviation |
| `cif` | VARCHAR(50) | Customer Information File number |
| `swift_code` | VARCHAR(11) NULL | SWIFT/BIC (optional — some counterparties don't have) |
| `country_code` | VARCHAR(2) NULL | ISO 3166-1 alpha-2 |
| `tax_id` | VARCHAR(20) NULL | Tax ID |
| `address` | TEXT NULL | Address |
| `fx_uses_limit` | BOOLEAN DEFAULT false | Whether FX deals consume credit limit (v3 feature) |
| `is_active` | BOOLEAN DEFAULT true | |
| `created_at` | TIMESTAMPTZ | |
| `created_by` | UUID FK → users | |
| `updated_at` | TIMESTAMPTZ | |
| `updated_by` | UUID FK → users | |
| `deleted_at` | TIMESTAMPTZ NULL | |

**Index:** `code`, `cif`, `swift_code`

#### 11. `currencies`

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID PK | |
| `code` | VARCHAR(3) UNIQUE | ISO 4217: USD, VND, EUR, AUD, GBP, JPY, KRW... |
| `numeric_code` | SMALLINT NULL | ISO 4217 numeric (e.g. USD=840, VND=704) |
| `name` | VARCHAR(100) | Full name |
| `decimal_places` | SMALLINT DEFAULT 2 | Decimal precision (VND=0, USD=2, JPY=0) |
| `is_active` | BOOLEAN DEFAULT true | |

#### 12. `currency_pairs`
> Determines FX calculation formula. Config-driven — no hardcoded logic per pair.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID PK | |
| `base_currency` | VARCHAR(3) FK → currencies | Base (e.g. USD in USD/VND) |
| `quote_currency` | VARCHAR(3) FK → currencies | Quote (e.g. VND in USD/VND) |
| `pair_code` | VARCHAR(7) UNIQUE | e.g. `USD/VND`, `EUR/USD`, `EUR/GBP` |
| `rate_decimal_places` | SMALLINT | Rate precision (2 for USD/VND, USD/JPY, USD/KRW; 4 for others) |
| `calculation_rule` | VARCHAR(20) | `MULTIPLY` (USD/VND, .../USD, cross) or `DIVIDE` (USD/...) |
| `result_currency` | VARCHAR(3) FK → currencies | Output currency |
| `is_active` | BOOLEAN DEFAULT true | |

#### 13. `bond_catalog`

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID PK | |
| `bond_code` | VARCHAR(50) UNIQUE | Bond identifier (e.g. TD2135068) |
| `issuer` | VARCHAR(500) | Issuing organization |
| `coupon_rate` | NUMERIC(10,4) | Coupon rate (%/year) |
| `payment_frequency` | VARCHAR(20) NULL | Coupon frequency: ANNUAL, SEMI_ANNUAL, QUARTERLY, ZERO_COUPON (Phase 2) |
| `issue_date` | DATE | Issue date |
| `maturity_date` | DATE | Maturity date |
| `face_value` | NUMERIC(20,0) | Face value (VND, integer) |
| `bond_type` | VARCHAR(20) | `GOVERNMENT`, `FINANCIAL_INSTITUTION`, `CERTIFICATE_OF_DEPOSIT` |
| `is_active` | BOOLEAN DEFAULT true | |
| `created_at` | TIMESTAMPTZ | |
| `created_by` | UUID FK → users | |
| `updated_at` | TIMESTAMPTZ | |
| `updated_by` | UUID FK → users | |

**Index:** `bond_code`, `bond_type`, `maturity_date`

#### 14. `settlement_instructions`
> Pay code / SSI. One counterparty can have multiple SSIs, even for the same currency.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID PK | |
| `counterparty_id` | UUID FK → counterparties | |
| `currency_code` | VARCHAR(3) FK → currencies | |
| `owner_type` | VARCHAR(15) | `INTERNAL` (KLB) or `COUNTERPARTY` |
| `account_number` | VARCHAR(100) | Account number |
| `bank_name` | VARCHAR(500) | Correspondent bank name |
| `swift_code` | VARCHAR(11) NULL | Correspondent bank SWIFT |
| `citad_code` | VARCHAR(20) NULL | Citad code (domestic) |
| `description` | TEXT NULL | Full SSI text |
| `is_default` | BOOLEAN DEFAULT false | Default SSI for counterparty + currency |
| `is_active` | BOOLEAN DEFAULT true | |
| `created_at` | TIMESTAMPTZ | |
| `created_by` | UUID FK → users | |
| `updated_at` | TIMESTAMPTZ | |
| `updated_by` | UUID FK → users | |

**Index:** `counterparty_id`, `currency_code`, `owner_type`

#### 15. `exchange_rates`
> FX rates for credit limit conversion. Published end of business day.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID PK | |
| `currency_code` | VARCHAR(3) FK → currencies | e.g. USD |
| `effective_date` | DATE | Effective date (end of previous business day) |
| `buy_transfer_rate` | NUMERIC(20,4) | Buy transfer rate |
| `sell_transfer_rate` | NUMERIC(20,4) | Sell transfer rate |
| `mid_rate` | NUMERIC(20,4) | = (buy + sell) / 2 — pre-calculated |
| `source` | VARCHAR(50) | Rate source (e.g. `KLB_DAILY`, `SBV`) |
| `created_at` | TIMESTAMPTZ | |
| `created_by` | UUID FK → users | |

**Unique:** `(currency_code, effective_date)`

#### 16. `business_calendar`
> Business day calendar for settlement date validation and rate lookups.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID PK | |
| `calendar_date` | DATE UNIQUE | |
| `country_code` | VARCHAR(2) DEFAULT 'VN' | |
| `is_business_day` | BOOLEAN | |
| `holiday_name` | VARCHAR(255) NULL | Holiday description if applicable |
| `created_at` | TIMESTAMPTZ | |

**Index:** `calendar_date`, `country_code, is_business_day`

---

### 💱 MODULE 1: FX — FOREIGN EXCHANGE (2 tables)

> **Design:** Separate header (`fx_deals`) and legs (`fx_deal_legs`).
> Spot/Forward = 1 header + 1 leg. Swap = 1 header + 2 legs.

#### 17. `fx_deals`

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID PK | |
| `deal_number` | VARCHAR(30) UNIQUE | Gapless deal number (generated via deal_sequences) |
| `ticket_number` | VARCHAR(20) NULL | External ticket number (optional) |
| `counterparty_id` | UUID FK → counterparties | |
| `deal_type` | VARCHAR(10) | `SPOT`, `FORWARD`, `SWAP` |
| `direction` | VARCHAR(10) | Spot/Fwd: `SELL`, `BUY`; Swap: `SELL_BUY`, `BUY_SELL` |
| `notional_amount` | NUMERIC(20,2) | Deal amount |
| `currency_code` | VARCHAR(3) FK → currencies | Deal currency |
| `pair_code` | VARCHAR(7) FK → currency_pairs | Currency pair (auto-derived) |
| `trade_date` | DATE | Trade date |
| `branch_id` | UUID FK → branches | Originating branch |
| `uses_credit_limit` | BOOLEAN DEFAULT false | Whether this deal consumes credit limit |
| `status` | VARCHAR(30) | Deal status (see enum below) |
| `note` | TEXT NULL | Free text note |
| `cloned_from_id` | UUID NULL FK → fx_deals | Cloned from (if applicable) |
| `cancel_reason` | TEXT NULL | Cancellation reason |
| `cancel_requested_by` | UUID NULL FK → users | |
| `cancel_requested_at` | TIMESTAMPTZ NULL | |
| `created_at` | TIMESTAMPTZ | |
| `created_by` | UUID FK → users | Dealer who created |
| `updated_at` | TIMESTAMPTZ | |
| `updated_by` | UUID FK → users | |
| `deleted_at` | TIMESTAMPTZ NULL | |

**Check:** `notional_amount > 0`
**Index:** `status`, `trade_date`, `counterparty_id`, `deal_type`, `created_by`

#### 18. `fx_deal_legs`

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID PK | |
| `deal_id` | UUID FK → fx_deals | |
| `leg_number` | SMALLINT | 1 (Spot/Fwd/Swap near leg), 2 (Swap far leg) |
| `value_date` | DATE | Value date |
| `settlement_date` | DATE | Settlement date |
| `exchange_rate` | NUMERIC(20,6) | Exchange rate |
| `converted_amount` | NUMERIC(20,2) | Calculated counter-amount |
| `converted_currency` | VARCHAR(3) FK → currencies | Counter-currency |
| `internal_ssi_id` | UUID FK → settlement_instructions | KLB pay code |
| `counterparty_ssi_id` | UUID FK → settlement_instructions | Counterparty pay code |
| `requires_international_settlement` | BOOLEAN | Whether international payment is needed |
| `created_at` | TIMESTAMPTZ | |
| `updated_at` | TIMESTAMPTZ | |
| `updated_by` | UUID FK → users | |

**Unique:** `(deal_id, leg_number)`
**Check:** `exchange_rate > 0`, `leg_number IN (1, 2)`

---

### 📜 MODULE 2: BOND — SECURITIES (2 tables)

#### 19. `bond_deals`

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID PK | |
| `deal_number` | VARCHAR(30) UNIQUE | Gapless deal number (generated via deal_sequences) |
| `bond_category` | VARCHAR(30) | `GOVERNMENT`, `FINANCIAL_INSTITUTION`, `CERTIFICATE_OF_DEPOSIT` |
| `trade_date` | DATE | Trade date |
| `branch_id` | UUID FK → branches | Originating branch |
| `order_date` | DATE NULL | Order date (Govi only) |
| `value_date` | DATE | Value date |
| `direction` | VARCHAR(5) | `BUY`, `SELL` |
| `counterparty_id` | UUID FK → counterparties | |
| `transaction_type` | VARCHAR(20) | `REPO`, `REVERSE_REPO`, `OUTRIGHT`, `OTHER` |
| `transaction_type_other` | VARCHAR(255) NULL | Custom type when OTHER |
| `bond_catalog_id` | UUID NULL FK → bond_catalog | Bond reference (required for Govi) |
| `bond_code_manual` | VARCHAR(50) NULL | Manual bond code (FI/CD) |
| `issuer` | VARCHAR(500) | Issuing organization |
| `coupon_rate` | NUMERIC(10,4) | Coupon rate (%) |
| `issue_date` | DATE NULL | Issue date |
| `maturity_date` | DATE | Maturity date |
| `quantity` | BIGINT | Number of bonds (integer) |
| `face_value` | NUMERIC(20,0) | Face value per bond (VND) |
| `discount_rate` | NUMERIC(10,4) | Discount rate (%) |
| `clean_price` | NUMERIC(20,0) | Clean price (VND) |
| `settlement_price` | NUMERIC(20,0) | Settlement/dirty price (VND) |
| `total_value` | NUMERIC(20,0) | = quantity × settlement_price |
| `portfolio_type` | VARCHAR(5) NULL | `HTM`, `AFS`, `HFT` — only when direction = BUY |
| `payment_date` | DATE | Payment date |
| `remaining_tenor_days` | INT | = maturity_date − payment_date (auto-calc) |
| `confirmation_method` | VARCHAR(20) | `EMAIL`, `REUTERS`, `OTHER` |
| `confirmation_other` | VARCHAR(255) NULL | |
| `contract_prepared_by` | VARCHAR(15) | `INTERNAL`, `COUNTERPARTY` |
| `status` | VARCHAR(30) | Deal status |
| `note` | TEXT NULL | Note (purchase date + price when selling) |
| `cloned_from_id` | UUID NULL FK → bond_deals | |
| `cancel_reason` | TEXT NULL | |
| `cancel_requested_by` | UUID NULL FK → users | |
| `cancel_requested_at` | TIMESTAMPTZ NULL | |
| `created_at` | TIMESTAMPTZ | |
| `created_by` | UUID FK → users | |
| `updated_at` | TIMESTAMPTZ | |
| `updated_by` | UUID FK → users | |
| `deleted_at` | TIMESTAMPTZ NULL | |

**Check:** `quantity > 0`, `face_value > 0`, `settlement_price > 0`
**Index:** `status`, `trade_date`, `counterparty_id`, `bond_category`, `bond_catalog_id`

#### 20. `bond_inventory`
> Bond portfolio inventory. Hard block when selling exceeds available quantity.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID PK | |
| `bond_catalog_id` | UUID NULL FK → bond_catalog | |
| `bond_code` | VARCHAR(50) | Bond identifier |
| `bond_category` | VARCHAR(30) | `GOVERNMENT`, `FINANCIAL_INSTITUTION`, `CERTIFICATE_OF_DEPOSIT` |
| `portfolio_type` | VARCHAR(5) | `HTM`, `AFS`, `HFT` |
| `available_quantity` | BIGINT | Current available quantity |
| `acquisition_date` | DATE NULL | Purchase date |
| `acquisition_price` | NUMERIC(20,0) NULL | Purchase price |
| `version` | INT DEFAULT 1 | Optimistic locking version |
| `updated_at` | TIMESTAMPTZ | |
| `updated_by` | UUID FK → users | |

**Unique:** `(bond_code, bond_category, portfolio_type)`
**Check:** `available_quantity >= 0`
**Usage:** `SELECT FOR UPDATE` when updating `available_quantity` to prevent race conditions.

---

### 💰 MODULE 3: MONEY MARKET (2 tables)

#### 21. `mm_interbank_deals`

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID PK | |
| `deal_number` | VARCHAR(30) UNIQUE | Gapless deal number (generated via deal_sequences) |
| `ticket_number` | VARCHAR(20) NULL | External ticket (optional) |
| `counterparty_id` | UUID FK → counterparties | |
| `branch_id` | UUID FK → branches | Originating branch |
| `currency_code` | VARCHAR(3) FK → currencies | VND or USD |
| `internal_ssi_id` | UUID FK → settlement_instructions | KLB settlement instruction |
| `counterparty_ssi_id` | UUID FK → settlement_instructions | Counterparty settlement instruction |
| `direction` | VARCHAR(20) | `PLACE` (Gửi tiền), `TAKE` (Nhận TG), `LEND`, `BORROW` |
| `principal_amount` | NUMERIC(20,2) | Principal amount |
| `interest_rate` | NUMERIC(10,6) | Interest rate (%/year) |
| `day_count_convention` | VARCHAR(15) | `ACT_365`, `ACT_360`, `ACT_ACT` |
| `trade_date` | DATE | Trade date |
| `effective_date` | DATE | Start/effective date |
| `tenor_days` | INT | Tenor in days |
| `maturity_date` | DATE | = effective_date + tenor_days |
| `interest_amount` | NUMERIC(20,2) | Calculated interest |
| `maturity_amount` | NUMERIC(20,2) | = principal + interest |
| `has_collateral` | BOOLEAN | Whether collateralized |
| `collateral_currency` | VARCHAR(3) NULL | Collateral currency |
| `collateral_description` | TEXT NULL | Collateral value (text — no freeze integration) |
| `requires_international_settlement` | BOOLEAN | Whether international payment needed |
| `status` | VARCHAR(30) | |
| `note` | TEXT NULL | |
| `cloned_from_id` | UUID NULL FK → mm_interbank_deals | |
| `cancel_reason` | TEXT NULL | |
| `cancel_requested_by` | UUID NULL FK → users | |
| `cancel_requested_at` | TIMESTAMPTZ NULL | |
| `created_at` | TIMESTAMPTZ | |
| `created_by` | UUID FK → users | |
| `updated_at` | TIMESTAMPTZ | |
| `updated_by` | UUID FK → users | |
| `deleted_at` | TIMESTAMPTZ NULL | |

**Check:** `principal_amount > 0`, `interest_rate > 0`, `tenor_days > 0`
**Index:** `status`, `trade_date`, `counterparty_id`, `maturity_date`, `direction`
**Note:** `ACT_ACT` day count requires application-level logic to handle leap years correctly.

#### 22. `mm_omo_repo_deals`
> OMO + State Treasury Repo. Separate table — different field structure, no QLRR, no TTQT, no credit limit.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID PK | |
| `deal_number` | VARCHAR(30) UNIQUE | Gapless deal number (generated via deal_sequences) |
| `deal_subtype` | VARCHAR(15) | `OMO`, `STATE_REPO` |
| `session_name` | VARCHAR(100) | Trading session (e.g. Session 1) |
| `trade_date` | DATE | |
| `branch_id` | UUID FK → branches | Originating branch |
| `counterparty_id` | UUID FK → counterparties | OMO: fixed SBV; Repo: dropdown |
| `notional_amount` | NUMERIC(20,0) | Deal notional value (VND) |
| `bond_catalog_id` | UUID FK → bond_catalog | Bond reference |
| `winning_rate` | NUMERIC(10,6) | Winning bid rate (%/year) |
| `tenor_days` | INT | Tenor in days |
| `settlement_date_1` | DATE | First settlement date |
| `settlement_date_2` | DATE | Second settlement date |
| `haircut_pct` | NUMERIC(5,2) | Haircut percentage |
| `status` | VARCHAR(30) | |
| `note` | TEXT NULL | |
| `cloned_from_id` | UUID NULL FK → mm_omo_repo_deals | |
| `cancel_reason` | TEXT NULL | |
| `cancel_requested_by` | UUID NULL FK → users | |
| `cancel_requested_at` | TIMESTAMPTZ NULL | |
| `created_at` | TIMESTAMPTZ | |
| `created_by` | UUID FK → users | |
| `updated_at` | TIMESTAMPTZ | |
| `updated_by` | UUID FK → users | |
| `deleted_at` | TIMESTAMPTZ NULL | |

**Check:** `tenor_days > 0`, `haircut_pct >= 0`
**Index:** `status`, `trade_date`, `deal_subtype`

---

### 📊 MODULE 4: CREDIT LIMIT (3 tables)

#### 23. `credit_limits`

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID PK | |
| `counterparty_id` | UUID FK → counterparties | |
| `limit_type` | VARCHAR(20) | `COLLATERALIZED`, `UNCOLLATERALIZED` |
| `limit_amount` | NUMERIC(20,2) NULL | Limit in VND. NULL = unlimited |
| `is_unlimited` | BOOLEAN DEFAULT false | Unlimited flag |
| `effective_from` | DATE | Effective start date of this version |
| `effective_to` | DATE NULL | Effective end date (NULL = current) |
| `is_current` | BOOLEAN DEFAULT true | Flag for the currently active record |
| `expiry_date` | DATE NULL | |
| `approval_reference` | TEXT NULL | CEO approval reference info |
| `created_at` | TIMESTAMPTZ | |
| `created_by` | UUID FK → users | |
| `updated_at` | TIMESTAMPTZ | |
| `updated_by` | UUID FK → users | |

**Index:** `counterparty_id`, `limit_type`, `effective_from`, `effective_to`

#### 24. `limit_utilization_snapshots`
> Point-in-time snapshot of limit usage. Append-only.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID PK | |
| `counterparty_id` | UUID FK → counterparties | |
| `snapshot_date` | DATE | |
| `limit_type` | VARCHAR(20) | `COLLATERALIZED`, `UNCOLLATERALIZED` |
| `limit_granted` | NUMERIC(20,2) NULL | Limit amount (NULL = unlimited) |
| `utilized_opening` | NUMERIC(20,2) | Opening utilization (VND equivalent) |
| `utilized_intraday` | NUMERIC(20,2) | Intraday utilization (VND equivalent) |
| `utilized_total` | NUMERIC(20,2) | Total utilization |
| `remaining` | NUMERIC(20,2) NULL | Remaining (NULL = unlimited) |
| `fx_rate_applied` | NUMERIC(20,4) NULL | USD→VND mid rate used for conversion |
| `breakdown_detail` | JSONB NULL | Detail: MM deals + FX deals + FI Bond settlement prices |
| `created_at` | TIMESTAMPTZ | |
| `created_by` | UUID FK → users | |

**Index:** `counterparty_id`, `snapshot_date`

#### 25. `limit_approval_records`
> Per-deal limit approval history. Snapshot captured at approval time (BRD 8.3).

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID PK | |
| `deal_module` | VARCHAR(10) | `FX`, `MM` |
| `deal_id` | UUID | Referenced deal ID |
| `counterparty_id` | UUID FK → counterparties | |
| `limit_type` | VARCHAR(20) | `COLLATERALIZED`, `UNCOLLATERALIZED` |
| `deal_amount_vnd` | NUMERIC(20,2) | Deal value in VND equivalent |
| `limit_snapshot` | JSONB | Snapshot: granted, utilized, remaining at approval time |
| `risk_officer_approved_by` | UUID NULL FK → users | L1 Risk Officer |
| `risk_officer_approved_at` | TIMESTAMPTZ NULL | |
| `risk_head_approved_by` | UUID NULL FK → users | L2 Risk Head |
| `risk_head_approved_at` | TIMESTAMPTZ NULL | |
| `approval_status` | VARCHAR(20) | `PENDING`, `APPROVED`, `REJECTED` |
| `rejection_reason` | TEXT NULL | |
| `created_at` | TIMESTAMPTZ | |

**Index:** `deal_module, deal_id`, `counterparty_id`, `approval_status`

---

### 🌐 MODULE 5: INTERNATIONAL PAYMENT (1 table)

#### 26. `international_payments`
> Auto-created when deal transitions to "Pending Settlement". BRD section 3.5.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID PK | |
| `source_module` | VARCHAR(10) | `FX`, `MM` |
| `source_deal_id` | UUID | Origin deal ID |
| `source_leg_number` | SMALLINT NULL | Swap: 1 (near) or 2 (far); others: NULL |
| `ticket_display` | VARCHAR(25) | Display ticket (Swap: suffix A/B) |
| `counterparty_id` | UUID FK → counterparties | |
| `debit_account` | VARCHAR(100) | KLB nostro account (HABIB) |
| `bic_code` | VARCHAR(11) NULL | Counterparty BIC |
| `currency_code` | VARCHAR(3) FK → currencies | |
| `amount` | NUMERIC(20,2) | Transfer amount |
| `transfer_date` | DATE | Settlement date |
| `counterparty_ssi` | TEXT | Counterparty SSI details |
| `original_trade_date` | DATE | Original deal trade date |
| `approved_by_division` | VARCHAR(255) NULL | Division approver reference |
| `settlement_status` | VARCHAR(20) | `PENDING`, `APPROVED`, `REJECTED` |
| `settled_by` | UUID NULL FK → users | Settlement officer |
| `settled_at` | TIMESTAMPTZ NULL | |
| `rejection_reason` | TEXT NULL | |
| `created_at` | TIMESTAMPTZ | |

**Index:** `transfer_date`, `settlement_status`, `source_module`

---

### ✅ WORKFLOW (3 tables)

> **Design Note:** `approval_actions` serves the workflow engine, providing a queryable state of approvals for a given deal. `audit_logs` serves compliance and regulatory needs, capturing a complete, immutable history of all user actions across the system. They serve different purposes and are not redundant.

#### 27. `approval_actions`
> Immutable log of every approval/rejection/recall action. Append-only.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID PK | |
| `deal_module` | VARCHAR(10) | `FX`, `BOND`, `MM_INTERBANK`, `MM_OMO_REPO` |
| `deal_id` | UUID | |
| `action_type` | VARCHAR(30) | See action types below |
| `status_before` | VARCHAR(30) | Status before action |
| `status_after` | VARCHAR(30) | Status after action |
| `performed_by` | UUID FK → users | |
| `performed_at` | TIMESTAMPTZ | |
| `reason` | TEXT NULL | Required for reject, recall, cancel |
| `metadata` | JSONB NULL | Additional data (e.g. limit snapshot) |

**Action types:**
```
DESK_HEAD_APPROVE, DESK_HEAD_RETURN,
DIRECTOR_APPROVE, DIRECTOR_REJECT,
RISK_OFFICER_APPROVE, RISK_OFFICER_REJECT,
RISK_HEAD_APPROVE, RISK_HEAD_REJECT,
ACCOUNTANT_APPROVE, ACCOUNTANT_REJECT,
CHIEF_ACCOUNTANT_APPROVE, CHIEF_ACCOUNTANT_REJECT,
SETTLEMENT_APPROVE, SETTLEMENT_REJECT,
DEALER_RECALL, DESK_HEAD_RECALL,
CANCEL_REQUEST, CANCEL_DESK_HEAD_APPROVE, CANCEL_DESK_HEAD_REJECT,
CANCEL_DIVISION_HEAD_APPROVE, CANCEL_DIVISION_HEAD_REJECT
```

**Index:** `deal_module, deal_id`, `performed_by`, `performed_at`

#### 28. `status_transition_rules`
> Config-driven state machine. No hardcoded status logic in application.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID PK | |
| `deal_module` | VARCHAR(20) | `FX`, `BOND`, `MM_INTERBANK`, `MM_OMO_REPO` |
| `from_status` | VARCHAR(30) | Current status |
| `to_status` | VARCHAR(30) | Target status |
| `required_role` | VARCHAR(50) FK → roles.code | Role required |
| `requires_reason` | BOOLEAN DEFAULT false | Must provide reason |
| `requires_confirmation` | BOOLEAN DEFAULT false | Show confirmation popup |
| `is_active` | BOOLEAN DEFAULT true | |

**Unique:** `(deal_module, from_status, to_status, required_role)`

#### 29. `deal_sequences`
> Gapless deal number generation. Advisory lock per module+date ensures no gaps.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID PK | |
| `module` | VARCHAR(20) | `FX`, `BOND`, `MM_INTERBANK`, `MM_OMO_REPO` |
| `prefix` | VARCHAR(10) | e.g. `FX`, `G`, `F`, `MM`, `OMO`, `RK` |
| `date_partition` | DATE | Business date |
| `last_sequence` | BIGINT DEFAULT 0 | Last used sequence number |
| `updated_at` | TIMESTAMPTZ | |

**Unique:** `(module, prefix, date_partition)`
**Usage:** `SELECT ... FOR UPDATE` on row → increment → generate deal number like `FX-20260403-0001`

---

### 📂 DOCUMENT MANAGEMENT (1 table)

#### 30. `documents`
> Centralized document storage with MinIO/S3 backend. Replaces simple `attachments`.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID PK | |
| `deal_module` | VARCHAR(20) | `FX`, `BOND`, `MM_INTERBANK`, `MM_OMO_REPO` |
| `deal_id` | UUID | Referenced deal |
| `document_type` | VARCHAR(30) | `TICKET`, `CONTRACT`, `CONFIRMATION`, `SUPPORTING`, `OTHER` |
| `file_name` | VARCHAR(500) | Original filename |
| `storage_bucket` | VARCHAR(100) | S3/MinIO bucket name |
| `storage_key` | VARCHAR(1000) | S3 object key: `{module}/{deal_id}/{uuid}_{filename}` |
| `file_size` | BIGINT | Size in bytes |
| `mime_type` | VARCHAR(100) | e.g. `application/pdf` |
| `checksum_sha256` | VARCHAR(64) | SHA-256 integrity hash |
| `scan_status` | VARCHAR(20) DEFAULT 'PENDING' | Virus scan: `PENDING`, `CLEAN`, `INFECTED`, `SKIPPED` |
| `scanned_at` | TIMESTAMPTZ NULL | |
| `version` | INT DEFAULT 1 | Document version (versioning support) |
| `is_current` | BOOLEAN DEFAULT true | Latest version flag |
| `uploaded_by` | UUID FK → users | |
| `uploaded_at` | TIMESTAMPTZ | |
| `deleted_at` | TIMESTAMPTZ NULL | |

**Index:** `deal_module, deal_id`, `storage_key`
**Bucket naming:** `treasury-docs` (single bucket, prefixed by module)
**Access:** Pre-signed URL with configurable TTL (default 15 minutes)

---

### 🔔 NOTIFICATION (1 table)

#### 31. `notifications`

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID PK | |
| `recipient_id` | UUID FK → users | |
| `channel` | VARCHAR(10) | `IN_APP`, `EMAIL` |
| `event_type` | VARCHAR(50) | e.g. `DEAL_PENDING_APPROVAL`, `DEAL_REJECTED`, `DEAL_CANCELLED` |
| `title` | VARCHAR(500) | |
| `body` | TEXT | |
| `deal_module` | VARCHAR(20) NULL | |
| `deal_id` | UUID NULL | |
| `is_read` | BOOLEAN DEFAULT false | |
| `read_at` | TIMESTAMPTZ NULL | |
| `delivery_status` | VARCHAR(20) DEFAULT 'PENDING' | `PENDING`, `SENT`, `FAILED`, `RETRYING` |
| `retry_count` | SMALLINT DEFAULT 0 | |
| `last_error` | TEXT NULL | Last delivery error |
| `next_retry_at` | TIMESTAMPTZ NULL | |
| `sent_at` | TIMESTAMPTZ NULL | Email sent timestamp |
| `created_at` | TIMESTAMPTZ | |

**Index:** `recipient_id, is_read`, `created_at`

---

### 📝 AUDIT (1 table)

#### 32. `audit_logs`
> Complete action history. **Append-only — NO update, NO delete.** BRD section 8.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID PK | |
| `user_id` | UUID FK → users | Actor |
| `user_full_name` | VARCHAR(255) | Snapshot (avoid join) |
| `user_department` | VARCHAR(100) | Snapshot |
| `user_branch_code` | VARCHAR(20) | Snapshot branch code |
| `action` | VARCHAR(50) | 14 event types per BRD 8.1 |
| `deal_module` | VARCHAR(20) | `FX`, `BOND`, `MM_INTERBANK`, `MM_OMO_REPO`, `CREDIT_LIMIT`, `INTERNATIONAL_PAYMENT`, `SYSTEM` |
| `deal_id` | UUID NULL | |
| `status_before` | VARCHAR(30) NULL | |
| `status_after` | VARCHAR(30) NULL | |
| `old_values` | JSONB NULL | Previous field values (on edit) |
| `new_values` | JSONB NULL | New field values (on edit) |
| `reason` | TEXT NULL | For recall, reject, cancel |
| `ip_address` | INET NULL | Client IP |
| `user_agent` | TEXT NULL | Browser/client info |
| `performed_at` | TIMESTAMPTZ | Precise timestamp |

**NO `updated_at`, NO `deleted_at`** — append-only by design.
**Index:** `deal_module, deal_id`, `user_id`, `performed_at`, `action`
**Partition:** Monthly on `performed_at` (if needed for scale)

---

## STATUS ENUM VALUES

### FX Statuses

| Code | Display Name | Description |
|------|-------------|-------------|
| `OPEN` | Open | Draft, editable by Dealer |
| `PENDING_L2_APPROVAL` | Chờ K.NV duyệt | Desk Head approved, pending Director |
| `REJECTED` | Từ chối | Rejected by Director |
| `PENDING_BOOKING` | Chờ hạch toán | Director approved, pending Accountant |
| `PENDING_CHIEF_ACCOUNTANT` | Chờ LĐ KTTC duyệt | Accountant L1 approved, pending Chief Accountant |
| `PENDING_SETTLEMENT` | Chờ TTQT | Chief Accountant approved, pending Settlement (when applicable) |
| `COMPLETED` | Hoàn thành | Fully completed |
| `VOIDED_BY_ACCOUNTING` | Hủy giao dịch | Voided by Accounting rejection |
| `VOIDED_BY_SETTLEMENT` | Hủy giao dịch | Voided by Settlement rejection |
| `CANCELLED` | Đã hủy | Cancelled after completion |

### Bond Statuses
Same as FX but **without:** `PENDING_RISK_APPROVAL`, `PENDING_SETTLEMENT`, `VOIDED_BY_SETTLEMENT`.

### MM Interbank Statuses
All statuses including:
| `PENDING_RISK_APPROVAL` | Chờ QLRR | Pending Risk limit approval |
| `VOIDED_BY_RISK` | Hủy giao dịch | Voided by Risk rejection |

### MM OMO / State Repo Statuses
Same as Bond — no Risk, no Settlement.

---

## SUMMARY TABLE

| # | Group | Table | Description |
|---|-------|-------|-------------|
| 1 | Auth | `users` | System users |
| 2 | Auth | `roles` | 10 roles (banking standard naming) |
| 3 | Auth | `permissions` | Granular permissions (resource + action) |
| 4 | Auth | `role_permissions` | Role → permission mapping (RBAC) |
| 5 | Auth | `user_roles` | User → role assignment |
| 6 | Auth | `auth_configs` | Standalone/Zitadel toggle |
| 7 | Auth | `external_role_mappings` | Zitadel group → role mapping |
| 8 | Auth | `user_sessions` | Standalone mode session management |
| 9 | Org | `branches` | Branch/office hierarchy |
| 10 | Master | `counterparties` | Trading counterparties |
| 11 | Master | `currencies` | Currency catalog |
| 12 | Master | `currency_pairs` | Pair + calculation rules |
| 13 | Master | `bond_catalog` | Bond/security catalog |
| 14 | Master | `settlement_instructions` | SSI / Pay codes |
| 15 | Master | `exchange_rates` | FX rates for limit conversion |
| 16 | Master | `business_calendar` | Business day & holiday calendar |
| 17 | FX | `fx_deals` | FX deal header |
| 18 | FX | `fx_deal_legs` | FX deal legs (1 or 2) |
| 19 | Bond | `bond_deals` | Bond/CD deals |
| 20 | Bond | `bond_inventory` | Bond portfolio inventory |
| 21 | MM | `mm_interbank_deals` | Interbank money market |
| 22 | MM | `mm_omo_repo_deals` | OMO + State Treasury Repo |
| 23 | Limit | `credit_limits` | Granted credit limits (SCD Type 2) |
| 24 | Limit | `limit_utilization_snapshots` | Point-in-time utilization |
| 25 | Limit | `limit_approval_records` | Per-deal limit approval |
| 26 | Payment | `international_payments` | International settlements |
| 27 | Workflow | `approval_actions` | Approval/rejection history |
| 28 | Workflow | `status_transition_rules` | State machine config |
| 29 | Workflow | `deal_sequences` | Gapless deal number generator |
| 30 | Document | `documents` | MinIO/S3 document management |
| 31 | Notify | `notifications` | In-app + email notifications |
| 32 | Audit | `audit_logs` | Complete audit trail |

**Total: 32 tables**

---

## DESIGN NOTES

### Why separate `fx_deal_legs`?
- Swap has 2 legs with different rates, dates, SSIs → avoids duplicate columns
- Spot/Forward = 1 leg → simple JOIN
- Extensible for future multi-leg structures

### Why separate `mm_omo_repo_deals`?
- OMO/Repo have fundamentally different fields (session, haircut, 2 settlement dates, no principal/interest)
- Different approval flow (no Risk, no Settlement)
- Avoids complex nullable columns

### Why `status_transition_rules`?
- Config-driven state machine — change flows by updating data, not code
- Easy validation: "from status X, can role Y transition to Z?"
- Simplifies testing

### Why `currency_pairs.calculation_rule`?
- BRD v3 has 4 formula types (multiply, divide, cross pair)
- DB config → no hardcoded per-pair logic
- Adding new pairs = INSERT, not code deployment

### Why `documents` with MinIO/S3?
- Centralized storage — all modules share one document system
- Versioning — overwrite creates new version, old preserved
- Security — pre-signed URLs with TTL, no direct file access
- Scalable — object storage handles any file size/volume
- Compliance — checksum integrity, audit trail on uploads

### User Auth: Standalone ↔ Zitadel
- `AUTH_MODE=standalone`: login via `users.username` + `password_hash`, self-issued JWT
- `AUTH_MODE=zitadel`: OIDC login → match `users.external_id`, roles from `external_role_mappings`
- Zero-migration switch — config only

---

*— End of document —*

**Version:** 3.0 | **Date:** 03/04/2026 | **Status:** Draft — Post-DBA review
