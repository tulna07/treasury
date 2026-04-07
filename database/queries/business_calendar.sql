-- ============================================================================
-- business_calendar.sql — Queries for business_calendar table
-- Treasury Management System — KienlongBank
-- ============================================================================

-- name: CreateBusinessCalendarEntry :one
INSERT INTO business_calendar (id, calendar_date, country_code, is_business_day, holiday_name, created_at)
VALUES (gen_random_uuid(), $1, $2, $3, $4, NOW())
RETURNING *;

-- name: IsBusinessDay :one
-- Kiểm tra một ngày có phải ngày làm việc không
SELECT is_business_day FROM business_calendar
WHERE calendar_date = $1 AND country_code = COALESCE(sqlc.narg('country_code'), 'VN');

-- name: GetPreviousBusinessDay :one
-- Lấy ngày làm việc trước đó — dùng cho tra cứu tỷ giá "ngày làm việc liền trước"
SELECT calendar_date FROM business_calendar
WHERE calendar_date < $1
  AND country_code = COALESCE(sqlc.narg('country_code'), 'VN')
  AND is_business_day = true
ORDER BY calendar_date DESC
LIMIT 1;

-- name: GetNextBusinessDay :one
-- Lấy ngày làm việc tiếp theo — dùng cho tính settlement date T+1, T+2
SELECT calendar_date FROM business_calendar
WHERE calendar_date > $1
  AND country_code = COALESCE(sqlc.narg('country_code'), 'VN')
  AND is_business_day = true
ORDER BY calendar_date ASC
LIMIT 1;

-- name: ListHolidaysByYear :many
-- Liệt kê ngày nghỉ/lễ trong năm — dùng cho admin quản lý lịch
SELECT * FROM business_calendar
WHERE EXTRACT(YEAR FROM calendar_date) = $1
  AND country_code = COALESCE(sqlc.narg('country_code'), 'VN')
  AND is_business_day = false
ORDER BY calendar_date ASC;

-- name: ListBusinessDaysByRange :many
-- Liệt kê ngày làm việc trong khoảng — dùng cho tính tenor
SELECT * FROM business_calendar
WHERE calendar_date BETWEEN $1 AND $2
  AND country_code = COALESCE(sqlc.narg('country_code'), 'VN')
  AND is_business_day = true
ORDER BY calendar_date ASC;

-- name: GetNthBusinessDayAfter :one
-- Lấy ngày làm việc thứ N sau một ngày — T+N calculation
-- Ví dụ: T+2 = GetNthBusinessDayAfter(trade_date, 2)
SELECT calendar_date FROM business_calendar
WHERE calendar_date > $1
  AND country_code = COALESCE(sqlc.narg('country_code'), 'VN')
  AND is_business_day = true
ORDER BY calendar_date ASC
LIMIT 1 OFFSET ($2 - 1);

-- name: BulkInsertBusinessCalendar :exec
-- Batch insert cho preload lịch cả năm
INSERT INTO business_calendar (id, calendar_date, country_code, is_business_day, holiday_name, created_at)
VALUES (gen_random_uuid(), $1, $2, $3, $4, NOW())
ON CONFLICT (calendar_date, country_code) DO UPDATE SET
    is_business_day = EXCLUDED.is_business_day,
    holiday_name = EXCLUDED.holiday_name;
