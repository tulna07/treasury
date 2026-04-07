"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";

// ─── Types ────────────────────────────────────────────────────

export interface MasterCounterparty {
  id: string;
  code: string;
  full_name: string;
  short_name: string;
  swift_code: string;
  country_code: string;
  cif: string;
  tax_id: string;
  address: string;
  fx_uses_limit: boolean;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CounterpartyFilters {
  search?: string;
  page?: number;
  page_size?: number;
}

export interface CounterpartyListResponse {
  data: MasterCounterparty[];
  total: number;
  page: number;
  page_size: number;
}

export interface CreateCounterpartyRequest {
  code: string;
  full_name: string;
  short_name: string;
  swift_code: string;
  country_code: string;
  cif: string;
}

export interface UpdateCounterpartyRequest {
  full_name?: string;
  short_name?: string;
  swift_code?: string;
  country_code?: string;
  tax_id?: string;
  address?: string;
  fx_uses_limit?: boolean;
}

export interface Currency {
  code: string;
  name: string;
}

export interface CurrencyPair {
  base: string;
  quote: string;
  code: string;
}

export interface Branch {
  id: string;
  code: string;
  name: string;
}

export interface ExchangeRate {
  currency_code: string;
  buy_rate: string;
  sell_rate: string;
  mid_rate: string;
  updated_at: string;
}

// ─── API Response Wrapper ─────────────────────────────────────

interface ApiResponse<T> {
  success: boolean;
  data: T;
}

// ─── Query Keys ───────────────────────────────────────────────

const masterKeys = {
  counterparties: ["master-counterparties"] as const,
  counterpartyList: (filters: CounterpartyFilters) =>
    [...masterKeys.counterparties, "list", filters] as const,
  counterpartyDetail: (id: string) =>
    [...masterKeys.counterparties, "detail", id] as const,
  currencies: ["currencies"] as const,
  currencyPairs: ["currency-pairs"] as const,
  branches: ["branches"] as const,
  exchangeRates: ["exchange-rates"] as const,
};

// ─── Hooks ────────────────────────────────────────────────────

export function useMasterCounterparties(filters: CounterpartyFilters = {}) {
  const params: Record<string, string> = {};
  if (filters.search) params.search = filters.search;
  if (filters.page) params.page = String(filters.page);
  if (filters.page_size) params.page_size = String(filters.page_size);

  return useQuery({
    queryKey: masterKeys.counterpartyList(filters),
    queryFn: async () => {
      const res = await api.get<ApiResponse<CounterpartyListResponse>>(
        "/counterparties",
        { params }
      );
      return res.data;
    },
  });
}

export function useMasterCounterparty(id: string) {
  return useQuery({
    queryKey: masterKeys.counterpartyDetail(id),
    queryFn: async () => {
      const res = await api.get<ApiResponse<MasterCounterparty>>(
        `/counterparties/${id}`
      );
      return res.data;
    },
    enabled: !!id,
  });
}

export function useCreateCounterparty() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (data: CreateCounterpartyRequest) => {
      const res = await api.post<ApiResponse<MasterCounterparty>>(
        "/counterparties",
        data
      );
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: masterKeys.counterparties });
    },
  });
}

export function useUpdateCounterparty(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (data: UpdateCounterpartyRequest) => {
      const res = await api.put<ApiResponse<MasterCounterparty>>(
        `/counterparties/${id}`,
        data
      );
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: masterKeys.counterparties });
    },
  });
}

export function useDeleteCounterparty() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/counterparties/${id}`);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: masterKeys.counterparties });
    },
  });
}

export function useCurrencies() {
  return useQuery({
    queryKey: masterKeys.currencies,
    queryFn: async () => {
      const res = await api.get<ApiResponse<Currency[]>>("/currencies");
      return res.data;
    },
  });
}

export function useCurrencyPairs() {
  return useQuery({
    queryKey: masterKeys.currencyPairs,
    queryFn: async () => {
      const res = await api.get<ApiResponse<CurrencyPair[]>>("/currency-pairs");
      return res.data;
    },
  });
}

export function useBranches() {
  return useQuery({
    queryKey: masterKeys.branches,
    queryFn: async () => {
      const res = await api.get<ApiResponse<Branch[]>>("/branches");
      return res.data;
    },
  });
}

export function useExchangeRates() {
  return useQuery({
    queryKey: masterKeys.exchangeRates,
    queryFn: async () => {
      const res = await api.get<ApiResponse<ExchangeRate[]>>("/exchange-rates");
      return res.data;
    },
  });
}
