"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";

// ─── Types ───────────────────��────────────────────────────────

export type BondCategory = "GOVERNMENT" | "FINANCIAL_INSTITUTION" | "CERTIFICATE_OF_DEPOSIT";
export type BondDirection = "BUY" | "SELL";
export type BondTransactionType = "REPO" | "REVERSE_REPO" | "OUTRIGHT" | "OTHER";
export type BondPortfolioType = "HTM" | "AFS" | "HFT";
export type BondConfirmationMethod = "EMAIL" | "REUTERS" | "OTHER";
export type BondContractPreparedBy = "INTERNAL" | "COUNTERPARTY";
export type BondStatus =
  | "OPEN"
  | "PENDING_L2_APPROVAL"
  | "REJECTED"
  | "PENDING_BOOKING"
  | "PENDING_CHIEF_ACCOUNTANT"
  | "COMPLETED"
  | "VOIDED_BY_ACCOUNTING"
  | "PENDING_CANCEL_L1"
  | "PENDING_CANCEL_L2"
  | "CANCELLED";

export interface BondDeal {
  id: string;
  deal_number: string;
  bond_category: BondCategory;
  trade_date: string;
  order_date?: string;
  value_date: string;
  direction: BondDirection;
  counterparty_id: string;
  counterparty_code: string;
  counterparty_name: string;
  transaction_type: BondTransactionType;
  transaction_type_other?: string;
  bond_catalog_id?: string;
  bond_code_manual?: string;
  bond_code_display: string;
  issuer: string;
  coupon_rate: string;
  issue_date?: string;
  maturity_date: string;
  quantity: number;
  face_value: string;
  discount_rate: string;
  clean_price: string;
  settlement_price: string;
  total_value: string;
  portfolio_type?: BondPortfolioType;
  payment_date: string;
  remaining_tenor_days: number;
  confirmation_method: BondConfirmationMethod;
  confirmation_other?: string;
  contract_prepared_by: BondContractPreparedBy;
  status: BondStatus;
  note?: string;
  cloned_from_id?: string;
  cancel_reason?: string;
  created_by: string;
  created_by_name: string;
  created_at: string;
  updated_at: string;
  version: number;
}

export interface BondDealListResponse {
  data: BondDeal[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export interface BondFilters {
  status?: string;
  bond_category?: string;
  direction?: string;
  counterparty_id?: string;
  from_date?: string;
  to_date?: string;
  deal_number?: string;
  exclude_cancelled?: boolean;
  page?: number;
  page_size?: number;
  sort_by?: string;
  sort_dir?: string;
}

export interface CreateBondDealRequest {
  bond_category: BondCategory;
  trade_date: string;
  order_date?: string;
  value_date: string;
  direction: BondDirection;
  counterparty_id: string;
  transaction_type: BondTransactionType;
  transaction_type_other?: string;
  bond_catalog_id?: string;
  bond_code_manual?: string;
  issuer: string;
  coupon_rate: number;
  issue_date?: string;
  maturity_date: string;
  quantity: number;
  face_value: number;
  discount_rate?: number;
  clean_price: number;
  settlement_price: number;
  total_value: number;
  portfolio_type?: BondPortfolioType;
  payment_date: string;
  remaining_tenor_days: number;
  confirmation_method: BondConfirmationMethod;
  confirmation_other?: string;
  contract_prepared_by: BondContractPreparedBy;
  note?: string;
}

export interface UpdateBondDealRequest extends Partial<CreateBondDealRequest> {
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

export interface BondInventoryItem {
  id: string;
  bond_code: string;
  bond_category: string;
  portfolio_type: string;
  available_quantity: number;
  acquisition_date: string;
  acquisition_price: string;
  version: number;
  updated_at: string;
  catalog_id?: string;
  catalog_issuer?: string;
  catalog_coupon_rate?: number;
  catalog_issue_date?: string;
  catalog_maturity_date?: string;
  catalog_face_value?: string;
  nominal_value: string;
  updated_by_name: string;
}

export interface BondCatalogItem {
  id: string;
  bond_code: string;
  issuer: string;
  coupon_rate: string;
  issue_date: string;
  maturity_date: string;
  face_value: string;
}

export interface CreateBondCatalogRequest {
  bond_code: string;
  issuer: string;
  coupon_rate: number;
  issue_date: string;
  maturity_date: string;
  face_value: number;
}

// ─── API Response Wrapper ─────────────────────────────────────

interface ApiResponse<T> {
  success: boolean;
  data: T;
  meta?: { request_id: string; timestamp: string };
}

// ─── Query Keys ─────��─────────────────────────────────────────

const bondKeys = {
  all: ["bonds"] as const,
  lists: () => [...bondKeys.all, "list"] as const,
  list: (filters: BondFilters) => [...bondKeys.lists(), filters] as const,
  details: () => [...bondKeys.all, "detail"] as const,
  detail: (id: string) => [...bondKeys.details(), id] as const,
  history: (id: string) => [...bondKeys.all, "history", id] as const,
  inventory: () => [...bondKeys.all, "inventory"] as const,
  catalog: () => [...bondKeys.all, "catalog"] as const,
};

// ─── Hooks ────────────────────────────────────────────────────

export function useBondDeals(filters: BondFilters = {}) {
  const params: Record<string, string> = {};
  if (filters.status) params.status = filters.status;
  if (filters.bond_category) params.bond_category = filters.bond_category;
  if (filters.direction) params.direction = filters.direction;
  if (filters.counterparty_id) params.counterparty_id = filters.counterparty_id;
  if (filters.from_date) params.from_date = filters.from_date;
  if (filters.to_date) params.to_date = filters.to_date;
  if (filters.deal_number) params.deal_number = filters.deal_number;
  if (filters.page) params.page = String(filters.page);
  if (filters.page_size) params.page_size = String(filters.page_size);
  if (filters.sort_by) params.sort_by = filters.sort_by;
  if (filters.sort_dir) params.sort_dir = filters.sort_dir;
  if (filters.exclude_cancelled === false) params.exclude_cancelled = "false";

  return useQuery({
    queryKey: bondKeys.list(filters),
    queryFn: async () => {
      const res = await api.get<ApiResponse<BondDealListResponse>>("/bonds", { params });
      return res.data;
    },
  });
}

export function useBondDeal(id: string) {
  return useQuery({
    queryKey: bondKeys.detail(id),
    queryFn: async () => {
      const res = await api.get<ApiResponse<BondDeal>>(`/bonds/${id}`);
      return res.data;
    },
    enabled: !!id,
  });
}

export function useBondInventory() {
  return useQuery({
    queryKey: bondKeys.inventory(),
    queryFn: async () => {
      const res = await api.get<ApiResponse<BondInventoryItem[]>>("/bonds/inventory");
      return res.data;
    },
  });
}

export function useCreateBondDeal() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (data: CreateBondDealRequest) => {
      const res = await api.post<ApiResponse<BondDeal>>("/bonds", data);
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: bondKeys.lists() });
    },
  });
}

export function useUpdateBondDeal(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (data: UpdateBondDealRequest) => {
      const res = await api.put<ApiResponse<BondDeal>>(`/bonds/${id}`, data);
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: bondKeys.lists() });
      queryClient.invalidateQueries({ queryKey: bondKeys.detail(id) });
    },
  });
}

export function useApproveBondDeal(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (data: ApproveRejectRequest) => {
      const res = await api.post<ApiResponse<BondDeal>>(`/bonds/${id}/approve`, data);
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: bondKeys.lists() });
      queryClient.invalidateQueries({ queryKey: bondKeys.detail(id) });
    },
  });
}

export function useRecallBondDeal(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (data: ReasonRequest) => {
      const res = await api.post<ApiResponse<BondDeal>>(`/bonds/${id}/recall`, data);
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: bondKeys.lists() });
      queryClient.invalidateQueries({ queryKey: bondKeys.detail(id) });
    },
  });
}

export function useCancelBondDeal(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (data: ReasonRequest) => {
      const res = await api.post<ApiResponse<BondDeal>>(`/bonds/${id}/cancel`, data);
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: bondKeys.lists() });
      queryClient.invalidateQueries({ queryKey: bondKeys.detail(id) });
    },
  });
}

export function useCancelApproveBondDeal(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (data: ApproveRejectRequest) => {
      const res = await api.post<ApiResponse<unknown>>(
        `/bonds/${id}/cancel-approve`,
        data
      );
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: bondKeys.lists() });
      queryClient.invalidateQueries({ queryKey: bondKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: bondKeys.history(id) });
    },
  });
}

export function useCloneBondDeal() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const res = await api.post<ApiResponse<BondDeal>>(`/bonds/${id}/clone`);
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: bondKeys.lists() });
    },
  });
}

export function useDeleteBondDeal() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.delete(`/bonds/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: bondKeys.lists() });
    },
  });
}

export function useBondHistory(dealId: string) {
  return useQuery({
    queryKey: bondKeys.history(dealId),
    queryFn: async () => {
      const res = await api.get<ApiResponse<ApprovalHistoryEntry[]>>(
        `/bonds/${dealId}/history`
      );
      return res.data;
    },
    enabled: !!dealId,
  });
}

// ─── Export ──────────────────────────────────────────────────

export function useExportBondDeals() {
  return useMutation({
    mutationFn: async (params: { from: string; to: string; password: string }) => {
      const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "/api/v1";
      const response = await fetch(`${API_BASE_URL}/bonds/deals/export`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify(params),
      });
      if (!response.ok) {
        const body = await response.json().catch(() => null);
        throw { error: body?.error || `Export failed (${response.status})`, status: response.status };
      }
      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `Treasury_Bond_Report_${params.from}_${params.to}.xlsx`;
      a.click();
      window.URL.revokeObjectURL(url);
    },
  });
}

// ─── Catalog ─────────────────────────────────────────────────

export function useBondCatalog() {
  return useQuery({
    queryKey: bondKeys.catalog(),
    queryFn: async () => {
      const res = await api.get<ApiResponse<BondCatalogItem[]>>("/bonds/catalog");
      return res.data;
    },
  });
}

export function useCreateBondCatalogItem() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (data: CreateBondCatalogRequest) => {
      const res = await api.post<ApiResponse<BondCatalogItem>>("/bonds/catalog", data);
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: bondKeys.catalog() });
    },
  });
}

export function useUpdateBondCatalogItem() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (data: { id: string } & CreateBondCatalogRequest) => {
      const { id, ...body } = data;
      const res = await api.put<ApiResponse<BondCatalogItem>>(`/bonds/catalog/${id}`, body);
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: bondKeys.catalog() });
    },
  });
}
