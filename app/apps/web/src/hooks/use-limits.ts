"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";

// ─── Types ───────────────────────────────────────────────────

export type LimitType = "COLLATERALIZED" | "UNCOLLATERALIZED";

/** Raw row returned by GET /limits — one row per limit_type per counterparty */
interface RawCreditLimit {
  id: string;
  counterparty_id: string;
  counterparty_name: string;
  cif_code: string;
  limit_type: "COLLATERALIZED" | "UNCOLLATERALIZED";
  limit_amount: string | null;
  is_unlimited: boolean;
  effective_from: string;
  effective_to?: string | null;
  is_current: boolean;
  expiry_date?: string | null;
  approval_reference?: string;
  created_at: string;
  updated_at: string;
}

export type ApprovalStatus =
  | "PENDING"
  | "APPROVED_RM"
  | "APPROVED_HEAD"
  | "REJECTED_RM"
  | "REJECTED_HEAD";

export interface CreditLimit {
  counterparty_id: string;
  counterparty_code: string;
  counterparty_name: string;
  cif: string;
  collateralized_limit: number | null;
  uncollateralized_limit: number | null;
  collateralized_unlimited: boolean;
  uncollateralized_unlimited: boolean;
  collateralized_utilized: number;
  uncollateralized_utilized: number;
  collateralized_remaining: number | null;
  uncollateralized_remaining: number | null;
  effective_date: string;
  expiry_date?: string;
  approval_reference?: string;
  version: number;
  updated_at: string;
}

export interface CreditLimitListResponse {
  data: CreditLimit[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export interface LimitFilters {
  search?: string;
  page?: number;
  page_size?: number;
}

export interface UpdateCreditLimitRequest {
  collateralized_limit: number | null;
  uncollateralized_limit: number | null;
  collateralized_unlimited: boolean;
  uncollateralized_unlimited: boolean;
  approval_reference?: string;
  version: number;
}

export interface DailySummaryRow {
  counterparty_name: string;
  counterparty_code: string;
  cif: string;
  limit_type: LimitType;
  granted_limit: number | null;
  granted_unlimited: boolean;
  mm_utilized: number;
  bond_utilized: number;
  fx_utilized: number;
  total_utilized: number;
  remaining: number | null;
  fx_rate: number;
}

export interface DailySummaryResponse {
  data: DailySummaryRow[];
  summary_date: string;
  fx_rate_usd_vnd: number;
}

export interface LimitApproval {
  id: string;
  deal_module: string;
  deal_number: string;
  counterparty_name: string;
  counterparty_code: string;
  limit_type: LimitType;
  amount_vnd: number;
  rm_status: ApprovalStatus;
  rm_approved_by?: string;
  rm_approved_at?: string;
  head_status: ApprovalStatus;
  head_approved_by?: string;
  head_approved_at?: string;
  created_at: string;
}

export interface LimitApprovalListResponse {
  data: LimitApproval[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export interface ApprovalFilters {
  status?: string;
  deal_module?: string;
  page?: number;
  page_size?: number;
}

export interface LimitUtilization {
  counterparty_id: string;
  counterparty_name: string;
  limit_type: LimitType;
  mm_principal: number;
  bond_settlement: number;
  fx_amount: number;
  total_utilized: number;
  granted_limit: number | null;
  granted_unlimited: boolean;
  remaining: number | null;
  fx_rate_used: number;
}

export interface ApproveRejectRequest {
  deal_id: string;
  action: "APPROVE" | "REJECT";
  reason?: string;
}

// ─── API Response Wrapper ─────────────────────────────────────

interface ApiResponse<T> {
  success: boolean;
  data: T;
  meta?: { request_id: string; timestamp: string };
}

// ─── Query Keys ──────────────────────────────────────────────

const limitKeys = {
  all: ["limits"] as const,
  lists: () => [...limitKeys.all, "list"] as const,
  list: (filters: LimitFilters) => [...limitKeys.lists(), filters] as const,
  details: () => [...limitKeys.all, "detail"] as const,
  detail: (counterpartyId: string) =>
    [...limitKeys.details(), counterpartyId] as const,
  dailySummary: (date: string) =>
    [...limitKeys.all, "daily-summary", date] as const,
  approvals: () => [...limitKeys.all, "approvals"] as const,
  approvalList: (filters: ApprovalFilters) =>
    [...limitKeys.approvals(), filters] as const,
  utilization: (counterpartyId: string) =>
    [...limitKeys.all, "utilization", counterpartyId] as const,
};

// ─── Transform ───────────────────────────────────────────────

/**
 * Group raw per-limit_type rows into one CreditLimit per counterparty.
 * The API returns separate COLLATERALIZED / UNCOLLATERALIZED rows;
 * the UI expects a single merged object.
 */
function transformLimits(raw: RawCreditLimit[]): CreditLimit[] {
  const map = new Map<string, CreditLimit>();
  for (const row of raw) {
    let entry = map.get(row.counterparty_id);
    if (!entry) {
      entry = {
        counterparty_id: row.counterparty_id,
        counterparty_code: row.cif_code || "",
        counterparty_name: row.counterparty_name,
        cif: row.cif_code || "",
        collateralized_limit: null,
        uncollateralized_limit: null,
        collateralized_unlimited: false,
        uncollateralized_unlimited: false,
        collateralized_utilized: 0,
        uncollateralized_utilized: 0,
        collateralized_remaining: null,
        uncollateralized_remaining: null,
        effective_date: row.effective_from,
        expiry_date: row.expiry_date ?? undefined,
        approval_reference: row.approval_reference,
        version: 0,
        updated_at: row.updated_at || row.created_at,
      };
      map.set(row.counterparty_id, entry);
    }

    const amount =
      row.limit_amount !== null && row.limit_amount !== undefined
        ? parseFloat(row.limit_amount)
        : null;

    if (row.limit_type === "COLLATERALIZED") {
      entry.collateralized_limit = amount;
      entry.collateralized_unlimited = row.is_unlimited;
    } else {
      entry.uncollateralized_limit = amount;
      entry.uncollateralized_unlimited = row.is_unlimited;
    }
  }
  return Array.from(map.values());
}

// ─── Hooks ───────────────────────────────────────────────────

export function useCreditLimits(filters: LimitFilters = {}) {
  const params: Record<string, string> = {};
  if (filters.search) params.search = filters.search;
  if (filters.page) params.page = String(filters.page);
  if (filters.page_size) params.page_size = String(filters.page_size);

  return useQuery({
    queryKey: limitKeys.list(filters),
    queryFn: async () => {
      // The API returns individual rows per limit_type (RawCreditLimit[]).
      // We need to group them into one CreditLimit per counterparty.
      const res = await api.get<
        ApiResponse<{ data: RawCreditLimit[]; total: number; page: number; page_size: number; total_pages?: number }>
      >("/limits", { params });

      const payload = res.data;
      // Handle both shapes: payload might be the list wrapper directly,
      // or it might have a nested `data` array.
      const rawRows: RawCreditLimit[] = Array.isArray(payload)
        ? (payload as unknown as RawCreditLimit[])
        : Array.isArray(payload.data)
          ? payload.data
          : [];

      const grouped = transformLimits(rawRows);

      const total = Array.isArray(payload) ? grouped.length : (payload.total ?? grouped.length);
      const page = Array.isArray(payload) ? 1 : (payload.page ?? 1);
      const pageSize = Array.isArray(payload) ? 20 : (payload.page_size ?? 20);
      const totalPages = Array.isArray(payload)
        ? Math.ceil(grouped.length / 20)
        : (payload.total_pages ?? Math.ceil(total / pageSize));

      return {
        data: grouped,
        total,
        page,
        page_size: pageSize,
        total_pages: totalPages,
      } as CreditLimitListResponse;
    },
  });
}

export function useCreditLimit(counterpartyId: string) {
  return useQuery({
    queryKey: limitKeys.detail(counterpartyId),
    queryFn: async () => {
      const res = await api.get<ApiResponse<CreditLimit>>(
        `/limits/${counterpartyId}`
      );
      return res.data;
    },
    enabled: !!counterpartyId,
  });
}

export function useUpdateCreditLimit(counterpartyId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (data: UpdateCreditLimitRequest) => {
      const res = await api.put<ApiResponse<CreditLimit>>(
        `/limits/${counterpartyId}`,
        data
      );
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: limitKeys.lists() });
      queryClient.invalidateQueries({
        queryKey: limitKeys.detail(counterpartyId),
      });
    },
  });
}

export function useDailySummary(date: string) {
  return useQuery({
    queryKey: limitKeys.dailySummary(date),
    queryFn: async () => {
      const res = await api.get<ApiResponse<DailySummaryResponse>>(
        "/limits/daily-summary",
        { params: { date } }
      );
      return res.data;
    },
    enabled: !!date,
  });
}

export function useExportDailySummary() {
  return useMutation({
    mutationFn: async (params: { date: string; password: string }) => {
      const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "/api/v1";
      const response = await fetch(
        `${API_BASE_URL}/limits/daily-summary/export`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          credentials: "include",
          body: JSON.stringify(params),
        }
      );
      if (!response.ok) {
        const body = await response.json().catch(() => null);
        throw {
          error: body?.error || `Export failed (${response.status})`,
          status: response.status,
        };
      }
      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `Limit_Daily_Summary_${params.date}.xlsx`;
      a.click();
      window.URL.revokeObjectURL(url);
    },
  });
}

export function useLimitApprovals(filters: ApprovalFilters = {}) {
  const params: Record<string, string> = {};
  if (filters.status) params.status = filters.status;
  if (filters.deal_module) params.deal_module = filters.deal_module;
  if (filters.page) params.page = String(filters.page);
  if (filters.page_size) params.page_size = String(filters.page_size);

  return useQuery({
    queryKey: limitKeys.approvalList(filters),
    queryFn: async () => {
      const res = await api.get<ApiResponse<LimitApprovalListResponse>>(
        "/limits/approvals",
        { params }
      );
      return res.data;
    },
  });
}

export function useLimitUtilization(counterpartyId: string) {
  return useQuery({
    queryKey: limitKeys.utilization(counterpartyId),
    queryFn: async () => {
      const res = await api.get<ApiResponse<LimitUtilization[]>>(
        `/limits/utilization/${counterpartyId}`
      );
      return res.data;
    },
    enabled: !!counterpartyId,
  });
}

export function useApproveLimitDeal() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (data: ApproveRejectRequest) => {
      const res = await api.post<ApiResponse<unknown>>(
        "/limits/approve",
        data
      );
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: limitKeys.approvals() });
      queryClient.invalidateQueries({ queryKey: limitKeys.lists() });
    },
  });
}

export function useRejectLimitDeal() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (data: ApproveRejectRequest) => {
      const res = await api.post<ApiResponse<unknown>>(
        "/limits/reject",
        data
      );
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: limitKeys.approvals() });
      queryClient.invalidateQueries({ queryKey: limitKeys.lists() });
    },
  });
}

export function useApproveLimitHead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (data: ApproveRejectRequest) => {
      const res = await api.post<ApiResponse<unknown>>(
        "/limits/approve-head",
        data
      );
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: limitKeys.approvals() });
      queryClient.invalidateQueries({ queryKey: limitKeys.lists() });
    },
  });
}

export function useRejectLimitHead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (data: ApproveRejectRequest) => {
      const res = await api.post<ApiResponse<unknown>>(
        "/limits/reject-head",
        data
      );
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: limitKeys.approvals() });
      queryClient.invalidateQueries({ queryKey: limitKeys.lists() });
    },
  });
}
