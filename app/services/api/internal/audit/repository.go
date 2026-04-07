package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kienlongbank/treasury-api/internal/model"
	"github.com/kienlongbank/treasury-api/internal/repository"
	"github.com/kienlongbank/treasury-api/pkg/apperror"
	"github.com/kienlongbank/treasury-api/pkg/dto"
)

type pgxRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new audit log repository.
func NewRepository(pool *pgxpool.Pool) repository.AuditLogRepository {
	return &pgxRepository{pool: pool}
}

func (r *pgxRepository) List(ctx context.Context, filter dto.AuditLogFilter, pag dto.PaginationRequest) ([]model.AuditLog, int64, error) {
	conditions, args, argIdx := r.buildFilterConditions(filter)

	whereClause := "TRUE"
	if len(conditions) > 0 {
		whereClause = strings.Join(conditions, " AND ")
	}

	var total int64
	err := r.pool.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM audit_logs WHERE %s", whereClause), args...,
	).Scan(&total)
	if err != nil {
		return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to count audit logs")
	}

	query := fmt.Sprintf(`
		SELECT id, user_id, user_full_name, user_department, user_branch_code,
		       action, deal_module, deal_id, status_before, status_after,
		       old_values, new_values, reason, host(ip_address), performed_at
		FROM audit_logs
		WHERE %s
		ORDER BY performed_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, argIdx, argIdx+1)
	args = append(args, pag.PageSize, pag.Offset())

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to query audit logs")
	}
	defer rows.Close()

	var logs []model.AuditLog
	for rows.Next() {
		var log model.AuditLog
		err := rows.Scan(
			&log.ID, &log.UserID, &log.UserFullName, &log.UserDepartment, &log.UserBranchCode,
			&log.Action, &log.DealModule, &log.DealID, &log.StatusBefore, &log.StatusAfter,
			&log.OldValues, &log.NewValues, &log.Reason, &log.IPAddress, &log.PerformedAt,
		)
		if err != nil {
			return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to scan audit log")
		}
		logs = append(logs, log)
	}
	return logs, total, nil
}

func (r *pgxRepository) Stats(ctx context.Context, dateFrom, dateTo string) ([]dto.AuditLogStatsResponse, error) {
	query := `
		SELECT action, COUNT(*) as count
		FROM audit_logs
		WHERE performed_at >= $1::date AND performed_at < ($2::date + interval '1 day')
		GROUP BY action ORDER BY count DESC`

	rows, err := r.pool.Query(ctx, query, dateFrom, dateTo)
	if err != nil {
		return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to query audit log stats")
	}
	defer rows.Close()

	var stats []dto.AuditLogStatsResponse
	for rows.Next() {
		var s dto.AuditLogStatsResponse
		if err := rows.Scan(&s.Action, &s.Count); err != nil {
			return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to scan audit stats")
		}
		stats = append(stats, s)
	}
	if stats == nil {
		stats = []dto.AuditLogStatsResponse{}
	}
	return stats, nil
}

func (r *pgxRepository) buildFilterConditions(filter dto.AuditLogFilter) ([]string, []interface{}, int) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if filter.UserID != nil {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIdx))
		args = append(args, *filter.UserID)
		argIdx++
	}
	if filter.DealModule != nil {
		conditions = append(conditions, fmt.Sprintf("deal_module = $%d", argIdx))
		args = append(args, *filter.DealModule)
		argIdx++
	}
	if filter.DealID != nil {
		conditions = append(conditions, fmt.Sprintf("deal_id = $%d", argIdx))
		args = append(args, *filter.DealID)
		argIdx++
	}
	if filter.Action != nil {
		conditions = append(conditions, fmt.Sprintf("action ILIKE $%d", argIdx))
		args = append(args, "%"+*filter.Action+"%")
		argIdx++
	}
	if filter.DateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("performed_at >= $%d::date", argIdx))
		args = append(args, *filter.DateFrom)
		argIdx++
	}
	if filter.DateTo != nil {
		conditions = append(conditions, fmt.Sprintf("performed_at < ($%d::date + interval '1 day')", argIdx))
		args = append(args, *filter.DateTo)
		argIdx++
	}

	return conditions, args, argIdx
}

// auditLogToResponse converts a model to DTO.
func AuditLogToResponse(log *model.AuditLog) dto.AuditLogResponse {
	resp := dto.AuditLogResponse{
		ID:             log.ID,
		UserID:         log.UserID,
		UserFullName:   log.UserFullName,
		UserDepartment: log.UserDepartment,
		UserBranchCode: log.UserBranchCode,
		Action:         log.Action,
		DealModule:     log.DealModule,
		DealID:         log.DealID,
		StatusBefore:   log.StatusBefore,
		StatusAfter:    log.StatusAfter,
		Reason:         log.Reason,
		IPAddress:      log.IPAddress,
		PerformedAt:    log.PerformedAt,
	}

	if len(log.OldValues) > 0 {
		var ov map[string]interface{}
		if json.Unmarshal(log.OldValues, &ov) == nil {
			resp.OldValues = ov
		}
	}
	if len(log.NewValues) > 0 {
		var nv map[string]interface{}
		if json.Unmarshal(log.NewValues, &nv) == nil {
			resp.NewValues = nv
		}
	}

	return resp
}

var _ repository.AuditLogRepository = (*pgxRepository)(nil)

// unused import guard
var _ = uuid.Nil
