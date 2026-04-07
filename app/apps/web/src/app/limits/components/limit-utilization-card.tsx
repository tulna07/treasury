"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Separator } from "@/components/ui/separator";
import { IconAlertCircle, IconInfinity } from "@tabler/icons-react";
import { useTranslation } from "@/lib/i18n";
import { useLimitUtilization } from "@/hooks/use-limits";
import type { LimitUtilization } from "@/hooks/use-limits";

function formatVND(value: number): string {
  return Number(value).toLocaleString("en-US", {
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  });
}

function UtilizationRow({
  label,
  value,
  bold,
  negative,
}: {
  label: string;
  value: string;
  bold?: boolean;
  negative?: boolean;
}) {
  return (
    <div className="flex items-center justify-between py-1.5">
      <span className="text-sm text-muted-foreground">{label}</span>
      <span
        className={`tabular-nums ${bold ? "font-semibold" : "font-medium"} ${negative ? "text-red-600 dark:text-red-400" : ""}`}
      >
        {value}
      </span>
    </div>
  );
}

function UtilizationSection({
  utilization,
  t,
}: {
  utilization: LimitUtilization;
  t: (key: string) => string;
}) {
  const remaining = utilization.remaining;
  const isNegative = remaining !== null && remaining < 0;

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-base">
          {utilization.limit_type === "COLLATERALIZED"
            ? t("limit.collateralized")
            : t("limit.uncollateralized")}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-1">
        <UtilizationRow
          label={t("limit.mmPrincipal")}
          value={formatVND(utilization.mm_principal)}
        />
        <UtilizationRow
          label={t("limit.bondSettlement")}
          value={formatVND(utilization.bond_settlement)}
        />
        <UtilizationRow
          label={t("limit.fxAmount")}
          value={formatVND(utilization.fx_amount)}
        />
        <Separator />
        <UtilizationRow
          label={t("limit.totalUtilized")}
          value={formatVND(utilization.total_utilized)}
          bold
        />
        <UtilizationRow
          label={t("limit.granted")}
          value={
            utilization.granted_unlimited
              ? t("limit.unlimited")
              : formatVND(utilization.granted_limit ?? 0)
          }
          bold
        />
        <Separator />
        <UtilizationRow
          label={t("limit.remaining")}
          value={
            utilization.granted_unlimited
              ? t("limit.unlimited")
              : formatVND(remaining ?? 0)
          }
          bold
          negative={isNegative}
        />
        {utilization.fx_rate_used > 0 && (
          <div className="pt-2">
            <Badge variant="outline" className="text-xs">
              {t("limit.fxRateUsed")}: {Number(utilization.fx_rate_used).toLocaleString("en-US")}
            </Badge>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

interface LimitUtilizationCardProps {
  counterpartyId: string;
}

export function LimitUtilizationCard({
  counterpartyId,
}: LimitUtilizationCardProps) {
  const { t } = useTranslation();
  const { data, isLoading, isError, error } =
    useLimitUtilization(counterpartyId);

  if (isLoading) {
    return (
      <div className="grid gap-4 md:grid-cols-2">
        <Skeleton className="h-64" />
        <Skeleton className="h-64" />
      </div>
    );
  }

  if (isError) {
    return (
      <Alert variant="destructive">
        <IconAlertCircle className="size-4" />
        <AlertDescription>
          {(error as { error?: string })?.error || t("limit.loadError")}
        </AlertDescription>
      </Alert>
    );
  }

  const utilizations = data ?? [];

  if (utilizations.length === 0) {
    return (
      <div className="py-12 text-center text-muted-foreground">
        {t("limit.noUtilization")}
      </div>
    );
  }

  return (
    <div className="grid gap-4 md:grid-cols-2">
      {utilizations.map((u) => (
        <UtilizationSection
          key={u.limit_type}
          utilization={u}
          t={t}
        />
      ))}
    </div>
  );
}
