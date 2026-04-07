-- ============================================================================
-- international_payments.sql — Queries for international_payments table
-- Treasury Management System — KienlongBank
-- ============================================================================

-- name: CreateInternationalPayment :one
INSERT INTO international_payments (
    id, source_module, source_deal_id, source_leg_number,
    ticket_display, counterparty_id, debit_account, bic_code,
    currency_code, amount, transfer_date, counterparty_ssi,
    original_trade_date, settlement_status, created_at
) VALUES (
    gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12,
    'PENDING', NOW()
)
RETURNING *;

-- name: GetInternationalPaymentByID :one
SELECT * FROM international_payments WHERE id = $1;

-- name: GetInternationalPaymentBySourceDeal :one
-- Tìm payment theo deal gốc (và leg nếu là swap)
SELECT * FROM international_payments
WHERE source_module = $1
  AND source_deal_id = $2
  AND (source_leg_number = sqlc.narg('source_leg_number') OR sqlc.narg('source_leg_number') IS NULL);

-- name: ApprovePayment :one
-- BP.TTQT duyệt thanh toán
UPDATE international_payments SET
    settlement_status = 'APPROVED',
    settled_by = $1,
    settled_at = NOW()
WHERE id = $2 AND settlement_status = 'PENDING'
RETURNING *;

-- name: RejectPayment :one
-- BP.TTQT từ chối thanh toán
UPDATE international_payments SET
    settlement_status = 'REJECTED',
    rejection_reason = $3,
    settled_by = $1,
    settled_at = NOW()
WHERE id = $2 AND settlement_status = 'PENDING'
RETURNING *;

-- name: ListPendingPaymentsToday :many
-- Lấy danh sách payment cần xử lý hôm nay
SELECT
    ip.*,
    c.full_name AS counterparty_name
FROM international_payments ip
JOIN counterparties c ON c.id = ip.counterparty_id
WHERE ip.transfer_date = CURRENT_DATE
  AND ip.settlement_status = 'PENDING'
ORDER BY ip.created_at ASC
LIMIT $1 OFFSET $2;

-- name: ListPaymentsByDateRange :many
-- Tìm kiếm payment theo khoảng ngày
SELECT
    ip.*,
    c.full_name AS counterparty_name
FROM international_payments ip
JOIN counterparties c ON c.id = ip.counterparty_id
WHERE ip.transfer_date BETWEEN $1 AND $2
ORDER BY ip.transfer_date DESC, ip.created_at DESC
LIMIT $3 OFFSET $4;

-- name: ListPaymentsByStatus :many
SELECT
    ip.*,
    c.full_name AS counterparty_name
FROM international_payments ip
JOIN counterparties c ON c.id = ip.counterparty_id
WHERE ip.settlement_status = $1
ORDER BY ip.transfer_date DESC, ip.created_at DESC
LIMIT $2 OFFSET $3;
