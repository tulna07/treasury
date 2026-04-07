"use client";

import { useMemo } from "react";
import Link from "next/link";
import { Header } from "@/components/header";
import { useTranslation } from "@/lib/i18n";
import {
  IconArrowUpRight,
  IconCurrencyDollar,
  IconCash,
  IconWorld,
  IconTrendingUp,
  IconTrendingDown,
  IconAlertTriangle,
  IconLoader2,
} from "@tabler/icons-react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { cn } from "@/lib/utils";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  ChartLegend,
  ChartLegendContent,
  type ChartConfig,
} from "@/components/ui/chart";
import {
  AreaChart,
  Area,
  BarChart,
  Bar,
  PieChart,
  Pie,
  Cell,
  XAxis,
  YAxis,
  CartesianGrid,
} from "recharts";
import { useDashboard } from "@/hooks/use-dashboard";

// ─── Helpers ────────────────────────────────────────────────
function getTxDetailUrl(tx: { id: string; module: string; deal_type?: string }): string {
  switch (tx.module) {
    case "FX": return `/fx/${tx.id}`;
    case "BOND": return `/bonds/${tx.id}`;
    case "MM": {
      const dt = (tx.deal_type || "").toUpperCase();
      if (dt.includes("OMO") || dt === "OMO") return `/mm/omo/${tx.id}`;
      if (dt.includes("REPO") || dt.includes("STATE_REPO")) return `/mm/repo/${tx.id}`;
      return `/mm/interbank/${tx.id}`;
    }
    case "SETTLEMENT": return `/settlements/${tx.id}`;
    default: return "#";
  }
}

// Status key mapping — tries fx.status.*, then falls back to readable format
const STATUS_KEY_MAP: Record<string, string> = {
  OPEN: "fx.status.OPEN",
  PENDING_TP_REVIEW: "fx.status.PENDING_TP_REVIEW",
  PENDING_L2_APPROVAL: "fx.status.PENDING_L2_APPROVAL",
  PENDING_RISK_APPROVAL: "fx.status.PENDING_RISK_APPROVAL",
  REJECTED: "fx.status.REJECTED",
  PENDING_BOOKING: "fx.status.PENDING_BOOKING",
  PENDING_CHIEF_ACCOUNTANT: "fx.status.PENDING_CHIEF_ACCOUNTANT",
  PENDING_SETTLEMENT: "fx.status.PENDING_SETTLEMENT",
  COMPLETED: "fx.status.COMPLETED",
  CANCELLED: "fx.status.CANCELLED",
  PENDING: "fx.status.OPEN",
  APPROVED: "fx.status.APPROVED",
  VOIDED_BY_ACCOUNTING: "fx.status.VOIDED_BY_ACCOUNTING",
  VOIDED_BY_RISK: "fx.status.VOIDED_BY_RISK",
  PENDING_CANCEL_L1: "fx.status.PENDING_CANCEL_L1",
  PENDING_CANCEL_L2: "fx.status.PENDING_CANCEL_L2",
};

function getStatusLabel(status: string, t: (key: string) => string): string {
  const key = STATUS_KEY_MAP[status];
  if (key) return t(key);
  return status.replace(/_/g, " ");
}

// Deal type key mapping — maps first word to i18n key
const DEAL_TYPE_KEY_MAP: Record<string, string> = {
  SPOT: "fx.type.SPOT",
  FORWARD: "fx.type.FORWARD",
  SWAP: "fx.type.SWAP",
  BUY: "fx.direction.BUY",
  SELL: "fx.direction.SELL",
  BUY_SELL: "fx.direction.BUY_SELL",
  SELL_BUY: "fx.direction.SELL_BUY",
  GOVERNMENT: "bond.category.GOVERNMENT",
  CORPORATE: "bond.category.FINANCIAL_INSTITUTION",
  CERTIFICATE_OF_DEPOSIT: "bond.category.CERTIFICATE_OF_DEPOSIT",
  PLACE: "mm.direction.PLACE",
  TAKE: "mm.direction.TAKE",
  LEND: "mm.direction.LEND",
  BORROW: "mm.direction.BORROW",
  OMO: "mm.omo",
  STATE_REPO: "mm.repoKbnn",
};

function getDealTypeLabel(dealType: string, t: (key: string) => string): string {
  // deal_type can be "BUY USD" or "PLACE VND" — split and translate first word
  const parts = dealType.split(" ");
  const key = DEAL_TYPE_KEY_MAP[parts[0]];
  const label = key ? t(key) : parts[0];
  return parts.length > 1 ? `${label} ${parts.slice(1).join(" ")}` : label;
}

function formatTy(value: number, t: (key: string) => string): string {
  return (value / 1_000_000_000).toFixed(0) + " " + t("dashboard.unit.billion");
}

function formatDate(dateStr: string): string {
  const d = new Date(dateStr);
  return `${String(d.getDate()).padStart(2, "0")}/${String(d.getMonth() + 1).padStart(2, "0")}`;
}

function formatAmount(amount: number, currency: string | null, t: (key: string) => string): string {
  if (amount >= 1_000_000_000) {
    return formatTy(amount, t) + (currency ? ` ${currency}` : "");
  }
  return new Intl.NumberFormat("vi-VN").format(amount) + (currency ? ` ${currency}` : "");
}

// ─── Multi-metric Stats Card ─────────────────────────────────
interface Metric {
  label: string;
  value: string | number;
  change?: number;
}

interface DashboardCardProps {
  title: string;
  icon: React.ElementType;
  variant: "default" | "success" | "danger" | "warning" | "info";
  metrics: Metric[];
}

const variantStyles = {
  default: {
    gradient: "from-primary/5 to-card dark:from-primary/10 dark:to-card",
    icon: "bg-primary/10 text-primary",
  },
  success: {
    gradient: "from-emerald-500/8 to-card dark:from-emerald-500/15 dark:to-card",
    icon: "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400",
  },
  danger: {
    gradient: "from-red-500/8 to-card dark:from-red-500/15 dark:to-card",
    icon: "bg-red-500/10 text-red-600 dark:text-red-400",
  },
  warning: {
    gradient: "from-amber-500/8 to-card dark:from-amber-500/15 dark:to-card",
    icon: "bg-amber-500/10 text-amber-600 dark:text-amber-400",
  },
  info: {
    gradient: "from-blue-500/8 to-card dark:from-blue-500/15 dark:to-card",
    icon: "bg-blue-500/10 text-blue-600 dark:text-blue-400",
  },
};

function DashboardCard({ title, icon: Icon, variant, metrics }: DashboardCardProps) {
  const styles = variantStyles[variant];
  const primary = metrics[0];
  const secondary = metrics.slice(1);

  return (
    <Card className={cn("@container/card bg-gradient-to-t shadow-xs", styles.gradient)}>
      <CardHeader className="pb-2">
        <div className="flex items-start justify-between gap-3">
          <CardDescription className="text-xs font-medium">{title}</CardDescription>
          <div className={cn("flex size-9 items-center justify-center rounded-xl shrink-0", styles.icon)}>
            <Icon className="size-5" />
          </div>
        </div>
        <div className="flex items-baseline gap-2 mt-1">
          <CardTitle className="text-2xl @[250px]/card:text-3xl font-semibold tabular-nums">
            {primary.value}
          </CardTitle>
          {primary.change !== undefined && (
            <Badge variant="outline" className="gap-0.5 text-[10px] px-1.5 py-0">
              {primary.change >= 0 ? (
                <IconTrendingUp className="size-3" />
              ) : (
                <IconTrendingDown className="size-3" />
              )}
              {primary.change >= 0 ? "+" : ""}
              {primary.change}%
            </Badge>
          )}
        </div>
        <p className="text-xs text-muted-foreground">{primary.label}</p>
      </CardHeader>
      {secondary.length > 0 && (
        <CardContent className="pt-0">
          <Separator className="mb-3" />
          <div className={cn(
            "grid gap-3",
            secondary.length === 1 ? "grid-cols-1" : "grid-cols-2"
          )}>
            {secondary.map((m, i) => (
              <div key={i}>
                <div className="flex items-baseline gap-1.5">
                  <span className="text-lg font-semibold tabular-nums">{m.value}</span>
                  {m.change !== undefined && (
                    <span className={cn(
                      "text-[10px] font-medium",
                      m.change >= 0 ? "text-emerald-600 dark:text-emerald-400" : "text-red-600 dark:text-red-400"
                    )}>
                      {m.change >= 0 ? "↑" : "↓"}{Math.abs(m.change)}%
                    </span>
                  )}
                </div>
                <p className="text-[11px] text-muted-foreground leading-tight">{m.label}</p>
              </div>
            ))}
          </div>
        </CardContent>
      )}
    </Card>
  );
}

// ─── Status colors ─────────────────────────────────────────
const statusColors: Record<string, string> = {
  OPEN: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300",
  PENDING_TP_REVIEW: "bg-cyan-100 text-cyan-800 dark:bg-cyan-900 dark:text-cyan-300",
  PENDING_L2_APPROVAL: "bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-300",
  PENDING_RISK_APPROVAL: "bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-300",
  PENDING_BOOKING: "bg-indigo-100 text-indigo-800 dark:bg-indigo-900 dark:text-indigo-300",
  PENDING_CHIEF_ACCOUNTANT: "bg-violet-100 text-violet-800 dark:bg-violet-900 dark:text-violet-300",
  PENDING_SETTLEMENT: "bg-teal-100 text-teal-800 dark:bg-teal-900 dark:text-teal-300",
  PENDING: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300",
  COMPLETED: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300",
  APPROVED: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300",
  REJECTED: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300",
  CANCELLED: "bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300",
  VOIDED_BY_ACCOUNTING: "bg-rose-100 text-rose-800 dark:bg-rose-900 dark:text-rose-300",
  VOIDED_BY_RISK: "bg-rose-100 text-rose-800 dark:bg-rose-900 dark:text-rose-300",
  PENDING_CANCEL_L1: "bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-300",
  PENDING_CANCEL_L2: "bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-300",
};

const PIE_COLORS: Record<string, string> = {
  fx: "hsl(24, 87%, 54%)",
  bond: "hsl(195, 74%, 54%)",
  mm: "hsl(207, 71%, 48%)",
  settlements: "hsl(14, 87%, 54%)",
};

// ─── Loading Skeleton ───────────────────────────────────────
function DashboardSkeleton() {
  return (
    <>
      <Header />
      <div className="flex flex-1 flex-col gap-6 p-4 md:p-6 pt-0">
        <div className="pt-4">
          <div className="h-8 w-48 bg-muted animate-pulse rounded" />
          <div className="h-4 w-64 bg-muted animate-pulse rounded mt-2" />
        </div>
        <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Card key={i} className="shadow-xs">
              <CardHeader className="pb-2">
                <div className="h-4 w-24 bg-muted animate-pulse rounded" />
                <div className="h-8 w-16 bg-muted animate-pulse rounded mt-2" />
                <div className="h-3 w-20 bg-muted animate-pulse rounded mt-1" />
              </CardHeader>
              <CardContent className="pt-0">
                <Separator className="mb-3" />
                <div className="grid grid-cols-2 gap-3">
                  <div className="h-6 bg-muted animate-pulse rounded" />
                  <div className="h-6 bg-muted animate-pulse rounded" />
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
        <div className="grid gap-4 md:grid-cols-7">
          <Card className="md:col-span-4">
            <CardContent className="pt-6">
              <div className="h-[300px] bg-muted animate-pulse rounded" />
            </CardContent>
          </Card>
          <Card className="md:col-span-3">
            <CardContent className="pt-6">
              <div className="h-[300px] bg-muted animate-pulse rounded" />
            </CardContent>
          </Card>
        </div>
      </div>
    </>
  );
}

// ─── Error State ────────────────────────────────────────────
function DashboardError({ message, onRetry }: { message: string; onRetry: () => void }) {
  const { t } = useTranslation();
  return (
    <>
      <Header />
      <div className="flex flex-1 flex-col items-center justify-center gap-4 p-4 md:p-6 min-h-[60vh]">
        <div className="flex flex-col items-center gap-3 text-center">
          <div className="flex size-16 items-center justify-center rounded-full bg-destructive/10">
            <IconAlertTriangle className="size-8 text-destructive" />
          </div>
          <h2 className="text-lg font-semibold">{t("dashboard.error.loadFailed")}</h2>
          <p className="text-sm text-muted-foreground max-w-md">{message}</p>
          <button
            onClick={onRetry}
            className="mt-2 inline-flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
          >
            <IconLoader2 className="size-4" />
            {t("dashboard.error.retry")}
          </button>
        </div>
      </div>
    </>
  );
}

// ─── Dashboard Page ─────────────────────────────────────────
export default function DashboardPage() {
  const { t } = useTranslation();
  const { data, isLoading, isError, error, refetch } = useDashboard();

  // Chart configs — inside component so labels react to locale changes
  const weeklyChartConfig = useMemo<ChartConfig>(() => ({
    fx: { label: t("dashboard.chart.fxFull"), color: "hsl(24, 87%, 54%)" },
    mm: { label: t("dashboard.chart.mmFull"), color: "hsl(207, 71%, 48%)" },
  }), [t]);

  const pieChartConfig = useMemo<ChartConfig>(() => ({
    fx: { label: t("dashboard.chart.fx"), color: "hsl(24, 87%, 54%)" },
    bond: { label: t("dashboard.chart.bond"), color: "hsl(195, 74%, 54%)" },
    mm: { label: t("dashboard.chart.mm"), color: "hsl(207, 71%, 48%)" },
    settlements: { label: t("dashboard.chart.settlements"), color: "hsl(14, 87%, 54%)" },
  }), [t]);

  const statusChartConfig = useMemo<ChartConfig>(() => ({
    open: { label: t("dashboard.chart.open"), color: "hsl(217, 91%, 60%)" },
    pending: { label: t("dashboard.chart.pending"), color: "hsl(45, 93%, 47%)" },
    completed: { label: t("dashboard.chart.completed"), color: "hsl(142, 71%, 45%)" },
    cancelled: { label: t("dashboard.chart.cancelled"), color: "hsl(0, 84%, 60%)" },
  }), [t]);

  if (isLoading) {
    return <DashboardSkeleton />;
  }

  if (isError || !data) {
    return (
      <DashboardError
        message={(error as { error?: string })?.error || t("dashboard.error.loadFailed")}
        onRetry={() => refetch()}
      />
    );
  }

  const { summary, daily_volume, module_distribution, status_daily, recent_transactions } = data;

  // Transform data for charts
  const weeklyData = daily_volume.map((d) => ({
    date: formatDate(d.trade_date),
    fx: +(d.fx_volume / 1e9).toFixed(1),
    mm: +(d.mm_volume / 1e9).toFixed(1),
  }));

  const pieData = [
    { name: "fx", value: module_distribution.fx_count, fill: PIE_COLORS.fx },
    { name: "bond", value: module_distribution.bond_count, fill: PIE_COLORS.bond },
    { name: "mm", value: module_distribution.mm_count, fill: PIE_COLORS.mm },
    { name: "settlements", value: module_distribution.settlements_count, fill: PIE_COLORS.settlements },
  ];

  const statusData = status_daily.map((d) => ({
    date: formatDate(d.trade_date),
    open: d.open_count,
    pending: d.pending_count,
    completed: d.completed_count,
    cancelled: d.cancelled_count,
  }));

  return (
    <>
      <Header />
      <div className="flex flex-1 flex-col gap-6 p-4 md:p-6 pt-0">
        <div className="pt-4">
          <h1 className="text-2xl font-bold tracking-tight">{t("dashboard.title")}</h1>
          <p className="text-muted-foreground">{t("dashboard.description")}</p>
        </div>

        {/* 4 Stats Cards */}
        <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
          <DashboardCard
            title={t("dashboard.card.todayOverview")}
            icon={IconArrowUpRight}
            variant="info"
            metrics={[
              { label: t("dashboard.metric.totalDeals"), value: summary.total_deals_today },
              { label: t("dashboard.metric.completed"), value: summary.completed_today },
              { label: t("dashboard.metric.pending"), value: summary.pending_today },
            ]}
          />
          <DashboardCard
            title={t("dashboard.card.fx")}
            icon={IconCurrencyDollar}
            variant="warning"
            metrics={[
              { label: t("dashboard.metric.totalNotional"), value: formatTy(summary.fx_total_notional, t) },
              { label: t("dashboard.metric.fxBuy"), value: summary.fx_buy_count },
              { label: t("dashboard.metric.fxSell"), value: summary.fx_sell_count },
            ]}
          />
          <DashboardCard
            title={t("dashboard.card.mmBond")}
            icon={IconCash}
            variant="success"
            metrics={[
              { label: t("dashboard.metric.mmOutstanding"), value: formatTy(summary.mm_outstanding, t) },
              { label: t("dashboard.metric.mmDeals"), value: summary.mm_active_count },
              { label: t("dashboard.metric.bondPortfolio"), value: formatTy(summary.bond_portfolio_value, t) },
            ]}
          />
          <DashboardCard
            title={t("dashboard.card.limitsSettlement")}
            icon={IconWorld}
            variant="danger"
            metrics={[
              { label: t("dashboard.metric.partnersWithLimits"), value: summary.counterparties_with_limits },
              { label: t("dashboard.metric.pendingSettlements"), value: summary.settlements_pending_count },
            ]}
          />
        </div>

        {/* Charts Row 1 */}
        <div className="grid gap-4 md:grid-cols-7">
          <Card className="md:col-span-4">
            <CardHeader>
              <CardTitle>{t("dashboard.chart.weeklyTitle")}</CardTitle>
              <CardDescription>{t("dashboard.chart.weeklyUnit")}</CardDescription>
            </CardHeader>
            <CardContent>
              <ChartContainer config={weeklyChartConfig} className="h-[300px] w-full">
                <AreaChart data={weeklyData} margin={{ top: 10, right: 10, left: 0, bottom: 0 }}>
                  <defs>
                    <linearGradient id="fillFx" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="var(--color-fx)" stopOpacity={0.8} />
                      <stop offset="95%" stopColor="var(--color-fx)" stopOpacity={0.1} />
                    </linearGradient>
                    <linearGradient id="fillMm" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="var(--color-mm)" stopOpacity={0.8} />
                      <stop offset="95%" stopColor="var(--color-mm)" stopOpacity={0.1} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid vertical={false} className="stroke-muted" />
                  <XAxis dataKey="date" tickLine={false} axisLine={false} tickMargin={8} className="text-xs fill-muted-foreground" />
                  <YAxis tickLine={false} axisLine={false} tickMargin={8} className="text-xs fill-muted-foreground" />
                  <ChartTooltip content={<ChartTooltipContent />} />
                  <ChartLegend content={<ChartLegendContent />} />
                  <Area type="monotone" dataKey="fx" stroke="var(--color-fx)" fill="url(#fillFx)" strokeWidth={2} />
                  <Area type="monotone" dataKey="mm" stroke="var(--color-mm)" fill="url(#fillMm)" strokeWidth={2} />
                </AreaChart>
              </ChartContainer>
            </CardContent>
          </Card>

          <Card className="md:col-span-3">
            <CardHeader>
              <CardTitle>{t("dashboard.chart.distributionTitle")}</CardTitle>
              <CardDescription>{t("dashboard.chart.distributionDesc")}</CardDescription>
            </CardHeader>
            <CardContent>
              <ChartContainer config={pieChartConfig} className="mx-auto h-[300px] w-full">
                <PieChart>
                  <ChartTooltip content={<ChartTooltipContent nameKey="name" />} />
                  <Pie data={pieData} dataKey="value" nameKey="name" cx="50%" cy="50%" innerRadius={60} outerRadius={100} paddingAngle={2}>
                    {pieData.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={entry.fill} />
                    ))}
                  </Pie>
                  <ChartLegend content={<ChartLegendContent nameKey="name" />} />
                </PieChart>
              </ChartContainer>
            </CardContent>
          </Card>
        </div>

        {/* Charts Row 2 */}
        <div className="grid gap-4 md:grid-cols-7">
          <Card className="md:col-span-4">
            <CardHeader>
              <CardTitle>{t("dashboard.chart.statusTitle")}</CardTitle>
              <CardDescription>{t("dashboard.chart.statusDesc")}</CardDescription>
            </CardHeader>
            <CardContent>
              <ChartContainer config={statusChartConfig} className="h-[300px] w-full">
                <BarChart data={statusData} margin={{ top: 10, right: 10, left: 0, bottom: 0 }}>
                  <CartesianGrid vertical={false} className="stroke-muted" />
                  <XAxis dataKey="date" tickLine={false} axisLine={false} tickMargin={8} className="text-xs fill-muted-foreground" />
                  <YAxis tickLine={false} axisLine={false} tickMargin={8} className="text-xs fill-muted-foreground" />
                  <ChartTooltip content={<ChartTooltipContent />} />
                  <ChartLegend content={<ChartLegendContent />} />
                  <Bar dataKey="open" stackId="a" fill="var(--color-open)" radius={[0, 0, 0, 0]} />
                  <Bar dataKey="pending" stackId="a" fill="var(--color-pending)" radius={[0, 0, 0, 0]} />
                  <Bar dataKey="completed" stackId="a" fill="var(--color-completed)" radius={[0, 0, 0, 0]} />
                  <Bar dataKey="cancelled" stackId="a" fill="var(--color-cancelled)" radius={[4, 4, 0, 0]} />
                </BarChart>
              </ChartContainer>
            </CardContent>
          </Card>

          <Card className="md:col-span-3">
            <CardHeader>
              <CardTitle>{t("dashboard.recentTransactions")}</CardTitle>
              <CardDescription>{t("dashboard.chart.recentTitle")}</CardDescription>
            </CardHeader>
            <CardContent>
              {/* Desktop table */}
              <div className="hidden md:block">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t("table.id")}</TableHead>
                    <TableHead>{t("table.type")}</TableHead>
                    <TableHead className="text-right">{t("table.amount")}</TableHead>
                    <TableHead>{t("table.status")}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {recent_transactions.map((tx) => (
                    <TableRow key={tx.id} className="cursor-pointer hover:bg-muted/50" onClick={() => window.location.href = getTxDetailUrl(tx)}>
                      <TableCell className="font-mono text-xs">{tx.ticket}</TableCell>
                      <TableCell className="text-xs">{getDealTypeLabel(tx.deal_type, t)}</TableCell>
                      <TableCell className="text-right text-xs font-medium">
                        {formatAmount(tx.amount, tx.currency, t)}
                      </TableCell>
                      <TableCell>
                        <Badge variant="secondary" className={statusColors[tx.status] || ""}>
                          {getStatusLabel(tx.status, t)}
                        </Badge>
                      </TableCell>
                    </TableRow>
                  ))}
                  {recent_transactions.length === 0 && (
                    <TableRow>
                      <TableCell colSpan={4} className="text-center text-sm text-muted-foreground py-8">
                        {t("dashboard.empty.noTransactions")}
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
              </div>

              {/* Mobile card view */}
              <div className="flex flex-col gap-2 md:hidden">
                {recent_transactions.map((tx) => (
                  <Link key={tx.id} href={getTxDetailUrl(tx)} className="flex items-center justify-between rounded-lg border p-3 transition-colors hover:bg-accent">
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <span className="font-mono text-xs font-medium">{tx.ticket}</span>
                        <Badge variant="outline" className="text-[10px] px-1.5">{tx.module}</Badge>
                      </div>
                      <p className="text-xs text-muted-foreground mt-0.5 truncate">{getDealTypeLabel(tx.deal_type, t)}</p>
                    </div>
                    <div className="text-right ml-3 shrink-0">
                      <div className="text-sm font-semibold tabular-nums">{formatAmount(tx.amount, tx.currency, t)}</div>
                      <Badge variant="secondary" className={`text-[10px] ${statusColors[tx.status] || ""}`}>
                        {getStatusLabel(tx.status, t)}
                      </Badge>
                    </div>
                  </Link>
                ))}
                {recent_transactions.length === 0 && (
                  <p className="text-center text-sm text-muted-foreground py-8">{t("dashboard.empty.noTransactions")}</p>
                )}
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </>
  );
}
