# DATABASE DESIGN — DANH SÁCH BẢNG

# HỆ THỐNG TREASURY — KIENLONGBANK

| Thông tin | Chi tiết |
|-----------|----------|
| **Phiên bản** | 1.0 — Draft |
| **Ngày** | 03/04/2026 |
| **Dựa trên** | BRD v3.0 (02/04/2026) + Test Cases v1 |
| **Tác giả** | KAI (AI Banking Assistant) |
| **Trạng thái** | Draft — Chờ review |

---

## NGUYÊN TẮC THIẾT KẾ

### Banking Standards — Secure by Design

1. **Soft delete** — Không xóa cứng dữ liệu. Mọi bản ghi có `deleted_at` (nullable timestamp)
2. **Audit columns** — Mọi bảng đều có `created_at`, `created_by`, `updated_at`, `updated_by`
3. **UUID primary key** — Dùng UUID v7 (time-sortable) thay vì auto-increment (chống enumeration attack)
4. **Immutable transactions** — Giao dịch sau khi phê duyệt không cho sửa, mọi thay đổi qua bản ghi mới hoặc audit log
5. **Decimal precision** — Dùng `NUMERIC(20,4)` cho tiền tệ, `NUMERIC(10,6)` cho tỷ giá/lãi suất — tránh floating-point errors
6. **Row-Level Security (RLS)** — PostgreSQL RLS policies theo role, không chỉ filter ở application layer
7. **Encryption at rest** — Các trường nhạy cảm đánh dấu, hỗ trợ column-level encryption nếu cần
8. **Foreign key constraints** — Enforce referential integrity tại DB level
9. **Check constraints** — Validate business rules tại DB level (VD: amount > 0, rate > 0)
10. **Index strategy** — Index theo query patterns: status, date range, counterparty, module

### User & Auth — Standalone + Zitadel Ready

- **Standalone mode:** Bảng `users` + `roles` + `user_roles` đầy đủ — Dev có thể chạy độc lập không cần IdP
- **Zitadel mode:** Bảng `auth_config` chứa config IdP. Khi bật Zitadel:
  - Login qua Zitadel OIDC → JWT token
  - Bảng `users.external_id` map với Zitadel user ID
  - Role mapping: Zitadel groups → `roles` qua bảng `external_role_mapping`
  - Sync user info từ JWT claims (name, email) — KHÔNG ghi đè DB bằng empty string
- **Chuyển đổi:** Toggle qua env/config `AUTH_MODE=standalone|zitadel` — không thay đổi schema

---

## TỔNG QUAN — 28 BẢNG

```
📦 Treasury Database
├── 🔐 AUTH & USER (5 bảng)
│   ├── users
│   ├── roles
│   ├── user_roles
│   ├── auth_config
│   └── external_role_mapping
│
├── 📋 MASTER DATA (6 bảng)
│   ├── counterparties          -- Đối tác
│   ├── currencies              -- Danh mục tiền tệ
│   ├── currency_pairs          -- Cặp tiền + quy tắc tính
│   ├── bond_catalog            -- Danh mục trái phiếu
│   ├── settlement_instructions -- Pay code / SSI
│   └── exchange_rates          -- Tỷ giá (hạn mức quy đổi)
│
├── 💱 MODULE 1: FX (2 bảng)
│   ├── fx_deals                -- GD Spot/Forward/Swap (header)
│   └── fx_deal_legs            -- Chân GD (Spot/Fwd = 1 chân, Swap = 2 chân)
│
├── 📜 MODULE 2: GTCG (2 bảng)
│   ├── bond_deals              -- GD Govi Bond / FI Bond / CCTG
│   └── bond_inventory          -- Tồn kho GTCG
│
├── 💰 MODULE 3: MM (2 bảng)
│   ├── mm_deals                -- GD Liên ngân hàng
│   └── mm_omo_repo_deals       -- GD OMO + Repo KBNN
│
├── 📊 MODULE 4: HẠN MỨC (3 bảng)
│   ├── credit_limits           -- Hạn mức được cấp theo đối tác
│   ├── limit_utilizations      -- Snapshot sử dụng hạn mức
│   └── limit_approvals         -- Lịch sử duyệt hạn mức
│
├── 🌐 MODULE 5: TTQT (1 bảng)
│   └── international_payments  -- Giao dịch chuyển tiền quốc tế
│
├── ✅ WORKFLOW & APPROVAL (2 bảng)
│   ├── approval_steps          -- Các bước phê duyệt của mỗi GD
│   └── deal_status_transitions -- Cấu hình chuyển trạng thái hợp lệ
│
├── 📎 ATTACHMENT (1 bảng)
│   └── attachments             -- File đính kèm (ticket, hợp đồng)
│
├── 🔔 NOTIFICATION (1 bảng)
│   └── notifications           -- Thông báo in-app + email
│
└── 📝 AUDIT (1 bảng)
    └── audit_logs              -- Toàn bộ lịch sử hành động
```

---

## CHI TIẾT TỪNG BẢNG

### 🔐 AUTH & USER (5 bảng)

#### 1. `users`
> Thông tin người dùng hệ thống. Standalone hoặc sync từ Zitadel.

| Cột | Kiểu | Mô tả |
|-----|------|-------|
| `id` | UUID PK | ID nội bộ |
| `external_id` | VARCHAR(255) NULL | ID từ Zitadel (khi AUTH_MODE=zitadel) |
| `username` | VARCHAR(100) UNIQUE | Tên đăng nhập (standalone) |
| `password_hash` | VARCHAR(255) NULL | BCrypt hash (chỉ dùng standalone mode) |
| `full_name` | VARCHAR(255) | Họ tên đầy đủ |
| `email` | VARCHAR(255) | Email công ty |
| `department` | VARCHAR(100) | Phòng/Ban/Khối (K.NV&ĐCTC, P.QLRR, P.KTTC, TTQT) |
| `position` | VARCHAR(100) | Chức danh |
| `is_active` | BOOLEAN DEFAULT true | Trạng thái hoạt động |
| `last_login_at` | TIMESTAMPTZ NULL | Lần đăng nhập cuối |
| `created_at` | TIMESTAMPTZ | |
| `updated_at` | TIMESTAMPTZ | |
| `deleted_at` | TIMESTAMPTZ NULL | Soft delete |

**Index:** `username`, `external_id`, `department`

#### 2. `roles`
> 10 role theo BRD v3, mục 2.4.

| Cột | Kiểu | Mô tả |
|-----|------|-------|
| `id` | UUID PK | |
| `code` | VARCHAR(50) UNIQUE | VD: `CV_KNV`, `TP_KNV`, `GD_TT`, `GD_KHOI`, `CV_QLRR`, `TPB_QLRR`, `CV_KTTC`, `LD_KTTC`, `BP_TTQT`, `ADMIN` |
| `name` | VARCHAR(255) | Tên hiển thị |
| `description` | TEXT NULL | Mô tả quyền hạn |
| `scope` | VARCHAR(50) | Phạm vi dữ liệu: `ALL`, `MODULE_SPECIFIC`, `STEP_SPECIFIC` |
| `created_at` | TIMESTAMPTZ | |

**Seed data:** 10 roles từ BRD mục 2.4

#### 3. `user_roles`
> Gán role cho user. 1 user có thể có nhiều role.

| Cột | Kiểu | Mô tả |
|-----|------|-------|
| `id` | UUID PK | |
| `user_id` | UUID FK → users | |
| `role_id` | UUID FK → roles | |
| `granted_at` | TIMESTAMPTZ | Ngày gán |
| `granted_by` | UUID FK → users | Người gán |

**Unique:** `(user_id, role_id)`

#### 4. `auth_config`
> Cấu hình xác thực. Toggle standalone ↔ Zitadel.

| Cột | Kiểu | Mô tả |
|-----|------|-------|
| `id` | UUID PK | |
| `auth_mode` | VARCHAR(20) | `standalone` hoặc `zitadel` |
| `issuer_url` | VARCHAR(500) NULL | Zitadel issuer URL |
| `client_id` | VARCHAR(255) NULL | OIDC Client ID |
| `client_secret_encrypted` | TEXT NULL | Encrypted client secret |
| `scopes` | VARCHAR(500) NULL | OIDC scopes |
| `auto_create_user` | BOOLEAN DEFAULT true | Tự tạo user khi login lần đầu qua Zitadel |
| `sync_user_info` | BOOLEAN DEFAULT true | Sync name/email từ JWT claims |
| `is_active` | BOOLEAN DEFAULT true | |
| `updated_at` | TIMESTAMPTZ | |
| `updated_by` | UUID FK → users | |

#### 5. `external_role_mapping`
> Map Zitadel groups/roles → internal roles.

| Cột | Kiểu | Mô tả |
|-----|------|-------|
| `id` | UUID PK | |
| `external_group` | VARCHAR(255) | Tên group/role trên Zitadel |
| `role_id` | UUID FK → roles | Role nội bộ tương ứng |
| `created_at` | TIMESTAMPTZ | |

**Unique:** `(external_group, role_id)`

---

### 📋 MASTER DATA (6 bảng)

#### 6. `counterparties`
> Danh sách đối tác giao dịch. BRD mục 5.1.

| Cột | Kiểu | Mô tả |
|-----|------|-------|
| `id` | UUID PK | |
| `code` | VARCHAR(20) UNIQUE | Mã nội bộ K.NV (VD: MSBI, ACB) |
| `full_name` | VARCHAR(500) | Tên đầy đủ |
| `short_name` | VARCHAR(255) NULL | Tên viết tắt |
| `cif` | VARCHAR(50) | Mã CIF khách hàng |
| `swift_code` | VARCHAR(11) NULL | SWIFT/BIC code (không bắt buộc) |
| `tax_id` | VARCHAR(20) NULL | Mã số thuế |
| `address` | TEXT NULL | Địa chỉ |
| `fx_uses_limit` | BOOLEAN DEFAULT false | FX có chiếm hạn mức không (v3) |
| `is_active` | BOOLEAN DEFAULT true | |
| `created_at` | TIMESTAMPTZ | |
| `created_by` | UUID FK → users | |
| `updated_at` | TIMESTAMPTZ | |
| `updated_by` | UUID FK → users | |
| `deleted_at` | TIMESTAMPTZ NULL | |

**Index:** `code`, `cif`, `swift_code`

#### 7. `currencies`
> Danh mục loại tiền. Dùng cho dropdown FX, MM.

| Cột | Kiểu | Mô tả |
|-----|------|-------|
| `id` | UUID PK | |
| `code` | VARCHAR(3) UNIQUE | ISO 4217: USD, VND, EUR, AUD, GBP, JPY, KRW... |
| `name` | VARCHAR(100) | Tên đầy đủ |
| `decimal_places` | SMALLINT DEFAULT 2 | Số chữ số thập phân (VND=0, USD=2, JPY=0) |
| `is_active` | BOOLEAN DEFAULT true | |

#### 8. `currency_pairs`
> Cặp tiền + quy tắc tính toán. Quyết định công thức Thành tiền FX.

| Cột | Kiểu | Mô tả |
|-----|------|-------|
| `id` | UUID PK | |
| `base_currency` | VARCHAR(3) FK → currencies | Tiền cơ sở (VD: USD trong USD/VND) |
| `quote_currency` | VARCHAR(3) FK → currencies | Tiền định giá (VD: VND trong USD/VND) |
| `pair_code` | VARCHAR(7) UNIQUE | VD: `USD/VND`, `EUR/USD`, `EUR/GBP` |
| `rate_decimal_places` | SMALLINT | Số thập phân tỷ giá (2 cho USD/VND, USD/JPY, USD/KRW; 4 cho còn lại) |
| `calculation_rule` | VARCHAR(20) | `MULTIPLY` (USD/VND, .../USD, cross), `DIVIDE` (USD/...) |
| `result_currency` | VARCHAR(3) FK → currencies | Đơn vị kết quả (VND, USD, hoặc tiền quote) |
| `is_active` | BOOLEAN DEFAULT true | |

**Mục đích:** Dev không cần hardcode logic — đọc từ DB, áp dụng đúng công thức.

#### 9. `bond_catalog`
> Danh mục trái phiếu (Govi Bond). BRD mục 5.2. Dùng cho dropdown OMO, Repo KBNN, Govi Bond.

| Cột | Kiểu | Mô tả |
|-----|------|-------|
| `id` | UUID PK | |
| `bond_code` | VARCHAR(50) UNIQUE | Mã trái phiếu (VD: TD2135068) |
| `issuer` | VARCHAR(500) | Tổ chức phát hành (VD: Kho bạc Nhà nước) |
| `coupon_rate` | NUMERIC(10,4) | Lãi suất coupon (%/năm) |
| `issue_date` | DATE | Ngày phát hành |
| `maturity_date` | DATE | Ngày đáo hạn |
| `face_value` | NUMERIC(20,0) | Mệnh giá (VND, số nguyên) |
| `bond_type` | VARCHAR(20) | `GOVI`, `FI`, `CCTG` |
| `is_active` | BOOLEAN DEFAULT true | |
| `created_at` | TIMESTAMPTZ | |
| `created_by` | UUID FK → users | |
| `updated_at` | TIMESTAMPTZ | |
| `updated_by` | UUID FK → users | |

**Index:** `bond_code`, `bond_type`, `maturity_date`

#### 10. `settlement_instructions`
> Pay code / SSI — chỉ dẫn thanh toán. BRD mục 5.3. 1 đối tác có thể có nhiều SSI, kể cả cùng loại tiền.

| Cột | Kiểu | Mô tả |
|-----|------|-------|
| `id` | UUID PK | |
| `counterparty_id` | UUID FK → counterparties | |
| `currency_code` | VARCHAR(3) FK → currencies | Loại tiền |
| `owner_type` | VARCHAR(10) | `KLB` (của KienlongBank) hoặc `COUNTERPARTY` (của đối tác) |
| `account_number` | VARCHAR(100) | Số tài khoản |
| `bank_name` | VARCHAR(500) | Tên ngân hàng trung gian |
| `swift_code` | VARCHAR(11) NULL | SWIFT code ngân hàng trung gian |
| `citad_code` | VARCHAR(20) NULL | Citad code (nội địa) |
| `description` | TEXT NULL | Mô tả chi tiết (full text SSI) |
| `is_default` | BOOLEAN DEFAULT false | SSI mặc định cho đối tác + tiền |
| `is_active` | BOOLEAN DEFAULT true | |
| `created_at` | TIMESTAMPTZ | |
| `created_by` | UUID FK → users | |
| `updated_at` | TIMESTAMPTZ | |
| `updated_by` | UUID FK → users | |

**Index:** `counterparty_id`, `currency_code`, `owner_type`

#### 11. `exchange_rates`
> Tỷ giá quy đổi cho hạn mức. BRD mục 3.4.3. Lưu cuối ngày LV.

| Cột | Kiểu | Mô tả |
|-----|------|-------|
| `id` | UUID PK | |
| `currency_code` | VARCHAR(3) FK → currencies | Loại tiền (VD: USD) |
| `effective_date` | DATE | Ngày hiệu lực (cuối ngày LV) |
| `buy_transfer_rate` | NUMERIC(20,4) | Tỷ giá mua chuyển khoản |
| `sell_transfer_rate` | NUMERIC(20,4) | Tỷ giá bán chuyển khoản |
| `mid_rate` | NUMERIC(20,4) | = (buy + sell) / 2 — tính sẵn |
| `source` | VARCHAR(50) | Nguồn tỷ giá (VD: `KLB_DAILY`, `NHNN`) |
| `created_at` | TIMESTAMPTZ | |
| `created_by` | UUID FK → users | |

**Unique:** `(currency_code, effective_date)`
**Index:** `effective_date`

---

### 💱 MODULE 1: FX — KINH DOANH NGOẠI TỆ (2 bảng)

> **Thiết kế:** Tách header (`fx_deals`) và chân giao dịch (`fx_deal_legs`).
> - Spot/Forward = 1 header + 1 leg
> - Swap = 1 header + 2 legs
> → Tránh duplicate columns, dễ mở rộng.

#### 12. `fx_deals`
> Header giao dịch FX. Chứa thông tin chung.

| Cột | Kiểu | Mô tả |
|-----|------|-------|
| `id` | UUID PK | |
| `deal_number` | BIGSERIAL UNIQUE | ID giao dịch tự tăng (hiển thị) |
| `ticket_number` | VARCHAR(20) NULL | Số Ticket (không bắt buộc) |
| `counterparty_id` | UUID FK → counterparties | Đối tác |
| `deal_type` | VARCHAR(10) | `SPOT`, `FORWARD`, `SWAP` |
| `direction` | VARCHAR(10) | Spot/Fwd: `SELL`, `BUY`; Swap: `SELL_BUY`, `BUY_SELL` |
| `amount` | NUMERIC(20,2) | Khối lượng giao dịch |
| `currency_code` | VARCHAR(3) FK → currencies | Loại tiền giao dịch |
| `pair_code` | VARCHAR(7) FK → currency_pairs | Cặp tiền (tự động) |
| `trade_date` | DATE | Ngày giao dịch |
| `uses_credit_limit` | BOOLEAN DEFAULT false | GD này có chiếm hạn mức không |
| `status` | VARCHAR(30) | Trạng thái (xem bảng enum bên dưới) |
| `note` | TEXT NULL | Ghi chú |
| `cloned_from_id` | UUID NULL FK → fx_deals | Clone từ GD nào (nếu có) |
| `cancel_reason` | TEXT NULL | Lý do hủy |
| `cancel_requested_by` | UUID NULL FK → users | Người yêu cầu hủy |
| `cancel_requested_at` | TIMESTAMPTZ NULL | Thời điểm yêu cầu hủy |
| `created_at` | TIMESTAMPTZ | |
| `created_by` | UUID FK → users | CV K.NV tạo |
| `updated_at` | TIMESTAMPTZ | |
| `updated_by` | UUID FK → users | |
| `deleted_at` | TIMESTAMPTZ NULL | |

**Check:** `amount > 0`
**Index:** `status`, `trade_date`, `counterparty_id`, `deal_type`, `created_by`

#### 13. `fx_deal_legs`
> Chân giao dịch FX. Spot/Forward = 1 row, Swap = 2 rows.

| Cột | Kiểu | Mô tả |
|-----|------|-------|
| `id` | UUID PK | |
| `deal_id` | UUID FK → fx_deals | |
| `leg_number` | SMALLINT | 1 (Spot/Fwd/Swap chân 1), 2 (Swap chân 2) |
| `value_date` | DATE | Ngày Giá trị |
| `settlement_date` | DATE | Ngày thực hiện |
| `exchange_rate` | NUMERIC(20,6) | Tỷ giá |
| `converted_amount` | NUMERIC(20,2) | Thành tiền (tính tự động) |
| `converted_currency` | VARCHAR(3) FK → currencies | Đơn vị kết quả |
| `klb_ssi_id` | UUID FK → settlement_instructions | Pay code KLB |
| `counterparty_ssi_id` | UUID FK → settlement_instructions | Pay code đối tác |
| `requires_ttqt` | BOOLEAN | Có cần TTQT không (tự động theo SSI đối tác) |
| `created_at` | TIMESTAMPTZ | |

**Unique:** `(deal_id, leg_number)`
**Check:** `exchange_rate > 0`, `leg_number IN (1, 2)`

---

### 📜 MODULE 2: GTCG — GIẤY TỜ CÓ GIÁ (2 bảng)

#### 14. `bond_deals`
> Giao dịch GTCG: Govi Bond, FI Bond, CCTG. BRD mục 3.2.

| Cột | Kiểu | Mô tả |
|-----|------|-------|
| `id` | UUID PK | |
| `deal_code` | VARCHAR(20) UNIQUE | Mã GD tự sinh: `G0000000001` (Govi), `F0000000001` (FI/CCTG) |
| `bond_category` | VARCHAR(10) | `GOVI`, `FI_BOND`, `CCTG` |
| `trade_date` | DATE | Ngày giao dịch |
| `order_date` | DATE NULL | Ngày đặt lệnh (chỉ Govi) |
| `value_date` | DATE | Ngày giá trị |
| `direction` | VARCHAR(10) | `BUY`, `SELL` |
| `counterparty_id` | UUID FK → counterparties | |
| `deal_type` | VARCHAR(20) | `REPO`, `REVERSE_REPO`, `OUTRIGHT`, `OTHER` |
| `deal_type_other` | VARCHAR(255) NULL | Nếu deal_type = OTHER |
| `bond_catalog_id` | UUID NULL FK → bond_catalog | Mã trái phiếu (Govi → bắt buộc, FI/CCTG → không bắt buộc) |
| `bond_code_manual` | VARCHAR(50) NULL | Mã GTCG nhập tay (FI Bond, CCTG) |
| `issuer` | VARCHAR(500) | Tổ chức phát hành |
| `coupon_rate` | NUMERIC(10,4) | Lãi suất coupon (%) |
| `issue_date` | DATE NULL | Ngày phát hành |
| `maturity_date` | DATE | Ngày đáo hạn |
| `quantity` | BIGINT | Số lượng (số nguyên) |
| `face_value` | NUMERIC(20,0) | Mệnh giá (VND) |
| `discount_rate` | NUMERIC(10,4) | Lãi suất chiết khấu (%) |
| `clean_price` | NUMERIC(20,0) | Giá sạch (VND) |
| `settlement_price` | NUMERIC(20,0) | Giá thanh toán (VND) |
| `total_value` | NUMERIC(20,0) | Tổng giá trị = Số lượng × Giá thanh toán |
| `accounting_type` | VARCHAR(5) NULL | `HTM`, `AFS`, `HFT` — chỉ khi direction = BUY |
| `payment_date` | DATE | Ngày thanh toán |
| `remaining_term_days` | INT | Kỳ hạn còn lại (ngày) — tính tự động |
| `confirmation_method` | VARCHAR(20) | `EMAIL`, `REUTERS`, `OTHER` |
| `confirmation_other` | VARCHAR(255) NULL | Nếu method = OTHER |
| `contract_maker` | VARCHAR(20) | `KLB`, `COUNTERPARTY` |
| `status` | VARCHAR(30) | Trạng thái |
| `note` | TEXT NULL | Ghi chú (ghi ngày mua + giá mua khi bán) |
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

#### 15. `bond_inventory`
> Tồn kho GTCG. Block cứng khi bán vượt tồn kho.

| Cột | Kiểu | Mô tả |
|-----|------|-------|
| `id` | UUID PK | |
| `bond_catalog_id` | UUID NULL FK → bond_catalog | Mã TP (Govi) |
| `bond_code` | VARCHAR(50) | Mã GTCG (dùng chung cho cả Govi, FI, CCTG) |
| `bond_category` | VARCHAR(10) | `GOVI`, `FI_BOND`, `CCTG` |
| `accounting_type` | VARCHAR(5) | `HTM`, `AFS`, `HFT` |
| `quantity_available` | BIGINT | Số lượng hiện có |
| `acquisition_date` | DATE NULL | Ngày mua |
| `acquisition_price` | NUMERIC(20,0) NULL | Giá mua |
| `updated_at` | TIMESTAMPTZ | |
| `updated_by` | UUID FK → users | |

**Unique:** `(bond_code, bond_category, accounting_type)`
**Check:** `quantity_available >= 0`

---

### 💰 MODULE 3: MM — KINH DOANH TIỀN TỆ (2 bảng)

#### 16. `mm_deals`
> Giao dịch Liên ngân hàng. BRD mục 3.3.1.

| Cột | Kiểu | Mô tả |
|-----|------|-------|
| `id` | UUID PK | |
| `deal_number` | BIGSERIAL UNIQUE | ID giao dịch |
| `ticket_number` | VARCHAR(20) NULL | Số Ticket (không bắt buộc) |
| `counterparty_id` | UUID FK → counterparties | |
| `currency_code` | VARCHAR(3) FK → currencies | VND hoặc USD |
| `klb_ssi_id` | UUID FK → settlement_instructions | Chỉ dẫn TT KLB |
| `counterparty_ssi_id` | UUID FK → settlement_instructions | Chỉ dẫn TT đối tác |
| `direction` | VARCHAR(20) | `DEPOSIT` (Gửi tiền), `RECEIVE_DEPOSIT` (Nhận TG), `LEND` (Cho vay), `BORROW` (Vay) |
| `principal_amount` | NUMERIC(20,2) | Số tiền gốc |
| `interest_rate` | NUMERIC(10,6) | Lãi suất (%/năm) |
| `day_count_convention` | VARCHAR(20) | `ACT_365`, `ACT_360`, `ACT_ACT` |
| `trade_date` | DATE | Ngày giao dịch |
| `effective_date` | DATE | Ngày hiệu lực |
| `tenor_days` | INT | Kỳ hạn (ngày) |
| `maturity_date` | DATE | Ngày đáo hạn = effective_date + tenor_days |
| `interest_amount` | NUMERIC(20,2) | Số tiền lãi (tính tự động) |
| `maturity_amount` | NUMERIC(20,2) | Số tiền đáo hạn = gốc + lãi |
| `has_collateral` | BOOLEAN | Có TSBĐ không |
| `collateral_currency` | VARCHAR(3) NULL | Loại tiền TSBĐ |
| `collateral_value` | TEXT NULL | Giá trị TSBĐ (text, không tích hợp phong tỏa) |
| `requires_ttqt` | BOOLEAN | Có cần TTQT không |
| `uses_credit_limit` | BOOLEAN DEFAULT true | Luôn chiếm hạn mức (Liên NH) |
| `status` | VARCHAR(30) | Trạng thái |
| `note` | TEXT NULL | |
| `cloned_from_id` | UUID NULL FK → mm_deals | |
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

#### 17. `mm_omo_repo_deals`
> Giao dịch OMO + Repo KBNN. Tách riêng vì cấu trúc trường khác MM Liên NH. Không qua QLRR, không qua TTQT, không chiếm hạn mức.

| Cột | Kiểu | Mô tả |
|-----|------|-------|
| `id` | UUID PK | |
| `deal_number` | BIGSERIAL UNIQUE | ID giao dịch |
| `deal_subtype` | VARCHAR(10) | `OMO`, `REPO_KBNN` |
| `session_name` | VARCHAR(100) | Phiên giao dịch (VD: Phiên 1) |
| `trade_date` | DATE | Ngày giao dịch |
| `counterparty_id` | UUID FK → counterparties | OMO: cố định NHNN; Repo KBNN: dropdown |
| `bond_catalog_id` | UUID FK → bond_catalog | Mã trái phiếu |
| `winning_rate` | NUMERIC(10,6) | Lãi suất trúng thầu (%/năm) |
| `tenor_days` | INT | Kỳ hạn (ngày) |
| `payment_date_1` | DATE | Ngày thanh toán 1 |
| `payment_date_2` | DATE | Ngày thanh toán 2 |
| `haircut_pct` | NUMERIC(5,2) | Hair cut (%) |
| `status` | VARCHAR(30) | Trạng thái |
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

### 📊 MODULE 4: HẠN MỨC (3 bảng)

#### 18. `credit_limits`
> Hạn mức liên ngân hàng được cấp. BRD mục 3.4.

| Cột | Kiểu | Mô tả |
|-----|------|-------|
| `id` | UUID PK | |
| `counterparty_id` | UUID FK → counterparties | |
| `limit_type` | VARCHAR(20) | `WITH_COLLATERAL`, `WITHOUT_COLLATERAL` |
| `limit_amount` | NUMERIC(20,2) NULL | Hạn mức (VND). NULL = "Không giới hạn" |
| `is_unlimited` | BOOLEAN DEFAULT false | True = Không giới hạn |
| `effective_date` | DATE | Ngày hiệu lực |
| `expiry_date` | DATE NULL | Ngày hết hạn (nếu có) |
| `approved_by_info` | TEXT NULL | Thông tin phê duyệt TGĐ (tham chiếu) |
| `created_at` | TIMESTAMPTZ | |
| `created_by` | UUID FK → users | |
| `updated_at` | TIMESTAMPTZ | |
| `updated_by` | UUID FK → users | |

**Index:** `counterparty_id`, `limit_type`, `effective_date`

#### 19. `limit_utilizations`
> Snapshot sử dụng hạn mức tại thời điểm duyệt. Append-only.

| Cột | Kiểu | Mô tả |
|-----|------|-------|
| `id` | UUID PK | |
| `counterparty_id` | UUID FK → counterparties | |
| `snapshot_date` | DATE | Ngày snapshot |
| `limit_type` | VARCHAR(20) | `WITH_COLLATERAL`, `WITHOUT_COLLATERAL` |
| `limit_granted` | NUMERIC(20,2) NULL | Hạn mức cấp (NULL = không giới hạn) |
| `utilized_beginning` | NUMERIC(20,2) | Đã SD đầu ngày (VND quy đổi) |
| `utilized_intraday` | NUMERIC(20,2) | SD trong ngày (VND quy đổi) |
| `utilized_total` | NUMERIC(20,2) | Tổng đã SD |
| `remaining` | NUMERIC(20,2) NULL | Còn lại (NULL = không giới hạn) |
| `exchange_rate_used` | NUMERIC(20,4) NULL | Tỷ giá quy đổi USD→VND đã dùng |
| `detail_json` | JSONB NULL | Chi tiết breakdown: MM + FX + FI Bond |
| `created_at` | TIMESTAMPTZ | |
| `created_by` | UUID FK → users | |

**Index:** `counterparty_id`, `snapshot_date`

#### 20. `limit_approvals`
> Lịch sử duyệt hạn mức cho từng GD. Snapshot tại thời điểm QLRR duyệt. BRD mục 8.3.

| Cột | Kiểu | Mô tả |
|-----|------|-------|
| `id` | UUID PK | |
| `deal_type` | VARCHAR(10) | `FX`, `MM` — loại GD chiếm hạn mức |
| `deal_id` | UUID | ID giao dịch (FK logic, không enforce vì đa bảng) |
| `counterparty_id` | UUID FK → counterparties | |
| `limit_type` | VARCHAR(20) | `WITH_COLLATERAL`, `WITHOUT_COLLATERAL` |
| `deal_amount_vnd` | NUMERIC(20,2) | Giá trị GD quy đổi VND |
| `limit_snapshot` | JSONB | Snapshot: cấp, đã SD, còn lại tại thời điểm duyệt |
| `cv_qlrr_approved_by` | UUID NULL FK → users | CV QLRR duyệt cấp 1 |
| `cv_qlrr_approved_at` | TIMESTAMPTZ NULL | |
| `tpb_qlrr_approved_by` | UUID NULL FK → users | TPB QLRR duyệt cấp 2 |
| `tpb_qlrr_approved_at` | TIMESTAMPTZ NULL | |
| `approval_status` | VARCHAR(20) | `PENDING`, `APPROVED`, `REJECTED` |
| `rejection_reason` | TEXT NULL | |
| `created_at` | TIMESTAMPTZ | |

---

### 🌐 MODULE 5: TTQT (1 bảng)

#### 21. `international_payments`
> Giao dịch chuyển tiền quốc tế. BRD mục 3.5. Tự động tạo khi GD FX/MM chuyển sang "Chờ TTQT".

| Cột | Kiểu | Mô tả |
|-----|------|-------|
| `id` | UUID PK | |
| `source_module` | VARCHAR(10) | `FX`, `MM` |
| `source_deal_id` | UUID | ID giao dịch gốc |
| `source_leg_number` | SMALLINT NULL | Swap: 1 hoặc 2; Spot/Fwd/MM: NULL |
| `ticket_display` | VARCHAR(25) | Số Ticket hiển thị (Swap: suffix A/B) |
| `counterparty_id` | UUID FK → counterparties | |
| `debit_account` | VARCHAR(100) | Trích tiền từ TK (HABIB KLB) |
| `bic_code` | VARCHAR(11) NULL | BIC CODE đối tác |
| `currency_code` | VARCHAR(3) FK → currencies | |
| `amount` | NUMERIC(20,2) | Số tiền chuyển |
| `transfer_date` | DATE | Ngày chuyển tiền |
| `counterparty_ssi` | TEXT | SSI đối tác |
| `original_trade_date` | DATE | Ngày tạo GD gốc |
| `approved_by_knv` | VARCHAR(255) NULL | Người duyệt tại K.NV |
| `ttqt_status` | VARCHAR(20) | `PENDING`, `APPROVED`, `REJECTED` |
| `ttqt_approved_by` | UUID NULL FK → users | |
| `ttqt_approved_at` | TIMESTAMPTZ NULL | |
| `rejection_reason` | TEXT NULL | |
| `created_at` | TIMESTAMPTZ | |

**Index:** `transfer_date`, `ttqt_status`, `source_module`

---

### ✅ WORKFLOW & APPROVAL (2 bảng)

#### 22. `approval_steps`
> Lịch sử các bước phê duyệt của mỗi giao dịch. Append-only — mỗi action = 1 row.

| Cột | Kiểu | Mô tả |
|-----|------|-------|
| `id` | UUID PK | |
| `deal_module` | VARCHAR(10) | `FX`, `BOND`, `MM`, `MM_OMO` |
| `deal_id` | UUID | ID giao dịch |
| `step_type` | VARCHAR(30) | `TP_APPROVE`, `TP_RETURN`, `GD_APPROVE`, `GD_REJECT`, `QLRR_CV_APPROVE`, `QLRR_CV_REJECT`, `QLRR_TPB_APPROVE`, `QLRR_TPB_REJECT`, `KTTC_CV_APPROVE`, `KTTC_CV_REJECT`, `KTTC_LD_APPROVE`, `KTTC_LD_REJECT`, `TTQT_APPROVE`, `TTQT_REJECT`, `RECALL_CV`, `RECALL_TP`, `CANCEL_REQUEST`, `CANCEL_TP_APPROVE`, `CANCEL_TP_REJECT`, `CANCEL_GD_APPROVE`, `CANCEL_GD_REJECT` |
| `status_before` | VARCHAR(30) | Trạng thái trước |
| `status_after` | VARCHAR(30) | Trạng thái sau |
| `performed_by` | UUID FK → users | Người thực hiện |
| `performed_at` | TIMESTAMPTZ | Thời điểm |
| `reason` | TEXT NULL | Lý do (bắt buộc khi reject, recall, cancel) |
| `metadata` | JSONB NULL | Dữ liệu bổ sung (VD: snapshot hạn mức) |

**Index:** `deal_module, deal_id`, `performed_by`, `performed_at`

#### 23. `deal_status_transitions`
> Cấu hình chuyển trạng thái hợp lệ. Dev dùng để validate, không hardcode.

| Cột | Kiểu | Mô tả |
|-----|------|-------|
| `id` | UUID PK | |
| `deal_module` | VARCHAR(10) | `FX`, `BOND`, `MM`, `MM_OMO` |
| `from_status` | VARCHAR(30) | Trạng thái hiện tại |
| `to_status` | VARCHAR(30) | Trạng thái đích |
| `required_role` | VARCHAR(50) FK → roles.code | Role cần có để thực hiện |
| `requires_reason` | BOOLEAN DEFAULT false | Bắt buộc nhập lý do |
| `requires_confirmation` | BOOLEAN DEFAULT false | Popup xác nhận |

**Unique:** `(deal_module, from_status, to_status, required_role)`

---

### 📎 ATTACHMENT (1 bảng)

#### 24. `attachments`
> File đính kèm. 1 file = 1 giao dịch. BRD: ticket, hợp đồng.

| Cột | Kiểu | Mô tả |
|-----|------|-------|
| `id` | UUID PK | |
| `deal_module` | VARCHAR(10) | `FX`, `BOND`, `MM`, `MM_OMO` |
| `deal_id` | UUID | ID giao dịch |
| `file_name` | VARCHAR(500) | Tên file gốc |
| `file_path` | VARCHAR(1000) | Đường dẫn storage |
| `file_size` | BIGINT | Kích thước (bytes) |
| `mime_type` | VARCHAR(100) | VD: `application/pdf` |
| `checksum` | VARCHAR(64) | SHA-256 hash — integrity check |
| `uploaded_by` | UUID FK → users | |
| `uploaded_at` | TIMESTAMPTZ | |
| `deleted_at` | TIMESTAMPTZ NULL | |

**Index:** `deal_module, deal_id`

---

### 🔔 NOTIFICATION (1 bảng)

#### 25. `notifications`
> Thông báo in-app + email. BRD mục 7.

| Cột | Kiểu | Mô tả |
|-----|------|-------|
| `id` | UUID PK | |
| `recipient_id` | UUID FK → users | Người nhận |
| `channel` | VARCHAR(10) | `IN_APP`, `EMAIL` |
| `event_type` | VARCHAR(50) | VD: `DEAL_PENDING_APPROVAL`, `DEAL_REJECTED`, `DEAL_CANCELLED` |
| `title` | VARCHAR(500) | Tiêu đề |
| `body` | TEXT | Nội dung |
| `deal_module` | VARCHAR(10) NULL | Module liên quan |
| `deal_id` | UUID NULL | GD liên quan |
| `is_read` | BOOLEAN DEFAULT false | Đã đọc |
| `read_at` | TIMESTAMPTZ NULL | |
| `sent_at` | TIMESTAMPTZ NULL | Thời điểm gửi email |
| `created_at` | TIMESTAMPTZ | |

**Index:** `recipient_id, is_read`, `created_at`

---

### 📝 AUDIT (1 bảng)

#### 26. `audit_logs`
> Toàn bộ lịch sử hành động. Append-only, KHÔNG sửa/xóa. BRD mục 8.

| Cột | Kiểu | Mô tả |
|-----|------|-------|
| `id` | UUID PK | |
| `user_id` | UUID FK → users | Người thực hiện |
| `user_full_name` | VARCHAR(255) | Snapshot tên (tránh join) |
| `user_department` | VARCHAR(100) | Snapshot đơn vị |
| `action` | VARCHAR(50) | 14 sự kiện theo BRD 8.1 |
| `deal_module` | VARCHAR(10) | `FX`, `BOND`, `MM`, `MM_OMO`, `LIMIT`, `TTQT`, `SYSTEM` |
| `deal_id` | UUID NULL | ID giao dịch |
| `status_before` | VARCHAR(30) NULL | Trạng thái trước |
| `status_after` | VARCHAR(30) NULL | Trạng thái sau |
| `old_values` | JSONB NULL | Giá trị cũ (khi sửa) |
| `new_values` | JSONB NULL | Giá trị mới (khi sửa) |
| `reason` | TEXT NULL | Lý do (recall, từ chối, hủy) |
| `ip_address` | INET NULL | IP client |
| `user_agent` | TEXT NULL | Browser/client info |
| `performed_at` | TIMESTAMPTZ | Timestamp chính xác |

**KHÔNG có `updated_at`, `deleted_at`** — append-only by design.
**Index:** `deal_module, deal_id`, `user_id`, `performed_at`, `action`
**Partition:** Theo tháng trên `performed_at` (nếu data lớn)

---

## ENUM VALUES — TRẠNG THÁI GIAO DỊCH

### Module FX

```
OPEN → PENDING_KNV_APPROVE → PENDING_ACCOUNTING → PENDING_LD_KTTC
→ PENDING_TTQT → COMPLETED
                           → COMPLETED (nếu không TTQT)

Nhánh từ chối:
PENDING_KNV_APPROVE → REJECTED (GĐ từ chối)
PENDING_ACCOUNTING → CANCELLED_BY_KTTC (KTTC từ chối)
PENDING_LD_KTTC → CANCELLED_BY_KTTC
PENDING_TTQT → CANCELLED_BY_TTQT
COMPLETED / PENDING_TTQT → CANCELLED (hủy sau hoàn thành)
```

| Code | Tên hiển thị |
|------|-------------|
| `OPEN` | Open |
| `PENDING_KNV_APPROVE` | Chờ K.NV duyệt |
| `REJECTED` | Từ chối |
| `PENDING_ACCOUNTING` | Chờ hạch toán |
| `PENDING_LD_KTTC` | Chờ LĐ KTTC duyệt |
| `PENDING_QLRR` | Chờ QLRR (chỉ MM Liên NH) |
| `PENDING_TTQT` | Chờ TTQT |
| `COMPLETED` | Hoàn thành |
| `CANCELLED_BY_KTTC` | Hủy giao dịch (KTTC từ chối) |
| `CANCELLED_BY_TTQT` | Hủy giao dịch (TTQT từ chối) |
| `CANCELLED_BY_QLRR` | Hủy giao dịch (QLRR từ chối) |
| `CANCELLED` | Đã hủy (hủy sau hoàn thành) |

### Module GTCG
Giống FX nhưng **KHÔNG có** `PENDING_QLRR`, `PENDING_TTQT`, `CANCELLED_BY_TTQT`, `CANCELLED_BY_QLRR`.

### Module MM Liên NH
Có **tất cả** trạng thái (bao gồm `PENDING_QLRR`).

### Module MM OMO / Repo KBNN
Giống GTCG — **KHÔNG có** `PENDING_QLRR`, `PENDING_TTQT`.

---

## TỔNG HỢP

| # | Nhóm | Bảng | Mô tả |
|---|------|------|-------|
| 1 | Auth | `users` | Người dùng |
| 2 | Auth | `roles` | 10 role |
| 3 | Auth | `user_roles` | Gán role |
| 4 | Auth | `auth_config` | Config standalone/Zitadel |
| 5 | Auth | `external_role_mapping` | Map Zitadel groups → roles |
| 6 | Master | `counterparties` | Đối tác |
| 7 | Master | `currencies` | Loại tiền |
| 8 | Master | `currency_pairs` | Cặp tiền + quy tắc tính |
| 9 | Master | `bond_catalog` | Danh mục trái phiếu |
| 10 | Master | `settlement_instructions` | Pay code / SSI |
| 11 | Master | `exchange_rates` | Tỷ giá quy đổi hạn mức |
| 12 | FX | `fx_deals` | GD FX header |
| 13 | FX | `fx_deal_legs` | Chân GD FX |
| 14 | GTCG | `bond_deals` | GD trái phiếu |
| 15 | GTCG | `bond_inventory` | Tồn kho GTCG |
| 16 | MM | `mm_deals` | GD Liên ngân hàng |
| 17 | MM | `mm_omo_repo_deals` | GD OMO + Repo KBNN |
| 18 | Limit | `credit_limits` | Hạn mức được cấp |
| 19 | Limit | `limit_utilizations` | Snapshot SD hạn mức |
| 20 | Limit | `limit_approvals` | Lịch sử duyệt hạn mức |
| 21 | TTQT | `international_payments` | Chuyển tiền quốc tế |
| 22 | Workflow | `approval_steps` | Bước phê duyệt |
| 23 | Workflow | `deal_status_transitions` | Cấu hình trạng thái |
| 24 | File | `attachments` | File đính kèm |
| 25 | Notify | `notifications` | Thông báo |
| 26 | Audit | `audit_logs` | Lịch sử hành động |

**Tổng: 26 bảng**

---

## GHI CHÚ THIẾT KẾ

### Tại sao tách `fx_deal_legs`?
- Swap có 2 chân với tỷ giá, ngày, SSI khác nhau → tách ra tránh duplicate columns
- Spot/Forward = 1 leg → JOIN đơn giản
- Dễ mở rộng nếu có multi-leg deals trong tương lai

### Tại sao tách `mm_omo_repo_deals`?
- OMO/Repo KBNN có cấu trúc trường KHÁC hẳn MM Liên NH (session, haircut, 2 ngày TT, không có gốc/lãi/kỳ hạn)
- Luồng phê duyệt khác (không QLRR, không TTQT)
- Tách giúp Dev không cần nullable columns phức tạp

### Tại sao `deal_status_transitions`?
- Config-driven thay vì hardcode — thay đổi luồng chỉ cần update data
- Dễ kiểm tra: "từ trạng thái X, role Y có quyền chuyển sang Z không?"
- Test cases verify dễ hơn

### Tại sao `currency_pairs` có `calculation_rule`?
- BRD v3 có 4 loại công thức khác nhau (USD/VND nhân, USD/XXX chia, XXX/USD nhân, cross pair nhân)
- Config trong DB → Dev không hardcode logic per-pair
- Thêm cặp tiền mới chỉ cần insert, không deploy code

### User standalone + Zitadel
- `AUTH_MODE=standalone`: login qua `users.username` + `password_hash`, session JWT tự phát
- `AUTH_MODE=zitadel`: login qua OIDC, match `users.external_id`, role từ `external_role_mapping`
- Chuyển đổi không cần migration — chỉ cần config

---

*— Hết tài liệu —*

**Phiên bản:** 1.0 | **Ngày:** 03/04/2026 | **Trạng thái:** Draft — Chờ review
