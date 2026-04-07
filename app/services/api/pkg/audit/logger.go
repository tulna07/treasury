// Package audit provides a shared audit logging helper for banking-grade audit trails.
package audit

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// Entry represents a single audit log entry.
type Entry struct {
	UserID       uuid.UUID
	FullName     string
	Department   string
	BranchCode   string
	Action       string
	DealModule   string
	DealID       *uuid.UUID
	StatusBefore string
	StatusAfter  string
	OldValues    interface{}
	NewValues    interface{}
	Reason       string
	IPAddress    string
	UserAgent    string
}

// Logger writes audit log entries to the database.
type Logger struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewLogger creates a new audit Logger.
func NewLogger(pool *pgxpool.Pool, logger *zap.Logger) *Logger {
	return &Logger{pool: pool, logger: logger}
}

// Log writes an audit entry to the audit_logs table.
func (l *Logger) Log(ctx context.Context, entry Entry) {
	oldJSON, _ := marshalNullable(entry.OldValues)
	newJSON, _ := marshalNullable(entry.NewValues)

	_, err := l.pool.Exec(ctx, `
		INSERT INTO audit_logs (
			user_id, user_full_name, user_department, user_branch_code,
			action, deal_module, deal_id,
			status_before, status_after,
			old_values, new_values,
			reason, ip_address, user_agent
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		entry.UserID, entry.FullName, nullStr(entry.Department), nullStr(entry.BranchCode),
		entry.Action, entry.DealModule, entry.DealID,
		nullStr(entry.StatusBefore), nullStr(entry.StatusAfter),
		oldJSON, newJSON,
		nullStr(entry.Reason), nullStr(entry.IPAddress), nullStr(entry.UserAgent),
	)
	if err != nil {
		l.logger.Error("failed to write audit log",
			zap.String("action", entry.Action),
			zap.String("module", entry.DealModule),
			zap.Error(err),
		)
	}
}

// ExtractIP gets the client IP from request headers or RemoteAddr.
func ExtractIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-Ip"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func marshalNullable(v interface{}) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	return json.Marshal(v)
}

func nullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
