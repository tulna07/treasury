"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";

// ─── Types ────────────────────────────────────────────────────

export interface AuditLog {
  id: string;
  user_id: string;
  user_full_name: string;
  user_department: string;
  action: string;
  deal_module: string;
  deal_id: string;
  status_before: string;
  status_after: string;
  old_values: Record<string, unknown> | null;
  new_values: Record<string, unknown> | null;
  reason: string;
  ip_address: string;
  performed_at: string;
}

export interface AuditLogFilters {
  user_id?: string;
  deal_module?: string;
  deal_id?: string;
  action?: string;
  date_from?: string;
  date_to?: string;
  page?: number;
  page_size?: number;
}

export interface AuditLogListResponse {
  data: AuditLog[];
  total: number;
  page: number;
  page_size: number;
}

export interface AuditStat {
  action: string;
  total: number;
}

// ─── API Response Wrapper ─────────────────────────────────────

interface ApiResponse<T> {
  success: boolean;
  data: T;
}

// ─── Query Keys ───────────────────────────────────────────────

const auditKeys = {
  all: ["audit-logs"] as const,
  lists: () => [...auditKeys.all, "list"] as const,
  list: (filters: AuditLogFilters) => [...auditKeys.lists(), filters] as const,
  stats: (dateFrom?: string, dateTo?: string) =>
    [...auditKeys.all, "stats", dateFrom, dateTo] as const,
};

// ─── Hooks ────────────────────────────────────────────────────

export function useAuditLogs(filters: AuditLogFilters = {}) {
  const params: Record<string, string> = {};
  if (filters.user_id) params.user_id = filters.user_id;
  if (filters.deal_module) params.deal_module = filters.deal_module;
  if (filters.deal_id) params.deal_id = filters.deal_id;
  if (filters.action) params.action = filters.action;
  if (filters.date_from) params.date_from = filters.date_from;
  if (filters.date_to) params.date_to = filters.date_to;
  if (filters.page) params.page = String(filters.page);
  if (filters.page_size) params.page_size = String(filters.page_size);

  return useQuery({
    queryKey: auditKeys.list(filters),
    queryFn: async () => {
      const res = await api.get<ApiResponse<AuditLogListResponse>>(
        "/admin/audit-logs",
        { params }
      );
      return res.data;
    },
  });
}

export function useAuditStats(dateFrom?: string, dateTo?: string) {
  const params: Record<string, string> = {};
  if (dateFrom) params.date_from = dateFrom;
  if (dateTo) params.date_to = dateTo;

  return useQuery({
    queryKey: auditKeys.stats(dateFrom, dateTo),
    queryFn: async () => {
      const res = await api.get<ApiResponse<AuditStat[]>>(
        "/admin/audit-logs/stats",
        { params }
      );
      return res.data;
    },
  });
}
