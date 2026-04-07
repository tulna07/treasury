"use client";

import { useParams, useRouter } from "next/navigation";
import { useCallback, useRef } from "react";
import { Header } from "@/components/header";
import { useTranslation } from "@/lib/i18n";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Separator } from "@/components/ui/separator";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  IconArrowLeft,
  IconAlertCircle,
  IconCheck,
  IconX,
  IconBuildingBank,
  IconCash,
  IconCalendar,
  IconInfoCircle,
  IconPaperclip,
  IconHistory,
  IconArrowBack,
  IconBan,
  IconClock,
  IconUpload,
  IconTrash,
  IconDownload,
  IconCertificate,
} from "@tabler/icons-react";
import { toast } from "sonner";
import { formatDate, extractErrorMessage } from "@/lib/utils";
import { useMMRepoDeal, useApproveMMRepo, useMMRepoHistory } from "@/hooks/use-mm";
import type { MMApprovalHistoryEntry } from "@/hooks/use-mm";
import { useAttachments, useUploadAttachments, useDeleteAttachment, downloadAttachment } from "@/hooks/use-attachments";
import { MMStatusBadge } from "../../components/mm-status-badge";

// ─── Helpers ──────────────────────────────────────────────────

const fmtAmt = (v: string | number) => Number(v).toLocaleString("en-US", { minimumFractionDigits: 2, maximumFractionDigits: 2 });
const fmtRate = (v: string | number) => Number(v).toLocaleString("en-US", { minimumFractionDigits: 4, maximumFractionDigits: 6 });
const fmtPct = (v: string | number) => Number(v).toLocaleString("en-US", { minimumFractionDigits: 2, maximumFractionDigits: 4 });
const fmtSize = (b: number) => b < 1024 ? `${b} B` : b < 1048576 ? `${(b / 1024).toFixed(1)} KB` : `${(b / 1048576).toFixed(1)} MB`;

function InfoRow({ label, children }: { label: string; children: React.ReactNode }) {
  return <div><dt className="text-muted-foreground">{label}</dt><dd className="font-medium">{children}</dd></div>;
}

// ─── History Icon ─────────────────────────────────────────────

function HistoryIcon({ actionType }: { actionType: string }) {
  if (actionType.includes("REJECT") || actionType.includes("VOID"))
    return <IconX className="size-4 text-red-500" />;
  if (actionType.includes("RECALL"))
    return <IconArrowBack className="size-4 text-amber-500" />;
  if (actionType.includes("CANCEL"))
    return <IconBan className="size-4 text-rose-500" />;
  if (actionType.includes("APPROVE"))
    return <IconCheck className="size-4 text-green-500" />;
  return <IconClock className="size-4 text-blue-500" />;
}

// ─── Approval Timeline ───────────────────────────────────────

function ApprovalTimeline({ entries }: { entries: MMApprovalHistoryEntry[] }) {
  const { t } = useTranslation();
  if (!entries || entries.length === 0) {
    return <p className="text-sm text-muted-foreground py-4 text-center">{t("mm.detail.history.empty")}</p>;
  }
  return (
    <div className="space-y-0">
      {entries.map((entry, idx) => (
        <div key={entry.id} className="flex gap-3">
          <div className="flex flex-col items-center">
            <div className="flex size-8 items-center justify-center rounded-full border bg-background">
              <HistoryIcon actionType={entry.action_type} />
            </div>
            {idx < entries.length - 1 && <div className="w-px flex-1 bg-border" />}
          </div>
          <div className="pb-6 pt-0.5 flex-1 min-w-0">
            <p className="text-sm font-medium">{entry.action_type.replace(/_/g, " ")}</p>
            <p className="text-xs text-muted-foreground mt-0.5">
              {entry.performer_name} · {formatDate(entry.performed_at)}
            </p>
            <div className="text-xs text-muted-foreground mt-0.5">
              {entry.from_status} → {entry.to_status}
            </div>
            {entry.reason && (
              <p className="text-sm mt-1 text-muted-foreground italic">&ldquo;{entry.reason}&rdquo;</p>
            )}
          </div>
        </div>
      ))}
    </div>
  );
}

// ─── Attachments Panel ────────────────────────────────────────

function AttachmentsPanel({ dealId, canUpload }: { dealId: string; canUpload: boolean }) {
  const { t } = useTranslation();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const { data: attachments = [], isLoading } = useAttachments("MM_GOVT_REPO", dealId);
  const uploadMutation = useUploadAttachments();
  const deleteMutation = useDeleteAttachment();

  const handleUpload = useCallback((files: FileList | null) => {
    if (!files || files.length === 0) return;
    uploadMutation.mutate(
      { dealModule: "MM_GOVT_REPO", dealId, files: Array.from(files) },
      {
        onSuccess: () => toast.success(t("mm.detail.attachments.upload")),
        onError: (err) => toast.error(extractErrorMessage(err)),
      },
    );
  }, [dealId, uploadMutation, t]);

  const handleDelete = useCallback((id: string) => {
    if (!confirm(t("mm.detail.attachments.deleteConfirm"))) return;
    deleteMutation.mutate(
      { id, dealModule: "MM_GOVT_REPO", dealId },
      { onError: (err) => toast.error(extractErrorMessage(err)) },
    );
  }, [dealId, deleteMutation, t]);

  if (isLoading) return <Skeleton className="h-32 w-full" />;

  return (
    <div className="space-y-4">
      {canUpload && (
        <div
          className="border-2 border-dashed rounded-lg p-6 text-center cursor-pointer hover:border-primary/50 transition-colors"
          onClick={() => fileInputRef.current?.click()}
          onDragOver={(e) => { e.preventDefault(); e.stopPropagation(); }}
          onDrop={(e) => { e.preventDefault(); e.stopPropagation(); handleUpload(e.dataTransfer.files); }}
        >
          <IconUpload className="size-8 mx-auto text-muted-foreground mb-2" />
          <p className="text-sm text-muted-foreground">{t("mm.detail.attachments.dragDrop")}</p>
          <input ref={fileInputRef} type="file" multiple className="hidden" onChange={(e) => handleUpload(e.target.files)} />
          {uploadMutation.isPending && <p className="text-xs text-muted-foreground mt-2">Uploading...</p>}
        </div>
      )}

      {attachments.length === 0 ? (
        <p className="text-sm text-muted-foreground py-4 text-center">{t("mm.detail.attachments.empty")}</p>
      ) : (
        <div className="space-y-2">
          {attachments.map((att) => (
            <div key={att.id} className="flex items-center justify-between rounded-lg border p-3 text-sm">
              <div className="flex items-center gap-2 min-w-0 flex-1">
                <IconPaperclip className="size-4 text-muted-foreground shrink-0" />
                <div className="min-w-0">
                  <p className="font-medium truncate">{att.file_name}</p>
                  <p className="text-xs text-muted-foreground">{fmtSize(att.file_size)} · {formatDate(att.created_at)}</p>
                </div>
              </div>
              <div className="flex items-center gap-1 shrink-0">
                <Button variant="ghost" size="icon" className="size-8" onClick={() => downloadAttachment(att.id)}>
                  <IconDownload className="size-4" />
                </Button>
                {canUpload && (
                  <Button variant="ghost" size="icon" className="size-8 text-destructive" onClick={() => handleDelete(att.id)} disabled={deleteMutation.isPending}>
                    <IconTrash className="size-4" />
                  </Button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

// ─── Main Page ────────────────────────────────────────────────

export default function MMRepoDetailPage() {
  const { t } = useTranslation();
  const router = useRouter();
  const params = useParams();
  const id = params.id as string;

  const { data: deal, isLoading, isError, error } = useMMRepoDeal(id);
  const approveMutation = useApproveMMRepo(id);
  const { data: historyData } = useMMRepoHistory(id);

  const isActionLoading = approveMutation.isPending;
  const canApprove = deal?.status === "PENDING_L2_APPROVAL" ||
    deal?.status === "PENDING_TP_REVIEW" ||
    deal?.status === "PENDING_RISK_APPROVAL" ||
    deal?.status === "PENDING_CHIEF_ACCOUNTANT";

  if (isLoading) return (
    <><Header /><div className="flex flex-1 flex-col gap-4 p-4 pt-0"><div className="pt-4 space-y-4">
      <Skeleton className="h-8 w-64" /><div className="grid gap-4 md:grid-cols-2"><Skeleton className="h-64 w-full" /><Skeleton className="h-64 w-full" /></div>
    </div></div></>
  );

  if (isError || !deal) return (
    <><Header /><div className="flex flex-1 flex-col gap-4 p-4 pt-0">
      <div className="pt-4"><Button variant="ghost" onClick={() => router.push("/mm")}><IconArrowLeft className="mr-2 size-4" />{t("mm.detail.backToList")}</Button></div>
      <Alert variant="destructive"><IconAlertCircle className="size-4" /><AlertDescription>{t("mm.detail.loadError")}: {extractErrorMessage(error)}</AlertDescription></Alert>
    </div></>
  );

  function handleApprove() {
    approveMutation.mutate(
      { action: "APPROVE" },
      { onSuccess: () => toast.success(t("mm.detail.approveSuccess")), onError: (err) => toast.error(extractErrorMessage(err)) },
    );
  }

  function handleReject() {
    approveMutation.mutate(
      { action: "REJECT", reason: "Rejected" },
      { onSuccess: () => toast.success(t("mm.detail.rejectSuccess")), onError: (err) => toast.error(extractErrorMessage(err)) },
    );
  }

  return (
    <>
      <Header />
      <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
        {/* Page Header */}
        <div className="flex flex-col gap-4 pt-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex items-center gap-4">
            <Button variant="ghost" size="icon" onClick={() => router.push("/mm")}>
              <IconArrowLeft className="size-4" />
            </Button>
            <div>
              <div className="flex items-center gap-3">
                <h1 className="text-2xl font-bold tracking-tight">{deal.deal_number || id.slice(0, 8)}</h1>
                <MMStatusBadge status={deal.status} />
                <Badge variant="outline">{deal.deal_subtype}</Badge>
              </div>
              <p className="text-muted-foreground">{t("mm.repo.detail.title")}</p>
            </div>
          </div>
          {canApprove && (
            <div className="flex items-center gap-2">
              <Button variant="destructive" size="sm" disabled={isActionLoading} onClick={handleReject}>
                <IconX className="mr-1.5 size-4" />
                {t("mm.detail.reject")}
              </Button>
              <Button size="sm" disabled={isActionLoading} onClick={handleApprove}>
                <IconCheck className="mr-1.5 size-4" />
                {t("mm.detail.approve")}
              </Button>
            </div>
          )}
        </div>

        {/* Tabs */}
        <Tabs defaultValue="info" className="w-full">
          <TabsList>
            <TabsTrigger value="info" className="gap-1.5">
              <IconInfoCircle className="size-3.5" />
              {t("mm.detail.tabs.info")}
            </TabsTrigger>
            <TabsTrigger value="attachments" className="gap-1.5">
              <IconPaperclip className="size-3.5" />
              {t("mm.detail.tabs.attachments")}
            </TabsTrigger>
            <TabsTrigger value="history" className="gap-1.5">
              <IconHistory className="size-3.5" />
              {t("mm.detail.tabs.history")}
            </TabsTrigger>
          </TabsList>

          {/* Tab 1: Deal Info */}
          <TabsContent value="info" className="space-y-4 mt-4">
            <div className="grid gap-4 md:grid-cols-2">
              {/* Card 1: Deal Info */}
              <Card>
                <CardHeader className="flex flex-row items-center gap-2 space-y-0 pb-3">
                  <IconBuildingBank className="size-5 text-muted-foreground" />
                  <CardTitle className="text-base">{t("mm.repo.detail.dealInfo")}</CardTitle>
                </CardHeader>
                <CardContent>
                  <dl className="grid grid-cols-2 gap-x-4 gap-y-3 text-sm">
                    <InfoRow label={t("mm.repo.detail.dealNumber")}><span className="font-mono">{deal.deal_number}</span></InfoRow>
                    <InfoRow label={t("mm.repo.detail.dealSubtype")}><Badge variant="outline">{deal.deal_subtype}</Badge></InfoRow>
                    <InfoRow label={t("mm.repo.detail.sessionName")}>{deal.session_name}</InfoRow>
                    <InfoRow label={t("mm.repo.detail.counterparty")}>{deal.counterparty_code} — {deal.counterparty_name}</InfoRow>
                    <InfoRow label={t("mm.repo.detail.status")}><MMStatusBadge status={deal.status} /></InfoRow>
                  </dl>
                  {deal.note && (
                    <>
                      <Separator className="my-4" />
                      <div className="text-sm">
                        <span className="text-muted-foreground">{t("mm.repo.detail.note")}:</span> <span>{deal.note}</span>
                      </div>
                    </>
                  )}
                </CardContent>
              </Card>

              {/* Card 2: Bond & Financial */}
              <Card>
                <CardHeader className="flex flex-row items-center gap-2 space-y-0 pb-3">
                  <IconCertificate className="size-5 text-muted-foreground" />
                  <CardTitle className="text-base">{t("mm.repo.detail.bondFinancial")}</CardTitle>
                </CardHeader>
                <CardContent>
                  <dl className="grid grid-cols-2 gap-x-4 gap-y-3 text-sm">
                    <InfoRow label={t("mm.repo.detail.bondCode")}><span className="font-mono">{deal.bond_code}</span></InfoRow>
                    <InfoRow label={t("mm.repo.detail.bondIssuer")}>{deal.bond_issuer}</InfoRow>
                    <InfoRow label={t("mm.repo.detail.couponRate")}><span className="tabular-nums">{fmtRate(deal.bond_coupon_rate)}%</span></InfoRow>
                    {deal.bond_maturity_date && (
                      <InfoRow label={t("mm.repo.detail.bondMaturityDate")}>{formatDate(deal.bond_maturity_date)}</InfoRow>
                    )}
                    <InfoRow label={t("mm.repo.detail.notionalAmount")}><span className="tabular-nums">{fmtAmt(deal.notional_amount)}</span></InfoRow>
                    <InfoRow label={t("mm.repo.detail.winningRate")}><span className="tabular-nums">{fmtRate(deal.winning_rate)}%</span></InfoRow>
                    <InfoRow label={t("mm.repo.detail.haircutPct")}><span className="tabular-nums">{fmtPct(deal.haircut_pct)}%</span></InfoRow>
                  </dl>
                </CardContent>
              </Card>

              {/* Card 3: Settlement Dates */}
              <Card>
                <CardHeader className="flex flex-row items-center gap-2 space-y-0 pb-3">
                  <IconCalendar className="size-5 text-muted-foreground" />
                  <CardTitle className="text-base">{t("mm.repo.detail.settlementDates")}</CardTitle>
                </CardHeader>
                <CardContent>
                  <dl className="grid grid-cols-2 gap-x-4 gap-y-3 text-sm">
                    <InfoRow label={t("mm.repo.detail.tradeDate")}>{formatDate(deal.trade_date)}</InfoRow>
                    <InfoRow label={t("mm.repo.detail.tenorDays")}>{deal.tenor_days} {t("mm.detail.days")}</InfoRow>
                    <InfoRow label={t("mm.repo.detail.settlementDate1")}>{formatDate(deal.settlement_date_1)}</InfoRow>
                    <InfoRow label={t("mm.repo.detail.settlementDate2")}>{formatDate(deal.settlement_date_2)}</InfoRow>
                  </dl>
                </CardContent>
              </Card>

              {/* Card 4: Metadata */}
              <Card>
                <CardHeader className="flex flex-row items-center gap-2 space-y-0 pb-3">
                  <IconCash className="size-5 text-muted-foreground" />
                  <CardTitle className="text-base">{t("mm.repo.detail.metadata")}</CardTitle>
                </CardHeader>
                <CardContent>
                  <dl className="grid grid-cols-2 gap-x-4 gap-y-3 text-sm">
                    <InfoRow label={t("mm.repo.detail.createdBy")}>{deal.created_by_name}</InfoRow>
                    <InfoRow label={t("mm.repo.detail.createdAt")}>{formatDate(deal.created_at)}</InfoRow>
                  </dl>
                </CardContent>
              </Card>
            </div>
          </TabsContent>

          {/* Tab 2: Attachments */}
          <TabsContent value="attachments" className="mt-4">
            <Card>
              <CardHeader>
                <CardTitle className="text-base">{t("mm.detail.tabs.attachments")}</CardTitle>
              </CardHeader>
              <CardContent>
                <AttachmentsPanel dealId={id} canUpload={deal.status === "OPEN"} />
              </CardContent>
            </Card>
          </TabsContent>

          {/* Tab 3: Approval History */}
          <TabsContent value="history" className="mt-4">
            <Card>
              <CardHeader>
                <CardTitle className="text-base">{t("mm.detail.tabs.history")}</CardTitle>
              </CardHeader>
              <CardContent>
                <ApprovalTimeline entries={historyData || []} />
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </div>
    </>
  );
}
