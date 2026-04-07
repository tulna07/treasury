-- ============================================================================
-- settlement_instructions.sql — Queries for settlement_instructions table
-- Treasury Management System — KienlongBank
-- ============================================================================

-- name: CreateSettlementInstruction :one
INSERT INTO settlement_instructions (
    id, counterparty_id, currency_code, owner_type,
    account_number, bank_name, swift_code, citad_code,
    description, is_default, is_active,
    created_at, created_by, updated_at, updated_by
) VALUES (
    gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, true,
    NOW(), $10, NOW(), $10
)
RETURNING *;

-- name: GetSettlementInstructionByID :one
SELECT * FROM settlement_instructions WHERE id = $1;

-- name: UpdateSettlementInstruction :one
UPDATE settlement_instructions SET
    account_number = COALESCE(sqlc.narg('account_number'), account_number),
    bank_name = COALESCE(sqlc.narg('bank_name'), bank_name),
    swift_code = COALESCE(sqlc.narg('swift_code'), swift_code),
    citad_code = COALESCE(sqlc.narg('citad_code'), citad_code),
    description = COALESCE(sqlc.narg('description'), description),
    is_default = COALESCE(sqlc.narg('is_default'), is_default),
    is_active = COALESCE(sqlc.narg('is_active'), is_active),
    updated_by = $1
WHERE id = $2
RETURNING *;

-- name: ListByCounterparty :many
SELECT * FROM settlement_instructions
WHERE counterparty_id = $1 AND is_active = true
ORDER BY owner_type ASC, currency_code ASC;

-- name: ListByCounterpartyAndCurrency :many
-- Lấy SSI theo đối tác + tiền tệ — dùng cho dropdown chọn pay code
SELECT * FROM settlement_instructions
WHERE counterparty_id = $1
  AND currency_code = $2
  AND is_active = true
ORDER BY is_default DESC, owner_type ASC;

-- name: ListByCounterpartyCurrencyAndOwner :many
SELECT * FROM settlement_instructions
WHERE counterparty_id = $1
  AND currency_code = $2
  AND owner_type = $3
  AND is_active = true
ORDER BY is_default DESC;

-- name: GetDefaultSSI :one
-- Lấy SSI mặc định cho counterparty + currency + owner_type
SELECT * FROM settlement_instructions
WHERE counterparty_id = $1
  AND currency_code = $2
  AND owner_type = $3
  AND is_default = true
  AND is_active = true
LIMIT 1;
