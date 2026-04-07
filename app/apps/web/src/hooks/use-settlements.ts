import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";

export interface InternationalPayment {
  id: string;
  source_module: string;
  source_deal_id: string;
  source_leg_number?: number;
  ticket_display: string;
  counterparty_id: string;
  counterparty_code: string;
  counterparty_name: string;
  debit_account: string;
  bic_code?: string;
  currency_code: string;
  amount: string;
  transfer_date: string;
  counterparty_ssi: string;
  original_trade_date: string;
  approved_by_division?: string;
  settlement_status: string;
  settled_by?: string;
  settled_by_name?: string;
  settled_at?: string;
  rejection_reason?: string;
  created_at: string;
}

interface ApiResponse<T> {
  data: T;
  total?: number;
  page?: number;
  page_size?: number;
  total_pages?: number;
  has_more?: boolean;
}

export interface SettlementFilters {
  status?: string;
  source_module?: string;
  transfer_date_from?: string;
  transfer_date_to?: string;
  page?: number;
  page_size?: number;
}

const keys = {
  all: ["settlements"] as const,
  list: (filters: SettlementFilters) => [...keys.all, "list", filters] as const,
  detail: (id: string) => [...keys.all, "detail", id] as const,
};

export function useSettlements(filters: SettlementFilters = {}) {
  return useQuery({
    queryKey: keys.list(filters),
    queryFn: async () => {
      const params: Record<string, string> = {};
      if (filters.status) params.status = filters.status;
      if (filters.source_module) params.source_module = filters.source_module;
      if (filters.transfer_date_from) params.transfer_date_from = filters.transfer_date_from;
      if (filters.transfer_date_to) params.transfer_date_to = filters.transfer_date_to;
      if (filters.page) params.page = String(filters.page);
      if (filters.page_size) params.page_size = String(filters.page_size);
      const res = await api.get<ApiResponse<InternationalPayment[]>>("/settlements", { params });
      return res;
    },
  });
}

export function useSettlement(id: string) {
  return useQuery({
    queryKey: keys.detail(id),
    queryFn: async () => {
      const res = await api.get<ApiResponse<InternationalPayment>>(`/settlements/${id}`);
      return res.data;
    },
    enabled: !!id,
  });
}

export function useApproveSettlement() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      return api.post(`/settlements/${id}/approve`, {});
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: keys.all }),
  });
}

export function useRejectSettlement() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, reason }: { id: string; reason: string }) => {
      return api.post(`/settlements/${id}/reject`, { reason });
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: keys.all }),
  });
}
