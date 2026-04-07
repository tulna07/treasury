"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";

// ─── Types ────────────────────────────────────────────────────

export type FxDealType = "SPOT" | "FORWARD" | "SWAP";
export type FxDirection = "BUY" | "SELL" | "BUY_SELL" | "SELL_BUY";
export type FxStatus =
  | "OPEN"
  | "PENDING_L1"
  | "PENDING_L2"
  | "PENDING_L2_APPROVAL"
  | "PENDING_CHIEF_ACCOUNTANT"
  | "PENDING_BOOKING"
  | "APPROVED"
  | "BOOKED_L1"
  | "BOOKED_L2"
  | "COMPLETED"
  | "SETTLED"
  | "REJECTED"
  | "CANCELLED"
  | "PENDING_CANCEL_L1"
  | "PENDING_CANCEL_L2";

export interface FxLeg {
  leg_number: number;
  value_date: string;
  exchange_rate: string;
  buy_currency: string;
  sell_currency: string;
  buy_amount: string;
  sell_amount: string;
  pay_code_klb?: string;
  pay_code_counterparty?: string;
  is_international?: boolean;
  execution_date?: string;
  settlement_amount?: string;
  settlement_currency?: string;
}

export interface FxDeal {
  id: string;
  ticket_number: string;
  counterparty_id: string;
  counterparty_code: string;
  counterparty_name: string;
  deal_type: FxDealType;
  direction: FxDirection;
  notional_amount: string;
  currency_code: string;
  trade_date: string;
  status: FxStatus;
  note: string;
  legs: FxLeg[];
  execution_date?: string;
  pay_code_klb?: string;
  pay_code_counterparty?: string;
  is_international?: boolean;
  settlement_amount?: string;
  settlement_currency?: string;
  attachment_path?: string;
  attachment_name?: string;
  attachments?: Array<{
    id: string;
    deal_module: string;
    deal_id: string;
    file_name: string;
    file_size: number;
    content_type: string;
    uploaded_by: string;
    created_at: string;
    download_url: string;
  }>;
  attachment_count?: number;
  created_by: string;
  created_at: string;
  updated_at: string;
  version: number;
}

export interface FxDealListResponse {
  data: FxDeal[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export interface FxFilters {
  status?: string;
  deal_type?: string;
  counterparty_id?: string;
  from_date?: string;
  to_date?: string;
  ticket_number?: string;
  exclude_cancelled?: boolean;
  page?: number;
  page_size?: number;
  sort_by?: string;
  sort_dir?: string;
}

export interface CreateFxLeg {
  leg_number: number;
  value_date: string;
  exchange_rate: number;
  buy_currency: string;
  sell_currency: string;
  buy_amount: number;
  sell_amount: number;
  pay_code_klb?: string;
  pay_code_counterparty?: string;
  execution_date?: string;
}

export interface CreateFxDealRequest {
  counterparty_id: string;
  deal_type: FxDealType;
  direction: FxDirection;
  notional_amount: number;
  currency_code: string;
  trade_date: string;
  execution_date?: string;
  pay_code_klb?: string;
  pay_code_counterparty?: string;
  note?: string;
  legs: CreateFxLeg[];
}

export interface UpdateFxDealRequest extends CreateFxDealRequest {
  version: number;
}

export interface ApproveRejectRequest {
  action: "APPROVE" | "REJECT";
  reason?: string;
}

export interface ReasonRequest {
  reason: string;
}

export interface ApprovalHistoryEntry {
  id: string;
  action_type: string;
  status_before: string;
  status_after: string;
  performed_by: string;
  performer_name: string;
  performed_at: string;
  reason: string;
}

// ─── API Response Wrapper ─────────────────────────────────────

interface ApiResponse<T> {
  success: boolean;
  data: T;
  meta?: { request_id: string; timestamp: string };
}

// ─── Query Keys ───────────────────────────────────────────────

const fxKeys = {
  all: ["fx"] as const,
  lists: () => [...fxKeys.all, "list"] as const,
  list: (filters: FxFilters) => [...fxKeys.lists(), filters] as const,
  details: () => [...fxKeys.all, "detail"] as const,
  detail: (id: string) => [...fxKeys.details(), id] as const,
  history: (id: string) => [...fxKeys.all, "history", id] as const,
};

// ─── Hooks ────────────────────────────────────────────────────

export function useFxDeals(filters: FxFilters = {}) {
  const params: Record<string, string> = {};
  if (filters.status) params.status = filters.status;
  if (filters.deal_type) params.deal_type = filters.deal_type;
  if (filters.counterparty_id) params.counterparty_id = filters.counterparty_id;
  if (filters.from_date) params.from_date = filters.from_date;
  if (filters.to_date) params.to_date = filters.to_date;
  if (filters.ticket_number) params.ticket_number = filters.ticket_number;
  if (filters.page) params.page = String(filters.page);
  if (filters.page_size) params.page_size = String(filters.page_size);
  if (filters.sort_by) params.sort_by = filters.sort_by;
  if (filters.sort_dir) params.sort_dir = filters.sort_dir;
  if (filters.exclude_cancelled === false) params.exclude_cancelled = "false";

  return useQuery({
    queryKey: fxKeys.list(filters),
    queryFn: async () => {
      const res = await api.get<ApiResponse<FxDealListResponse>>("/fx", { params });
      return res.data;
    },
  });
}

export function useFxDeal(id: string) {
  return useQuery({
    queryKey: fxKeys.detail(id),
    queryFn: async () => {
      const res = await api.get<ApiResponse<FxDeal>>(`/fx/${id}`);
      return res.data;
    },
    enabled: !!id,
  });
}

export function useCreateFxDeal() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (data: CreateFxDealRequest) => {
      const res = await api.post<ApiResponse<FxDeal>>("/fx", data);
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: fxKeys.lists() });
    },
  });
}

export function useUpdateFxDeal(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (data: UpdateFxDealRequest) => {
      const res = await api.put<ApiResponse<FxDeal>>(`/fx/${id}`, data);
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: fxKeys.lists() });
      queryClient.invalidateQueries({ queryKey: fxKeys.detail(id) });
    },
  });
}

export function useApproveFxDeal(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (data: ApproveRejectRequest) => {
      const res = await api.post<ApiResponse<FxDeal>>(`/fx/${id}/approve`, data);
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: fxKeys.lists() });
      queryClient.invalidateQueries({ queryKey: fxKeys.detail(id) });
    },
  });
}

export function useRecallFxDeal(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (data: ReasonRequest) => {
      const res = await api.post<ApiResponse<FxDeal>>(`/fx/${id}/recall`, data);
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: fxKeys.lists() });
      queryClient.invalidateQueries({ queryKey: fxKeys.detail(id) });
    },
  });
}

export function useCancelFxDeal(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (data: ReasonRequest) => {
      const res = await api.post<ApiResponse<FxDeal>>(`/fx/${id}/cancel`, data);
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: fxKeys.lists() });
      queryClient.invalidateQueries({ queryKey: fxKeys.detail(id) });
    },
  });
}

export function useCloneFxDeal() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const res = await api.post<ApiResponse<FxDeal>>(`/fx/${id}/clone`);
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: fxKeys.lists() });
    },
  });
}

export function useDeleteFxDeal() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.delete(`/fx/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: fxKeys.lists() });
    },
  });
}

export function useApprovalHistory(dealId: string) {
  return useQuery({
    queryKey: fxKeys.history(dealId),
    queryFn: async () => {
      const res = await api.get<ApiResponse<ApprovalHistoryEntry[]>>(
        `/fx/${dealId}/history`
      );
      return res.data;
    },
    enabled: !!dealId,
  });
}

export function useExportFxDeals() {
  return useMutation({
    mutationFn: async (params: { from: string; to: string; password: string }) => {
      const response = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL || "/api/v1"}/fx/deals/export`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          credentials: "include",
          body: JSON.stringify(params),
        }
      );
      if (!response.ok) {
        const body = await response.json().catch(() => null);
        throw { error: body?.error || `Export failed (${response.status})`, status: response.status };
      }
      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `Treasury_FX_Report_${params.from}_${params.to}.xlsx`;
      a.click();
      window.URL.revokeObjectURL(url);
    },
  });
}

export function useCancelApproveFxDeal(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (data: ApproveRejectRequest) => {
      const res = await api.post<ApiResponse<unknown>>(
        `/fx/${id}/cancel-approve`,
        data
      );
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: fxKeys.lists() });
      queryClient.invalidateQueries({ queryKey: fxKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: fxKeys.history(id) });
    },
  });
}
