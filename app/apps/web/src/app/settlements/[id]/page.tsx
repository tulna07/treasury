"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { Header } from "@/components/header";
import { useTranslation } from "@/lib/i18n";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Textarea } from "@/components/ui/textarea";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  IconArrowLeft,
  IconAlertCircle,
  IconCheck,
  IconX,
  IconCreditCard,
  IconBuildingBank,
  IconShieldCheck,
} from "@tabler/icons-react";
import { toast } from "sonner";
import { formatDate, extractErrorMessage } from "@/lib/utils";
import {
  useSettlement,
  useApproveSettlement,
  useRejectSettlement,
} from "@/hooks/use-settlements";

function formatAmount(value: string | number) {
  return Number(value).toLocaleString("en-US", {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });
}

function InfoRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="font-medium">{children}</dd>
    </div>
  );
}

function StatusBadge({ status }: { status: string }) {
  const variant = status === "APPROVED"
    ? "default"
    : status === "REJECTED"
      ? "destructive"
      : "secondary";
  return <Badge variant={variant}>{status}</Badge>;
}

export default function SettlementDetailPage() {
  const { t } = useTranslation();
  const router = useRouter();
  const params = useParams();
  const id = params.id as string;

  const { data: payment, isLoading, isError, error } = useSettlement(id);
  const approveMutation = useApproveSettlement();
  const rejectMutation = useRejectSettlement();

  const [rejectOpen, setRejectOpen] = useState(false);
  const [rejectReason, setRejectReason] = useState("");

  const isActionLoading = approveMutation.isPending || rejectMutation.isPending;
  const canAct = payment?.settlement_status === "PENDING";

  // ─── Loading State ──────────────────────────────────────────
  if (isLoading) {
    return (
      <>
        <Header />
        <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
          <div className="pt-4 space-y-4">
            <Skeleton className="h-8 w-64" />
            <div className="grid gap-4 md:grid-cols-2">
              <Skeleton className="h-64 w-full" />
              <Skeleton className="h-64 w-full" />
            </div>
            <Skeleton className="h-48 w-full" />
          </div>
        </div>
      </>
    );
  }

  // ─── Error State ────────────────────────────────────────────
  if (isError || !payment) {
    return (
      <>
        <Header />
        <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
          <div className="pt-4">
            <Button variant="ghost" onClick={() => router.push("/settlements")}>
              <IconArrowLeft className="mr-2 size-4" />
              {t("settlement.detail.backToList")}
            </Button>
          </div>
          <Alert variant="destructive">
            <IconAlertCircle className="size-4" />
            <AlertDescription>
              {t("settlement.detail.loadError")}: {extractErrorMessage(error)}
            </AlertDescription>
          </Alert>
        </div>
      </>
    );
  }

  // ─── Action Handlers ───────────────────────────────────────
  function handleApprove() {
    approveMutation.mutate(id, {
      onSuccess: () => toast.success(t("settlement.approved")),
      onError: (err) => toast.error(extractErrorMessage(err)),
    });
  }

  function handleRejectConfirm() {
    if (!rejectReason.trim()) return;
    rejectMutation.mutate(
      { id, reason: rejectReason.trim() },
      {
        onSuccess: () => {
          toast.success(t("settlement.rejected"));
          setRejectOpen(false);
          setRejectReason("");
        },
        onError: (err) => toast.error(extractErrorMessage(err)),
      }
    );
  }

  return (
    <>
      <Header />
      <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
        {/* ─── Page Header ──────────────────────────────────── */}
        <div className="flex flex-col gap-4 pt-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex items-center gap-4">
            <Button variant="ghost" size="icon" onClick={() => router.push("/settlements")}>
              <IconArrowLeft className="size-4" />
            </Button>
            <div>
              <div className="flex items-center gap-3">
                <h1 className="text-2xl font-bold tracking-tight">
                  {payment.ticket_display}
                </h1>
                <StatusBadge status={payment.settlement_status} />
              </div>
              <p className="text-muted-foreground">{t("settlement.detail.title")}</p>
            </div>
          </div>

          {canAct && (
            <div className="flex items-center gap-2">
              <Button
                variant="destructive"
                size="sm"
                disabled={isActionLoading}
                onClick={() => setRejectOpen(true)}
              >
                <IconX className="mr-1.5 size-4" />
                {t("settlement.detail.reject")}
              </Button>
              <Button
                size="sm"
                disabled={isActionLoading}
                onClick={handleApprove}
              >
                <IconCheck className="mr-1.5 size-4" />
                {t("settlement.detail.approve")}
              </Button>
            </div>
          )}
        </div>

        {/* ─── Cards Grid ───────────────────────────────────── */}
        <div className="grid gap-4 md:grid-cols-2">
          {/* Payment Info */}
          <Card>
            <CardHeader className="flex flex-row items-center gap-2 space-y-0 pb-3">
              <IconCreditCard className="size-5 text-muted-foreground" />
              <CardTitle className="text-base">{t("settlement.detail.paymentInfo")}</CardTitle>
            </CardHeader>
            <CardContent>
              <dl className="grid grid-cols-2 gap-x-4 gap-y-3 text-sm">
                <InfoRow label={t("settlement.detail.ticket")}>
                  <span className="font-mono">{payment.ticket_display}</span>
                </InfoRow>
                <InfoRow label={t("settlement.detail.sourceModule")}>
                  <Badge variant="outline">{payment.source_module}</Badge>
                </InfoRow>
                <InfoRow label={t("settlement.detail.sourceDealId")}>
                  <span className="font-mono">{payment.source_deal_id}</span>
                </InfoRow>
                <InfoRow label={t("settlement.detail.counterparty")}>
                  {payment.counterparty_code} — {payment.counterparty_name}
                </InfoRow>
                <InfoRow label={t("settlement.detail.currency")}>
                  <span className="font-mono">{payment.currency_code}</span>
                </InfoRow>
                <InfoRow label={t("settlement.detail.amount")}>
                  <span className="tabular-nums font-semibold">
                    {formatAmount(payment.amount)}
                  </span>
                </InfoRow>
              </dl>
            </CardContent>
          </Card>

          {/* Settlement Details */}
          <Card>
            <CardHeader className="flex flex-row items-center gap-2 space-y-0 pb-3">
              <IconBuildingBank className="size-5 text-muted-foreground" />
              <CardTitle className="text-base">{t("settlement.detail.settlementDetails")}</CardTitle>
            </CardHeader>
            <CardContent>
              <dl className="grid grid-cols-2 gap-x-4 gap-y-3 text-sm">
                <InfoRow label={t("settlement.detail.debitAccount")}>
                  <span className="font-mono">{payment.debit_account}</span>
                </InfoRow>
                <InfoRow label={t("settlement.detail.bicCode")}>
                  <span className="font-mono">{payment.bic_code || "—"}</span>
                </InfoRow>
                <InfoRow label={t("settlement.detail.transferDate")}>
                  {formatDate(payment.transfer_date)}
                </InfoRow>
                <InfoRow label={t("settlement.detail.originalTradeDate")}>
                  {formatDate(payment.original_trade_date)}
                </InfoRow>
                <InfoRow label={t("settlement.detail.approvedByDivision")}>
                  {payment.approved_by_division || "—"}
                </InfoRow>
                <div className="col-span-2">
                  <dt className="text-muted-foreground">{t("settlement.detail.counterpartySsi")}</dt>
                  <dd className="mt-1 whitespace-pre-wrap rounded-md bg-muted/50 p-2 font-mono text-xs">
                    {payment.counterparty_ssi || "—"}
                  </dd>
                </div>
              </dl>
            </CardContent>
          </Card>

          {/* Status & Audit */}
          <Card className="md:col-span-2">
            <CardHeader className="flex flex-row items-center gap-2 space-y-0 pb-3">
              <IconShieldCheck className="size-5 text-muted-foreground" />
              <CardTitle className="text-base">{t("settlement.detail.statusAudit")}</CardTitle>
            </CardHeader>
            <CardContent>
              <dl className="grid grid-cols-2 gap-x-4 gap-y-3 text-sm md:grid-cols-4">
                <InfoRow label={t("settlement.detail.status")}>
                  <StatusBadge status={payment.settlement_status} />
                </InfoRow>
                <InfoRow label={t("settlement.detail.settledBy")}>
                  {payment.settled_by_name || "—"}
                </InfoRow>
                <InfoRow label={t("settlement.detail.settledAt")}>
                  {payment.settled_at ? formatDate(payment.settled_at) : "—"}
                </InfoRow>
                <InfoRow label={t("settlement.detail.createdAt")}>
                  {formatDate(payment.created_at)}
                </InfoRow>
                {payment.settlement_status === "REJECTED" && payment.rejection_reason && (
                  <div className="col-span-2 md:col-span-4">
                    <dt className="text-muted-foreground">{t("settlement.detail.rejectionReason")}</dt>
                    <dd className="mt-1 rounded-md bg-destructive/10 p-2 text-sm text-destructive">
                      {payment.rejection_reason}
                    </dd>
                  </div>
                )}
              </dl>
            </CardContent>
          </Card>
        </div>
      </div>

      {/* ─── Reject Dialog ──────────────────────────────────── */}
      <Dialog open={rejectOpen} onOpenChange={setRejectOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("settlement.rejectTitle")}</DialogTitle>
            <DialogDescription>{payment.ticket_display}</DialogDescription>
          </DialogHeader>
          <Textarea
            placeholder={t("settlement.rejectReason")}
            value={rejectReason}
            onChange={(e) => setRejectReason(e.target.value)}
            rows={3}
          />
          <DialogFooter>
            <Button variant="ghost" onClick={() => setRejectOpen(false)}>
              {t("common.cancel")}
            </Button>
            <Button
              variant="destructive"
              disabled={!rejectReason.trim() || rejectMutation.isPending}
              onClick={handleRejectConfirm}
            >
              {t("settlement.confirmReject")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
