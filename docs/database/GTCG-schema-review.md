# Rà soát DB Schema GTCG vs BRD v3

**Ngày rà soát:** 05/04/2026
**BRD tham chiếu:** BRD-Treasury-System-v3.md (v3.0 — 02/04/2026)
**Schema files:** `005_bond.sql`, `003_master_data.sql`, `009_workflow.sql`, `010_document.sql`

---

## 1. Tổng quan kết quả

| Severity | Số lượng | Mô tả |
|----------|----------|-------|
| **Critical** | 2 | Thiếu trạng thái trung gian cho cancel 2 cấp; Query cancel bỏ qua luồng duyệt |
| **High** | 3 | Format deal_number sai vs BRD; FI Bond issue_date logic chưa rõ; bond_catalog thiếu soft delete |
| **Medium** | 4 | Thiếu cancel_approved_by_l1; "Chờ LĐ KTTC duyệt" chưa map rõ; coupon_rate precision; bond_inventory thiếu trường |
| **Low** | 3 | bond_catalog thiếu description; payment_frequency chưa dùng Phase 1; comment sai format |

**Tổng: 12 GAPs**

---

## 2. Bảng so sánh chi tiết: `bond_deals`

### 2.1 Trường dữ liệu — Govi Bond (BRD §3.2.1)

| # | Trường BRD | Loại | Schema hiện tại | Status | Ghi chú |
|---|-----------|------|-----------------|--------|---------|
| 1 | Ngày giao dịch | Bắt buộc | `trade_date DATE NOT NULL` | ✅ OK | |
| 2 | Ngày đặt lệnh | Bắt buộc | `order_date DATE NULL` | ⚠️ | BRD nói bắt buộc nhưng schema cho NULL. Govi cần bắt buộc, FI không có trường này → NULL OK cho FI. App-level validation đủ |
| 3 | Ngày giá trị | Bắt buộc | `value_date DATE NOT NULL` | ✅ OK | |
| 4 | Chiều giao dịch | Bắt buộc | `direction VARCHAR(5) — 'BUY'/'SELL'` | ✅ OK | BRD dùng "Bên mua"/"Bên bán" → UI mapping |
| 5 | Đối tác | Bắt buộc | `counterparty_id UUID NOT NULL` | ✅ OK | FK → counterparties |
| 6 | Tên đối tác | Tự động | — (join counterparties.full_name) | ✅ OK | View cần JOIN |
| 7 | Loại giao dịch | Bắt buộc | `transaction_type + transaction_type_other` | ✅ OK | REPO/REVERSE_REPO/OUTRIGHT/OTHER |
| 8 | Mã trái phiếu | Bắt buộc (Govi) | `bond_catalog_id UUID NULL` | ✅ OK | FK → bond_catalog. NULL cho FI/CCTG |
| 9 | Lãi suất coupon | Tự động (Govi) | `coupon_rate NUMERIC(10,4) NOT NULL` | ⚠️ Medium | BRD: "tối đa 2 chữ số thập phân" cho Govi, "4 chữ số" cho FI. Schema dùng 4 → OK tương thích. Nhưng Govi hiển thị cần round 2 |
| 10 | Ngày phát hành | Tự động (Govi) | `issue_date DATE NULL` | ✅ OK | Govi: copy từ bond_catalog. NULL cho edge case |
| 11 | Ngày đáo hạn | Tự động (Govi) | `maturity_date DATE NOT NULL` | ✅ OK | |
| 12 | Số lượng | Bắt buộc | `quantity BIGINT NOT NULL` | ✅ OK | CHECK > 0, số nguyên |
| 13 | Mệnh giá | Tự động (Govi) | `face_value NUMERIC(20,0) NOT NULL` | ✅ OK | Số nguyên VND |
| 14 | Lãi suất chiết khấu | Bắt buộc | `discount_rate NUMERIC(10,4)` | ✅ OK | |
| 15 | Giá sạch | Bắt buộc | `clean_price NUMERIC(20,0) NOT NULL` | ✅ OK | |
| 16 | Giá thanh toán | Bắt buộc | `settlement_price NUMERIC(20,0) NOT NULL` | ✅ OK | |
| 17 | Tổng giá trị GD | Tự động | `total_value NUMERIC(20,0) NOT NULL` | ✅ OK | = quantity × settlement_price |
| 18 | Hạch toán (HTM/AFS/HFT) | Bắt buộc khi mua | `portfolio_type VARCHAR(5) NULL` | ✅ OK | NULL khi bán |
| 19 | Ngày thanh toán | Bắt buộc | `payment_date DATE NOT NULL` | ✅ OK | |
| 20 | Kỳ hạn còn lại | Tự động | `remaining_tenor_days INT NOT NULL` | ✅ OK | = maturity_date − payment_date |
| 21 | Xác nhận giao dịch | Bắt buộc | `confirmation_method + confirmation_other` | ✅ OK | EMAIL/REUTERS/OTHER |
| 22 | Lập hợp đồng | Bắt buộc | `contract_prepared_by VARCHAR(15)` | ✅ OK | INTERNAL/COUNTERPARTY |
| 23 | File đính kèm | Không bắt buộc | `documents` table (deal_module='BOND') | ✅ OK | Quan hệ qua deal_id |
| 24 | Ghi chú | Không bắt buộc | `note TEXT NULL` | ✅ OK | |

### 2.2 Trường dữ liệu — FI Bond & CCTG (BRD §3.2.2)

| # | Trường BRD | Loại | Schema hiện tại | Status | Ghi chú |
|---|-----------|------|-----------------|--------|---------|
| 1 | Loại GTCG | Bắt buộc | `bond_category` → 'FINANCIAL_INSTITUTION' / 'CERTIFICATE_OF_DEPOSIT' | ✅ OK | |
| 2 | Mã GTCG | Không bắt buộc | `bond_code_manual VARCHAR(50) NULL` | ✅ OK | Text tự do cho FI/CCTG |
| 3 | Tổ chức phát hành | Bắt buộc | `issuer VARCHAR(500) NOT NULL` | ✅ OK | FI: nhập tay. Govi: copy từ catalog |
| 4 | Lãi suất coupon | Bắt buộc (FI) | `coupon_rate NUMERIC(10,4) NOT NULL` | ✅ OK | FI nhập tay, tối đa 4 decimal |
| 5 | Lãi suất chiết khấu | Bắt buộc | `discount_rate NUMERIC(10,4)` | ✅ OK | |
| 6 | Ngày phát hành | **Tự động** | `issue_date DATE NULL` | 🔴 **GAP-HIGH** | Xem [GAP #4](#gap-4) |
| 7 | Ngày đáo hạn | Bắt buộc | `maturity_date DATE NOT NULL` | ✅ OK | FI nhập tay |
| 8 | Số lượng | Bắt buộc | `quantity BIGINT NOT NULL` | ✅ OK | |
| 9 | Mệnh giá | Bắt buộc (FI) | `face_value NUMERIC(20,0) NOT NULL` | ✅ OK | FI nhập tay |

### 2.3 Status Flow (BRD §10.1 — Module GTCG)

| # | Trạng thái BRD | Mapping Schema | Status | Ghi chú |
|---|---------------|----------------|--------|---------|
| 1 | Open | `OPEN` | ✅ OK | |
| 2 | Chờ K.NV duyệt | `PENDING_L2_APPROVAL` | ✅ OK | TP đã duyệt, chờ GĐ |
| 3 | Từ chối | `REJECTED` | ✅ OK | GĐ từ chối |
| 4 | Chờ hạch toán | `PENDING_BOOKING` | ✅ OK | Chờ CV KTTC cấp 1 |
| 5 | Chờ LĐ KTTC duyệt | `PENDING_CHIEF_ACCOUNTANT` | ✅ OK | CV KTTC đã duyệt, chờ LĐ |
| 6 | Hoàn thành | `COMPLETED` | ✅ OK | |
| 7 | Hủy giao dịch | `VOIDED_BY_ACCOUNTING` | ✅ OK | KTTC từ chối |
| 8 | Đã hủy | `CANCELLED` | ⚠️ | Xem [GAP #1](#gap-1) — thiếu trạng thái trung gian |

### 2.4 Cancel Flow (BRD §4.2)

| # | Bước Cancel BRD | Trạng thái cần | Schema hiện tại | Status |
|---|----------------|----------------|-----------------|--------|
| 1 | CV yêu cầu hủy | Cần trạng thái trung gian | ❌ Không có | 🔴 **GAP-CRITICAL** |
| 2 | TP duyệt hủy cấp 1 | `PENDING_CANCEL_L1` | ❌ Không có | 🔴 **GAP-CRITICAL** |
| 3 | GĐ duyệt hủy cấp 2 | `PENDING_CANCEL_L2` | ❌ Không có | 🔴 **GAP-CRITICAL** |
| 4 | Hoàn thành hủy | `CANCELLED` | ✅ Có | |
| 5 | Từ chối hủy | Quay về trạng thái cũ | ❌ Không track | 🔴 **GAP-CRITICAL** |

---

## 3. Bảng so sánh: `bond_catalog` (BRD §5.2)

| # | Trường BRD | Schema hiện tại | Status | Ghi chú |
|---|-----------|-----------------|--------|---------|
| 1 | Mã trái phiếu | `bond_code VARCHAR(50) NOT NULL UNIQUE` | ✅ OK | |
| 2 | Tổ chức phát hành | `issuer VARCHAR(500) NOT NULL` | ✅ OK | Govi Bond hiển thị tự động |
| 3 | Lãi suất coupon | `coupon_rate NUMERIC(10,4) NOT NULL` | ✅ OK | |
| 4 | Ngày phát hành | `issue_date DATE NOT NULL` | ✅ OK | |
| 5 | Ngày đáo hạn | `maturity_date DATE NOT NULL` | ✅ OK | CHECK maturity > issue |
| 6 | Mệnh giá | `face_value NUMERIC(20,0) NOT NULL` | ✅ OK | |
| 7 | Soft delete | ❌ Không có `deleted_at` | 🟡 **GAP-HIGH** | Xem [GAP #5](#gap-5) |
| 8 | Kỳ trả coupon | `payment_frequency` (Phase 2) | ✅ OK | Sẵn sàng cho Phase 2 |

---

## 4. Bảng so sánh: `bond_inventory` (BRD §5.4 — chờ bổ sung)

| # | Trường | Schema hiện tại | Status | Ghi chú |
|---|--------|-----------------|--------|---------|
| 1 | bond_catalog_id | `UUID NULL` | ✅ OK | FK → bond_catalog |
| 2 | bond_code | `VARCHAR(50) NOT NULL` | ✅ OK | |
| 3 | bond_category | `VARCHAR(30) NOT NULL` | ✅ OK | |
| 4 | portfolio_type | `VARCHAR(5) NOT NULL` | ✅ OK | HTM/AFS/HFT |
| 5 | available_quantity | `BIGINT NOT NULL DEFAULT 0` | ✅ OK | CHECK >= 0 |
| 6 | acquisition_date | `DATE NULL` | ✅ OK | |
| 7 | acquisition_price | `NUMERIC(20,0) NULL` | ✅ OK | |
| 8 | version | `INT NOT NULL DEFAULT 1` | ✅ OK | Optimistic locking |
| 9 | issuer (join) | — cần join bond_catalog | ⚠️ Medium | Xem [GAP #8](#gap-8) |
| 10 | maturity_date (join) | — cần join bond_catalog | ⚠️ Medium | View cần join |

---

## 5. Workflow Schema (BRD §4, §6)

### 5.1 `deal_sequences` (BRD §3.2.5)

| # | Kiểm tra | BRD yêu cầu | Schema | Status |
|---|---------|-------------|--------|--------|
| 1 | Govi Bond prefix | `G` + 10 chữ số: `Gxxxxxxxxxx` | prefix='G', comment nói `G-20260403-0001` | 🟡 **GAP-HIGH** |
| 2 | FI Bond/CCTG prefix | `F` + 10 chữ số: `Fxxxxxxxxxx` | prefix='F' | 🟡 Cùng issue format |
| 3 | Ví dụ BRD | `G0000000001`, `F0000000001` | `G-20260403-0001` (có dấu gạch, có ngày) | 🟡 **GAP-HIGH** |

### 5.2 `approval_actions` — Action types cho GTCG

| # | Action BRD | Schema action_type | Status |
|---|-----------|-------------------|--------|
| 1 | TP duyệt cấp 1 | `DESK_HEAD_APPROVE` | ✅ OK |
| 2 | TP trả lại | `DESK_HEAD_RETURN` | ✅ OK |
| 3 | GĐ duyệt cấp 2 | `DIRECTOR_APPROVE` | ✅ OK |
| 4 | GĐ từ chối | `DIRECTOR_REJECT` | ✅ OK |
| 5 | CV KTTC hạch toán cấp 1 | `ACCOUNTANT_APPROVE` | ✅ OK |
| 6 | CV KTTC từ chối | `ACCOUNTANT_REJECT` | ✅ OK |
| 7 | LĐ KTTC hạch toán cấp 2 | `CHIEF_ACCOUNTANT_APPROVE` | ✅ OK |
| 8 | LĐ KTTC từ chối | `CHIEF_ACCOUNTANT_REJECT` | ✅ OK |
| 9 | CV Recall | `DEALER_RECALL` | ✅ OK |
| 10 | TP Recall | `DESK_HEAD_RECALL` | ✅ OK |
| 11 | CV yêu cầu hủy | `CANCEL_REQUEST` | ✅ OK |
| 12 | TP duyệt hủy cấp 1 | `CANCEL_DESK_HEAD_APPROVE` | ✅ OK |
| 13 | TP từ chối hủy | `CANCEL_DESK_HEAD_REJECT` | ✅ OK |
| 14 | GĐ duyệt hủy cấp 2 | `CANCEL_DIVISION_HEAD_APPROVE` | ✅ OK |
| 15 | GĐ từ chối hủy | `CANCEL_DIVISION_HEAD_REJECT` | ✅ OK |

> **Nhận xét:** `approval_actions` đã đầy đủ action types cho cancel 2 cấp. Tuy nhiên `bond_deals.status` constraint **thiếu** trạng thái trung gian để track cancel đang chờ duyệt.

### 5.3 `status_transition_rules`

Schema thiết kế config-driven, đúng hướng. Cần seed data cho BOND module với đầy đủ transitions bao gồm cancel flow.

### 5.4 `documents` — File đính kèm (BRD §3.2.1, §3.2.2)

| # | Kiểm tra | Status |
|---|---------|--------|
| 1 | deal_module = 'BOND' | ✅ OK — CHECK constraint có 'BOND' |
| 2 | document_type types | ✅ OK — TICKET, CONTRACT, CONFIRMATION, SUPPORTING, OTHER |
| 3 | Virus scan | ✅ OK — scan_status field |
| 4 | Version control | ✅ OK — version + is_current |

---

## 6. GAP LIST chi tiết

<a id="gap-1"></a>
### GAP #1 — CRITICAL: Thiếu trạng thái trung gian cho Cancel 2 cấp

**BRD tham chiếu:** §4.2 Hủy giao dịch
**Mô tả:** BRD v3 yêu cầu cancel qua 2 cấp: CV yêu cầu → TP duyệt cấp 1 → GĐ duyệt cấp 2. Schema hiện tại `bond_deals.status` chỉ có `CANCELLED` là trạng thái cuối, không có trạng thái trung gian cho giao dịch đang chờ duyệt hủy.

**Hiện tại:**
```sql
-- bond_deals status constraint
'OPEN', 'PENDING_L2_APPROVAL', 'REJECTED',
'PENDING_BOOKING', 'PENDING_CHIEF_ACCOUNTANT',
'COMPLETED', 'VOIDED_BY_ACCOUNTING', 'CANCELLED'
```

**Thiếu:**
- `PENDING_CANCEL_L1` — CV đã yêu cầu hủy, chờ TP duyệt
- `PENDING_CANCEL_L2` — TP đã duyệt hủy, chờ GĐ duyệt

**Đề xuất:**
```sql
ALTER TABLE bond_deals DROP CONSTRAINT chk_bond_deals_status;
ALTER TABLE bond_deals ADD CONSTRAINT chk_bond_deals_status CHECK (status IN (
    'OPEN', 'PENDING_L2_APPROVAL', 'REJECTED',
    'PENDING_BOOKING', 'PENDING_CHIEF_ACCOUNTANT',
    'COMPLETED', 'VOIDED_BY_ACCOUNTING',
    'PENDING_CANCEL_L1', 'PENDING_CANCEL_L2', 'CANCELLED'
));

-- Tương tự cho fx_deals (cùng issue)
ALTER TABLE fx_deals DROP CONSTRAINT chk_fx_deals_status;
ALTER TABLE fx_deals ADD CONSTRAINT chk_fx_deals_status CHECK (status IN (
    'OPEN', 'PENDING_L2_APPROVAL', 'REJECTED',
    'PENDING_BOOKING', 'PENDING_CHIEF_ACCOUNTANT',
    'PENDING_SETTLEMENT', 'COMPLETED',
    'VOIDED_BY_ACCOUNTING', 'VOIDED_BY_SETTLEMENT',
    'PENDING_CANCEL_L1', 'PENDING_CANCEL_L2', 'CANCELLED'
));
```

**Lý do:** Không có trạng thái trung gian → không thể:
1. Hiển thị GD đang chờ duyệt hủy trên dashboard TP/GĐ
2. Chặn CV tạo thêm thao tác trên GD đang chờ hủy
3. TP/GĐ từ chối hủy và quay GD về trạng thái cũ (cần lưu `status_before_cancel`)

---

<a id="gap-2"></a>
### GAP #2 — CRITICAL: Query `SetBondDealCancelRequest` bỏ qua luồng duyệt 2 cấp

**File:** `database/queries/bond_deals.sql` dòng 117-125
**Mô tả:** Query hiện tại set thẳng `status = 'CANCELLED'` khi CV yêu cầu hủy, bỏ qua 2 bước duyệt.

**Hiện tại:**
```sql
UPDATE bond_deals SET
    cancel_reason = $3,
    cancel_requested_by = $1,
    cancel_requested_at = NOW(),
    status = 'CANCELLED',   -- ← WRONG: skip TP + GĐ approval
    updated_by = $1
WHERE id = $2 AND deleted_at IS NULL AND status = $4
RETURNING *;
```

**Đề xuất:** Tách thành 3 queries:
```sql
-- 1. CV yêu cầu hủy → PENDING_CANCEL_L1
-- name: RequestBondDealCancel :one
UPDATE bond_deals SET
    cancel_reason = $3,
    cancel_requested_by = $1,
    cancel_requested_at = NOW(),
    status = 'PENDING_CANCEL_L1',
    updated_by = $1
WHERE id = $2 AND deleted_at IS NULL AND status IN ('COMPLETED', 'PENDING_SETTLEMENT')
RETURNING *;

-- 2. TP duyệt hủy cấp 1 → PENDING_CANCEL_L2
-- name: ApproveBondDealCancelL1 :one
UPDATE bond_deals SET
    status = 'PENDING_CANCEL_L2',
    updated_by = $1
WHERE id = $2 AND deleted_at IS NULL AND status = 'PENDING_CANCEL_L1'
RETURNING *;

-- 3. GĐ duyệt hủy cấp 2 → CANCELLED
-- name: ApproveBondDealCancelL2 :one
UPDATE bond_deals SET
    status = 'CANCELLED',
    updated_by = $1
WHERE id = $2 AND deleted_at IS NULL AND status = 'PENDING_CANCEL_L2'
RETURNING *;
```

---

<a id="gap-3"></a>
### GAP #3 — HIGH: Format deal_number không khớp BRD

**BRD tham chiếu:** §3.2.5 Mã giao dịch
**Mô tả:** BRD ghi: `Gxxxxxxxxxx` (G + 10 chữ số, VD: `G0000000001`). Schema comment ghi: `G-20260403-0001` (có dấu gạch, có ngày).

**Hiện tại:** `deal_number VARCHAR(30)`, comment: `G-20260403-0001`
**BRD v3.0.1:** Đã sửa khớp schema → `G-YYYYMMDD-NNNN`

**Quyết định (05/04/2026 — anh Minh Nguyen):** Giữ format `G-YYYYMMDD-NNNN` (schema hiện tại).
- Chứa ngày → dễ tra cứu
- Reset sequence mỗi ngày → number nhỏ gọn
- ✅ BRD đã cập nhật v3.0.1 cho khớp schema
- **Status: RESOLVED**

---

<a id="gap-4"></a>
### GAP #4 — HIGH: FI Bond `issue_date` — "Tự động" nhưng không có nguồn

**BRD tham chiếu:** §3.2.2 — Trường "Ngày phát hành", Loại: "Tự động"
**Mô tả:** BRD nói Ngày phát hành cho FI Bond là "Tự động" nhưng FI Bond **không dùng** `bond_catalog` dropdown (khác Govi). Không rõ giá trị lấy từ đâu.

**Schema hiện tại:** `issue_date DATE NULL` — cho phép NULL, OK cho trường hợp FI không có issue_date từ catalog.

**Phân tích:**
- Govi Bond: `issue_date` lấy tự động từ `bond_catalog.issue_date` → rõ ràng
- FI Bond: Không có bond_catalog FK → "Tự động" có thể là:
  - (a) = `trade_date` (ngày giao dịch) — hợp lý nhất cho CCTG
  - (b) Nhập tay nhưng hiển thị read-only sau khi nhập lần đầu
  - (c) Lỗi BRD — thực tế phải nhập tay

**Quyết định (05/04/2026 — anh Minh Nguyen):** FI Bond/CCTG `issue_date` = nhập tay, không bắt buộc. BRD ghi "Tự động" là lỗi — đã sửa trong v3.0.1.
- Schema giữ nguyên `DATE NULL` → đúng
- App logic: Govi → auto-fill từ catalog (read-only). FI/CCTG → input field nhập tay
- ✅ BRD đã cập nhật v3.0.1
- **Status: RESOLVED**

---

<a id="gap-5"></a>
### GAP #5 — HIGH: `bond_catalog` thiếu `deleted_at` (soft delete)

**Mô tả:** Bảng `counterparties` có `deleted_at` cho soft delete. `bond_catalog` chỉ có `is_active` nhưng không có `deleted_at`. Khi deactivate trái phiếu cũ, cần đảm bảo:
1. GD cũ vẫn join được bond_catalog
2. Trái phiếu đã deactivate không hiện trong dropdown tạo GD mới

**Hiện tại:** `is_active BOOLEAN NOT NULL DEFAULT true` — đủ cho mục đích ẩn/hiện.

**Đề xuất:** `is_active` đã đủ cho Phase 1. Nếu cần soft delete hoàn chỉnh:
```sql
ALTER TABLE bond_catalog ADD COLUMN deleted_at TIMESTAMPTZ NULL;
CREATE INDEX idx_bond_catalog_active ON bond_catalog (is_active) WHERE deleted_at IS NULL;
```

**Severity giảm xuống Medium** vì `is_active` có thể thay thế `deleted_at` cho Phase 1.

---

<a id="gap-6"></a>
### GAP #6 — MEDIUM: Thiếu `cancel_approved_by_l1` tracking

**BRD tham chiếu:** §4.2, §8.1 (Audit trail)
**Mô tả:** Schema `bond_deals` có `cancel_requested_by` và `cancel_requested_at` (CV yêu cầu), nhưng **thiếu** trường track TP đã duyệt hủy cấp 1.

**Phân tích:** `approval_actions` table đã log đủ (CANCEL_DESK_HEAD_APPROVE, CANCEL_DIVISION_HEAD_APPROVE). Trường trên `bond_deals` chỉ là denormalization cho tiện query.

**Đề xuất:** Không cần thêm column. Dùng JOIN `approval_actions` khi cần. Nếu muốn denormalize:
```sql
ALTER TABLE bond_deals ADD COLUMN cancel_approved_l1_by UUID NULL REFERENCES users(id);
ALTER TABLE bond_deals ADD COLUMN cancel_approved_l1_at TIMESTAMPTZ NULL;
```

**Khuyến nghị:** Bỏ qua — dùng `approval_actions` JOIN là đủ.

---

<a id="gap-7"></a>
### GAP #7 — MEDIUM: "Chờ LĐ KTTC duyệt" vs PENDING_CHIEF_ACCOUNTANT

**BRD tham chiếu:** §10.1
**Mô tả:** BRD liệt kê trạng thái "Chờ LĐ KTTC duyệt" riêng biệt với "Chờ hạch toán". Schema đã có `PENDING_CHIEF_ACCOUNTANT` tương ứng. Mapping đúng nhưng cần confirm:
- `PENDING_BOOKING` = CV KTTC chưa duyệt → "Chờ hạch toán"
- `PENDING_CHIEF_ACCOUNTANT` = CV KTTC đã duyệt, chờ LĐ → "Chờ LĐ KTTC duyệt"

**Status:** ✅ OK — mapping đúng, chỉ cần document rõ.

---

<a id="gap-8"></a>
### GAP #8 — MEDIUM: `bond_inventory` thiếu trường cho summary view

**BRD tham chiếu:** §5.4 (chờ bổ sung)
**Mô tả:** Hiện `bond_inventory` chỉ lưu quantity + price. Để hiển thị danh mục tồn kho cần join thêm `bond_catalog` cho issuer, maturity_date, coupon_rate.

**Phân tích:**
- `bond_catalog_id UUID NULL` — có FK nhưng NULL cho FI Bond/CCTG (không có catalog)
- FI Bond/CCTG tồn kho cần issuer, maturity_date → hiện không lưu trên inventory

**Đề xuất:** Thêm denormalized fields hoặc tạo view:
```sql
-- Option A: View (khuyến nghị — không đổi schema)
-- Xem 013_bond_views.sql → v_bond_inventory_summary

-- Option B: Denormalize (nếu query performance cần)
ALTER TABLE bond_inventory ADD COLUMN issuer VARCHAR(500) NULL;
ALTER TABLE bond_inventory ADD COLUMN maturity_date DATE NULL;
ALTER TABLE bond_inventory ADD COLUMN coupon_rate NUMERIC(10,4) NULL;
```

**Khuyến nghị:** Option A — View join bond_deals để lấy info. BRD §5.4 đang chờ bổ sung nên chưa cần thay đổi schema.

---

<a id="gap-9"></a>
### GAP #9 — MEDIUM: Coupon rate display precision khác nhau Govi vs FI

**BRD tham chiếu:** §3.2.1 vs §3.2.2
**Mô tả:**
- Govi Bond: "tối đa 2 chữ số thập phân" (VD: 2,51%)
- FI Bond: "tối đa 4 chữ số thập phân" (VD: 2,5000%)

**Schema:** `coupon_rate NUMERIC(10,4)` — lưu 4 decimal cho cả hai → OK ở DB level.

**Đề xuất:** Application/UI xử lý format hiển thị theo `bond_category`:
- GOVERNMENT → round 2 decimal, format VN (dấu `,`)
- FINANCIAL_INSTITUTION / CERTIFICATE_OF_DEPOSIT → 4 decimal, format VN

**Status:** Không cần sửa schema. Document cho FE team.

---

<a id="gap-10"></a>
### GAP #10 — LOW: `bond_catalog` thiếu `description` / `notes`

**Mô tả:** Một số trái phiếu có thông tin bổ sung (đặc điểm đặc biệt, điều kiện trả coupon sớm...). Hiện không có trường mô tả.

**Đề xuất:** Phase 2. Nếu cần:
```sql
ALTER TABLE bond_catalog ADD COLUMN description TEXT NULL;
```

---

<a id="gap-11"></a>
### GAP #11 — LOW: Schema comment `deal_number` format sai

**File:** `005_bond.sql` dòng 79
**Hiện tại:** `'Mã giao dịch gapless: G-20260403-0001 (Govi), F-20260403-0001 (FI/CD)'`
**Cần sửa:** Phải khớp với format thực tế sau khi resolve GAP #3.

---

<a id="gap-12"></a>
### GAP #12 — LOW: bond_catalog `payment_frequency` Phase 2

**Mô tả:** Trường đã có, Phase 1 không dùng. Không cần action.

---

## 7. Tổng hợp đề xuất

### Cần sửa ngay (Critical + High)

| # | Hành động | File | SQL |
|---|----------|------|-----|
| 1 | Thêm `PENDING_CANCEL_L1`, `PENDING_CANCEL_L2` vào bond_deals status | `005_bond.sql` | `ALTER TABLE bond_deals DROP CONSTRAINT chk_bond_deals_status; ALTER TABLE bond_deals ADD CONSTRAINT chk_bond_deals_status CHECK (...)` |
| 2 | Tương tự cho fx_deals | `004_fx.sql` | Xem GAP #1 |
| 3 | Sửa query `SetBondDealCancelRequest` → tách 3 queries | `bond_deals.sql` | Xem GAP #2 |
| 4 | ~~Xác nhận format deal_number~~ | ✅ RESOLVED | Giữ `G-YYYYMMDD-NNNN`, BRD đã cập nhật |
| 5 | ~~Xác nhận FI Bond issue_date~~ | ✅ RESOLVED | Nhập tay, không bắt buộc. BRD đã sửa |

### Nên làm (Medium)

| # | Hành động | Ghi chú |
|---|----------|---------|
| 6 | Tạo SQL views cho GTCG | Xem `013_bond_views.sql` |
| 7 | Document coupon_rate display rules cho FE | GAP #9 |
| 8 | Seed status_transition_rules cho BOND module | Bao gồm cancel flow |

### Để sau (Low)

| # | Hành động | Phase |
|---|----------|-------|
| 9 | Thêm description cho bond_catalog | Phase 2 |
| 10 | Sửa schema comments | Anytime |

---

## 8. Ma trận GTCG Status × Allowed Actions

```
                        ┌─────────────────────────────────────────────────────────────────────┐
                        │                     Allowed Actions per Status                       │
┌───────────────────────┼──────┬─────────┬──────────┬───────────┬────────┬────────┬───────────┤
│ Status                │ Edit │ Recall  │ Approve  │ Reject    │ Cancel │ Clone  │ Book      │
│                       │ (CV) │ (CV/TP) │          │           │ (CV)   │ (CV)   │ (KTTC)    │
├───────────────────────┼──────┼─────────┼──────────┼───────────┼────────┼────────┼───────────┤
│ OPEN                  │  ✅  │   —     │ TP→L1    │    —      │   —    │   —    │    —      │
│ PENDING_L2_APPROVAL   │  ❌  │  ✅     │ GĐ→L2   │ GĐ reject │   —    │   —    │    —      │
│ REJECTED              │  ❌  │   —     │    —      │    —      │   —    │  ✅   │    —      │
│ PENDING_BOOKING       │  ❌  │   —     │    —      │ CV KTTC   │   —    │   —    │  ✅ L1   │
│ PENDING_CHIEF_ACCT    │  ❌  │   —     │    —      │ LĐ KTTC  │   —    │   —    │  ✅ L2   │
│ COMPLETED             │  ❌  │   —     │    —      │    —      │  ✅   │   —    │    —      │
│ VOIDED_BY_ACCOUNTING  │  ❌  │   —     │    —      │    —      │   —    │  ✅   │    —      │
│ PENDING_CANCEL_L1 *   │  ❌  │   —     │ TP→hủy   │ TP reject │   —    │   —    │    —      │
│ PENDING_CANCEL_L2 *   │  ❌  │   —     │ GĐ→hủy   │ GĐ reject│   —    │   —    │    —      │
│ CANCELLED             │  ❌  │   —     │    —      │    —      │   —    │   —    │    —      │
└───────────────────────┴──────┴─────────┴──────────┴───────────┴────────┴────────┴───────────┘

* PENDING_CANCEL_L1, PENDING_CANCEL_L2 = trạng thái mới cần thêm (GAP #1)
```

---

*Tài liệu này được tạo tự động từ rà soát schema ngày 05/04/2026.*
