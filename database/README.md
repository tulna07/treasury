# Treasury Database — sqlc Setup

Database layer cho Treasury Management System sử dụng [sqlc](https://sqlc.dev/) để generate type-safe Go code từ SQL.

## Cài đặt

### sqlc

```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

### golang-migrate (cho database migrations)

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

## Generate Go code

```bash
cd database
sqlc generate
```

Output sẽ được tạo trong `internal/db/` (xem `sqlc.yaml` để cấu hình).

## Cấu trúc thư mục

```
database/
├── README.md               # File này
├── sqlc.yaml               # Cấu hình sqlc
├── schema/                 # DDL — định nghĩa bảng
│   ├── 001_auth.sql        # users, roles, permissions, sessions, user_roles, role_permissions
│   ├── 002_organization.sql # branches
│   ├── 003_master_data.sql  # counterparties, currencies, currency_pairs, bond_catalog,
│   │                        # settlement_instructions, exchange_rates, business_calendar
│   ├── 004_fx.sql           # fx_deals, fx_deal_legs
│   ├── 005_bond.sql         # bond_deals, bond_inventory
│   ├── 006_money_market.sql # mm_interbank_deals, mm_omo_repo_deals
│   ├── 007_credit_limit.sql # credit_limits, limit_utilization_snapshots, limit_approval_records
│   ├── 008_international_payment.sql # international_payments
│   ├── 009_workflow.sql     # deal_sequences, approval_actions, status_transition_rules
│   ├── 010_document.sql     # documents
│   ├── 011_notification.sql # notifications
│   └── 012_audit.sql        # audit_logs (partitioned by month)
├── queries/                # DML — truy vấn (sqlc annotations)
│   ├── users.sql
│   ├── roles.sql
│   ├── permissions.sql
│   ├── sessions.sql
│   ├── branches.sql
│   ├── counterparties.sql
│   ├── currencies.sql
│   ├── currency_pairs.sql
│   ├── bond_catalog.sql
│   ├── settlement_instructions.sql
│   ├── exchange_rates.sql
│   ├── business_calendar.sql
│   ├── fx_deals.sql
│   ├── fx_deal_legs.sql
│   ├── bond_deals.sql
│   ├── bond_inventory.sql
│   ├── mm_interbank_deals.sql
│   ├── mm_omo_repo_deals.sql
│   ├── credit_limits.sql
│   ├── limit_snapshots.sql
│   ├── limit_approvals.sql
│   ├── international_payments.sql
│   ├── deal_sequences.sql
│   ├── approval_actions.sql
│   ├── status_transitions.sql
│   ├── documents.sql
│   ├── notifications.sql
│   └── audit_logs.sql
└── migrations/             # golang-migrate files (tạo riêng)
```

## Thêm query mới

1. Mở file `.sql` tương ứng trong `queries/`
2. Thêm query với annotation sqlc:

```sql
-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1 AND deleted_at IS NULL;
```

3. Chạy `sqlc generate` để tạo lại Go code
4. Import và sử dụng trong application code

### Annotation types

| Annotation | Mô tả | Go return type |
|-----------|-------|----------------|
| `:one` | Trả về 1 row | `(Model, error)` |
| `:many` | Trả về nhiều rows | `([]Model, error)` |
| `:exec` | Không trả về data | `error` |
| `:execrows` | Trả về số rows affected | `(int64, error)` |
| `:execresult` | Trả về sql.Result | `(sql.Result, error)` |

## Type Mappings

| PostgreSQL | Go Type | Ghi chú |
|-----------|---------|---------|
| `UUID` | `uuid.UUID` | `github.com/google/uuid` |
| `DECIMAL/NUMERIC` | `pgtype.Numeric` | `github.com/jackc/pgx/v5/pgtype` — tránh float |
| `JSONB` | `json.RawMessage` hoặc custom type | Cấu hình trong `sqlc.yaml` |
| `TIMESTAMPTZ` | `time.Time` | Standard library |
| `INET` | `netip.Addr` | `net/netip` |
| `TEXT[]` | `[]string` | pgx auto-maps |
| `BOOLEAN` | `bool` | |
| `BIGINT` | `int64` | |
| `SMALLINT` | `int16` | |

### Cấu hình type override trong sqlc.yaml

```yaml
overrides:
  - db_type: "uuid"
    go_type:
      import: "github.com/google/uuid"
      type: "UUID"
  - db_type: "numeric"
    go_type:
      import: "github.com/shopspring/decimal"
      type: "Decimal"
  - db_type: "jsonb"
    go_type:
      import: "encoding/json"
      type: "RawMessage"
  - db_type: "inet"
    go_type:
      import: "net/netip"
      type: "Addr"
```

## Migration Strategy

Khuyến nghị sử dụng [golang-migrate](https://github.com/golang-migrate/migrate):

### Tạo migration mới

```bash
migrate create -ext sql -dir database/migrations -seq add_new_table
```

Sẽ tạo 2 files:
- `000001_add_new_table.up.sql`
- `000001_add_new_table.down.sql`

### Chạy migrations

```bash
# Lên version mới nhất
migrate -path database/migrations -database "postgres://user:pass@localhost:5432/treasury?sslmode=disable" up

# Rollback 1 version
migrate -path database/migrations -database "..." down 1

# Xem version hiện tại
migrate -path database/migrations -database "..." version
```

### Quy tắc migration

1. **Không bao giờ sửa migration đã chạy** — tạo migration mới
2. **Luôn có cả up và down** — rollback phải hoạt động
3. **Test down migration** trước khi merge PR
4. **Một migration = một thay đổi logic** — dễ review, dễ rollback
5. **Dùng transaction** trong migration khi có thể (PostgreSQL hỗ trợ DDL trong transaction)

## Lưu ý quan trọng

- **Soft delete**: Hầu hết bảng dùng `deleted_at IS NULL` — luôn thêm filter này
- **Partitioned tables**: `audit_logs` phân vùng theo tháng — query phải include `performed_at` cho partition pruning
- **Append-only tables**: `approval_actions`, `audit_logs` — KHÔNG CÓ UPDATE/DELETE queries
- **Sequence generation**: `deal_sequences` dùng `ON CONFLICT DO UPDATE` pattern — tự động tạo row mới cho ngày mới
- **DECIMAL precision**: Dùng `pgtype.Numeric` hoặc `shopspring/decimal` — **KHÔNG dùng float64** cho số tiền
