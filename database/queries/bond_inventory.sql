-- ============================================================================
-- bond_inventory.sql — Queries for bond_inventory table
-- Treasury Management System — KienlongBank
-- ============================================================================

-- name: CreateBondInventory :one
INSERT INTO bond_inventory (
    id, bond_catalog_id, bond_code, bond_category, portfolio_type,
    available_quantity, acquisition_date, acquisition_price,
    version, updated_at, updated_by
) VALUES (
    gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7,
    1, NOW(), $8
)
RETURNING *;

-- name: GetBondInventoryByID :one
SELECT * FROM bond_inventory WHERE id = $1;

-- name: GetBondInventoryByBondCode :one
-- Lấy tồn kho theo bond_code + category + portfolio
SELECT * FROM bond_inventory
WHERE bond_code = $1 AND bond_category = $2 AND portfolio_type = $3;

-- name: GetAvailableQuantity :one
-- Lấy số lượng khả dụng — dùng cho validation trước khi bán
SELECT available_quantity FROM bond_inventory
WHERE bond_code = $1 AND bond_category = $2 AND portfolio_type = $3;

-- name: DecrementBondQuantity :one
-- Giảm tồn kho khi bán — OPTIMISTIC LOCKING qua version check
-- Nếu version không khớp → row không được update → application retry
-- Nếu available_quantity < sell_quantity → CHECK constraint fail → error
UPDATE bond_inventory SET
    available_quantity = available_quantity - sqlc.arg('sell_quantity')::BIGINT,
    version = version + 1,
    updated_by = $1
WHERE bond_code = $2
  AND bond_category = $3
  AND portfolio_type = $4
  AND version = sqlc.arg('expected_version')::INT
  AND available_quantity >= sqlc.arg('sell_quantity')::BIGINT
RETURNING *;

-- name: IncrementBondQuantity :one
-- Tăng tồn kho khi mua hoàn thành
UPDATE bond_inventory SET
    available_quantity = available_quantity + sqlc.arg('buy_quantity')::BIGINT,
    version = version + 1,
    updated_by = $1
WHERE bond_code = $2 AND bond_category = $3 AND portfolio_type = $4
RETURNING *;

-- name: UpsertBondInventory :one
-- Insert hoặc update tồn kho — dùng khi mua bond mới chưa có record
INSERT INTO bond_inventory (
    id, bond_catalog_id, bond_code, bond_category, portfolio_type,
    available_quantity, acquisition_date, acquisition_price,
    version, updated_at, updated_by
) VALUES (
    gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, 1, NOW(), $8
)
ON CONFLICT (bond_code, bond_category, portfolio_type) DO UPDATE SET
    available_quantity = bond_inventory.available_quantity + EXCLUDED.available_quantity,
    version = bond_inventory.version + 1,
    updated_by = EXCLUDED.updated_by
RETURNING *;

-- name: ListBondInventory :many
SELECT * FROM bond_inventory
WHERE available_quantity > 0
ORDER BY bond_code ASC
LIMIT $1 OFFSET $2;

-- name: ListBondInventoryByCategory :many
SELECT * FROM bond_inventory
WHERE bond_category = $1 AND available_quantity > 0
ORDER BY bond_code ASC
LIMIT $2 OFFSET $3;
