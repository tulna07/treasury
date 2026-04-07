import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";

export interface DashboardSummary {
  total_deals_today: number;
  completed_today: number;
  pending_today: number;
  fx_total_notional: number;
  fx_buy_count: number;
  fx_sell_count: number;
  mm_outstanding: number;
  mm_active_count: number;
  bond_portfolio_value: number;
  settlements_pending_count: number;
  counterparties_with_limits: number;
}

export interface DailyVolume {
  trade_date: string;
  fx_count: number;
  fx_volume: number;
  mm_count: number;
  mm_volume: number;
  bond_count: number;
  bond_volume: number;
}

export interface ModuleDistribution {
  fx_count: number;
  bond_count: number;
  mm_count: number;
  settlements_count: number;
}

export interface StatusDaily {
  trade_date: string;
  open_count: number;
  pending_count: number;
  completed_count: number;
  cancelled_count: number;
}

export interface RecentTransaction {
  id: string;
  ticket: string;
  module: string;
  deal_type: string;
  amount: number;
  currency: string | null;
  status: string;
  created_at: string;
}

export interface DashboardData {
  summary: DashboardSummary;
  daily_volume: DailyVolume[];
  module_distribution: ModuleDistribution;
  status_daily: StatusDaily[];
  recent_transactions: RecentTransaction[];
}

export function useDashboard() {
  return useQuery({
    queryKey: ["dashboard"],
    queryFn: async () => {
      const res = await api.get<{ success: boolean; data: DashboardData }>("/dashboard");
      return res.data;
    },
    refetchInterval: 60_000, // auto-refresh every 60s
  });
}
