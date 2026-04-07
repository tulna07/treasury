"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";

// ─── Types ─────────────────────────────────────────────────────

export type MMDealStatus =
  | "OPEN"
  | "PENDING_L2_APPROVAL"
  | "PENDING_TP_REVIEW"
  | "PENDING_BOOKING"
  | "PENDING_CHIEF_ACCOUNTANT"
  | "PENDING_RISK_APPROVAL"
  | "PENDING_SETTLEMENT"
  | "COMPLETED"
  | "REJECTED"
  | "VOIDED_BY_ACCOUNTING"
  | "VOIDED_BY_SETTLEMENT"
  | "VOIDED_BY_RISK"
  | "CANCELLED"
  | "PENDING_CANCEL_L1"
  | "PENDING_CANCEL_L2";

export type MMInterbankDirection = "PLACE" | "TAKE" | "LEND" | "BORROW";

export interface MMInterbankDeal {
  id: string;
  deal_number: string;
  ticket_number?: string;
  counterparty_id: string;
  counterparty_code: string;
  counterparty_name: string;
  branch_code?: string;
  branch_name?: string;
  currency_code: string;
  direction: MMInterbankDirection;
  principal_amount: string;
  interest_rate: string;
  day_count_convention: string;
  trade_date: string;
  effective_date: string;
  tenor_days: number;
  maturity_date: string;
  interest_amount: string;
  maturity_amount: string;
  has_collateral: boolean;
  collateral_currency?: string;
  collateral_description?: string;
  requires_international_settlement: boolean;
  status: MMDealStatus;
  note?: string;
  cloned_from_id?: string;
  cancel_reason?: string;
  created_by: string;
  created_by_name: string;
  created_at: string;
  updated_at: string;
  version: number;
}

export interface MMOMORepoDeal {
  id: string;
  deal_number: string;
  deal_subtype: "OMO" | "STATE_REPO";
  session_name: string;
  trade_date: string;
  counterparty_id: string;
  counterparty_code: string;
  counterparty_name: string;
  branch_code?: string;
  branch_name?: string;
  notional_amount: string;
  bond_catalog_id: string;
  bond_code: string;
  bond_issuer: string;
  bond_coupon_rate: string;
  bond_maturity_date?: string;
  winning_rate: string;
  tenor_days: number;
  settlement_date_1: string;
  settlement_date_2: string;
  haircut_pct: string;
  status: MMDealStatus;
  note?: string;
  cloned_from_id?: string;
  cancel_reason?: string;
  created_by: string;
  created_by_name: string;
  created_at: string;
  updated_at: string;
  version: number;
}

export interface MMListResponse<T> {
  data: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export interface MMFilters {
  status?: string;
  direction?: string;
  currency_code?: string;
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

export interface ApproveRejectRequest {
  action: "APPROVE" | "REJECT";
  reason?: string;
}

export interface ReasonRequest {
  reason: string;
}

// ─── API Response Wrapper ─────────────────────────────────────

interface ApiResponse<T> {
  success: boolean;
  data: T;
  meta?: { request_id: string; timestamp: string };
}

// ─── Query Keys ───────────────────────────────────────────────

export const mmKeys = {
  all: ["mm"] as const,
  interbank: () => [...mmKeys.all, "interbank"] as const,
  interbankLists: () => [...mmKeys.interbank(), "list"] as const,
  interbankList: (filters: MMFilters) => [...mmKeys.interbankLists(), filters] as const,
  interbankDetail: (id: string) => [...mmKeys.interbank(), "detail", id] as const,
  interbankHistory: (id: string) => [...mmKeys.interbank(), "history", id] as const,
  omo: () => [...mmKeys.all, "omo"] as const,
  omoLists: () => [...mmKeys.omo(), "list"] as const,
  omoList: (filters: MMFilters) => [...mmKeys.omoLists(), filters] as const,
  omoDetail: (id: string) => [...mmKeys.omo(), "detail", id] as const,
  omoHistory: (id: string) => [...mmKeys.omo(), "history", id] as const,
  repo: () => [...mmKeys.all, "repo"] as const,
  repoLists: () => [...mmKeys.repo(), "list"] as const,
  repoList: (filters: MMFilters) => [...mmKeys.repoLists(), filters] as const,
  repoDetail: (id: string) => [...mmKeys.repo(), "detail", id] as const,
  repoHistory: (id: string) => [...mmKeys.repo(), "history", id] as const,
};

// ─── Helpers ──────────────────────────────────────────────────

function buildParams(filters: MMFilters): Record<string, string> {
  const params: Record<string, string> = {};
  if (filters.status) params.status = filters.status;
  if (filters.direction) params.direction = filters.direction;
  if (filters.currency_code) params.currency_code = filters.currency_code;
  if (filters.counterparty_id) params.counterparty_id = filters.counterparty_id;
  if (filters.from_date) params.from_date = filters.from_date;
  if (filters.to_date) params.to_date = filters.to_date;
  if (filters.deal_number) params.deal_number = filters.deal_number;
  if (filters.page) params.page = String(filters.page);
  if (filters.page_size) params.page_size = String(filters.page_size);
  if (filters.sort_by) params.sort_by = filters.sort_by;
  if (filters.sort_dir) params.sort_dir = filters.sort_dir;
  if (filters.exclude_cancelled === false) params.exclude_cancelled = "false";
  return params;
}

// ─── Interbank Hooks ──────────────────────────────────────────

export function useMMInterbankDeals(filters: MMFilters = {}) {
  return useQuery({
    queryKey: mmKeys.interbankList(filters),
    queryFn: async () => {
      const res = await api.get<ApiResponse<MMListResponse<MMInterbankDeal>>>("/mm/interbank", {
        params: buildParams(filters),
      });
      return res.data;
    },
  });
}

export function useMMInterbankDeal(id: string) {
  return useQuery({
    queryKey: mmKeys.interbankDetail(id),
    queryFn: async () => {
      const res = await api.get<ApiResponse<MMInterbankDeal>>(`/mm/interbank/${id}`);
      return res.data;
    },
    enabled: !!id,
  });
}

export function useCreateMMInterbank() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (data: Record<string, unknown>) => {
      const res = await api.post<ApiResponse<MMInterbankDeal>>("/mm/interbank", data);
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: mmKeys.interbankLists() });
    },
  });
}

export function useApproveMMInterbank(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (data: ApproveRejectRequest) => {
      const res = await api.post<ApiResponse<MMInterbankDeal>>(`/mm/interbank/${id}/approve`, data);
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: mmKeys.interbankLists() });
      queryClient.invalidateQueries({ queryKey: mmKeys.interbankDetail(id) });
    },
  });
}

export function useDeleteMMInterbank() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.delete(`/mm/interbank/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: mmKeys.interbankLists() });
    },
  });
}

// ─── OMO Hooks ────────────────────────────────────────────────

export function useMMOMODeals(filters: MMFilters = {}) {
  return useQuery({
    queryKey: mmKeys.omoList(filters),
    queryFn: async () => {
      const res = await api.get<ApiResponse<MMListResponse<MMOMORepoDeal>>>("/mm/omo", {
        params: buildParams(filters),
      });
      return res.data;
    },
  });
}

export function useMMOMODeal(id: string) {
  return useQuery({
    queryKey: mmKeys.omoDetail(id),
    queryFn: async () => {
      const res = await api.get<ApiResponse<MMOMORepoDeal>>(`/mm/omo/${id}`);
      return res.data;
    },
    enabled: !!id,
  });
}

export function useCreateMMOMO() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (data: Record<string, unknown>) => {
      const res = await api.post<ApiResponse<MMOMORepoDeal>>("/mm/omo", data);
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: mmKeys.omoLists() });
    },
  });
}

export function useApproveMMOMO(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (data: ApproveRejectRequest) => {
      const res = await api.post<ApiResponse<MMOMORepoDeal>>(`/mm/omo/${id}/approve`, data);
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: mmKeys.omoLists() });
      queryClient.invalidateQueries({ queryKey: mmKeys.omoDetail(id) });
    },
  });
}

export function useDeleteMMOMO() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.delete(`/mm/omo/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: mmKeys.omoLists() });
    },
  });
}

// ─── Repo KBNN Hooks ──────────────────────────────────────────

export function useMMRepoDeals(filters: MMFilters = {}) {
  return useQuery({
    queryKey: mmKeys.repoList(filters),
    queryFn: async () => {
      const res = await api.get<ApiResponse<MMListResponse<MMOMORepoDeal>>>("/mm/govt-repo", {
        params: buildParams(filters),
      });
      return res.data;
    },
  });
}

export function useMMRepoDeal(id: string) {
  return useQuery({
    queryKey: mmKeys.repoDetail(id),
    queryFn: async () => {
      const res = await api.get<ApiResponse<MMOMORepoDeal>>(`/mm/govt-repo/${id}`);
      return res.data;
    },
    enabled: !!id,
  });
}

export function useCreateMMRepo() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (data: Record<string, unknown>) => {
      const res = await api.post<ApiResponse<MMOMORepoDeal>>("/mm/govt-repo", data);
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: mmKeys.repoLists() });
    },
  });
}

export function useApproveMMRepo(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (data: ApproveRejectRequest) => {
      const res = await api.post<ApiResponse<MMOMORepoDeal>>(`/mm/govt-repo/${id}/approve`, data);
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: mmKeys.repoLists() });
      queryClient.invalidateQueries({ queryKey: mmKeys.repoDetail(id) });
    },
  });
}

export function useDeleteMMRepo() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.delete(`/mm/govt-repo/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: mmKeys.repoLists() });
    },
  });
}

// ─── Approval History ──────────────────────────────────────────

export interface MMApprovalHistoryEntry {
  id: string;
  action_type: string;
  from_status: string;
  to_status: string;
  performer_id: string;
  performer_name: string;
  reason?: string;
  performed_at: string;
}

export function useMMOMOHistory(id: string) {
  return useQuery({
    queryKey: mmKeys.omoHistory(id),
    queryFn: async () => {
      const res = await api.get<ApiResponse<MMApprovalHistoryEntry[]>>(`/mm/omo/${id}/history`);
      return res.data;
    },
    enabled: !!id,
  });
}

export function useMMRepoHistory(id: string) {
  return useQuery({
    queryKey: mmKeys.repoHistory(id),
    queryFn: async () => {
      const res = await api.get<ApiResponse<MMApprovalHistoryEntry[]>>(`/mm/govt-repo/${id}/history`);
      return res.data;
    },
    enabled: !!id,
  });
}

export function useMMInterbankHistory(id: string) {
  return useQuery({
    queryKey: mmKeys.interbankHistory(id),
    queryFn: async () => {
      const res = await api.get<ApiResponse<MMApprovalHistoryEntry[]>>(`/mm/interbank/${id}/history`);
      return res.data;
    },
    enabled: !!id,
  });
}

// ─── Export ─────────────────────────────────────────────────────

export function useExportMMInterbankDeals() {
  return useMutation({
    mutationFn: async (params: { from: string; to: string; password: string }) => {
      const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "/api/v1";
      const response = await fetch(`${API_BASE_URL}/mm/interbank/export`, {
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
      a.download = `MM-Interbank-${params.from}_${params.to}.xlsx`;
      a.click();
      window.URL.revokeObjectURL(url);
    },
  });
}
