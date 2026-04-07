"use client";

import { useParams, useRouter } from "next/navigation";
import { Header } from "@/components/header";
import { useTranslation } from "@/lib/i18n";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Separator } from "@/components/ui/separator";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  IconArrowLeft,
  IconAlertCircle,
  IconCheck,
  IconX,
  IconArrowBack,
  IconBan,
  IconClock,
  IconPaperclip,
} from "@tabler/icons-react";
import { toast } from "sonner";
import { formatDate, extractErrorMessage } from "@/lib/utils";
import {
  useBondDeal,
  useApproveBondDeal,
  useRecallBondDeal,
  useCancelBondDeal,
  useCancelApproveBondDeal,
  useCloneBondDeal,
  useDeleteBondDeal,
  useBondHistory,
} from "@/hooks/use-bonds";
import type { ApprovalHistoryEntry } from "@/hooks/use-bonds";
import { BondStatusBadge } from "../components/bond-status-badge";
import { BondActionButtons } from "../components/bond-action-buttons";
import { BondAttachmentUpload } from "../components/bond-attachment-upload";
import { BondAttachmentList } from "../components/bond-attachment-list";
import { useAuthStore } from "@/lib/auth-store";

function HistoryIcon({ actionType }: { actionType: string }) {
  if (actionType.includes("REJECT") || actionType.includes("VOID")) {
    return <IconX className="size-4 text-red-500" />;
  }
  if (actionType.includes("RECALL")) {
    return <IconArrowBack className="size-4 text-amber-500" />;
  }
  if (actionType.includes("CANCEL_REQUEST")) {
    return <IconBan className="size-4 text-rose-500" />;
  }
  if (actionType.includes("APPROVE")) {
    return <IconCheck className="size-4 text-green-500" />;
  }
  return <IconClock className="size-4 text-blue-500" />;
}

function ApprovalTimeline({ entries }: { entries: ApprovalHistoryEntry[] }) {
  const { t } = useTranslation();

  if (!entries || entries.length === 0) {
    return (
      <p className="text-sm text-muted-foreground py-4 text-center">
        {t("bond.history.noHistory")}
      </p>
    );
  }

  return (
    <div className="space-y-0">
      {entries.map((entry, idx) => (
        <div key={entry.id} className="flex gap-3">
          <div className="flex flex-col items-center">
            <div className="flex size-8 items-center justify-center rounded-full border bg-background">
              <HistoryIcon actionType={entry.action_type} />
            </div>
            {idx < entries.length - 1 && (
              <div className="w-px flex-1 bg-border" />
            )}
          </div>
          <div className="pb-6 pt-0.5 flex-1 min-w-0">
            <p className="text-sm font-medium">
              {t(`bond.history.action.${entry.action_type}`) !== `bond.history.action.${entry.action_type}`
                ? t(`bond.history.action.${entry.action_type}`)
                : entry.action_type.replace(/_/g, " ")}
            </p>
            <p className="text-xs text-muted-foreground mt-0.5">
              {entry.performer_name} · {formatDate(entry.performed_at)}
            </p>
            <div className="text-xs text-muted-foreground mt-0.5">
              {entry.status_before} → {entry.status_after}
            </div>
            {entry.reason && (
              <p className="text-sm mt-1 text-muted-foreground italic">
                &ldquo;{entry.reason}&rdquo;
              </p>
            )}
          </div>
        </div>
      ))}
    </div>
  );
}

function formatAmount(value: string | number) {
  return Number(value).toLocaleString("en-US", {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });
}

export default function BondDetailPage() {
  const { t } = useTranslation();
  const router = useRouter();
  const params = useParams();
  const id = params.id as string;

  const { data, isLoading, isError, error } = useBondDeal(id);
  const approveMutation = useApproveBondDeal(id);
  const recallMutation = useRecallBondDeal(id);
  const cancelMutation = useCancelBondDeal(id);
  const cancelApproveMutation = useCancelApproveBondDeal(id);
  const cloneMutation = useCloneBondDeal();
  const deleteMutation = useDeleteBondDeal();
  const { data: historyData } = useBondHistory(id);
  const user = useAuthStore((s) => s.user);

  const deal = data;
  const isActionLoading =
    approveMutation.isPending ||
    recallMutation.isPending ||
    cancelMutation.isPending ||
    cancelApproveMutation.isPending ||
    cloneMutation.isPending ||
    deleteMutation.isPending;

  if (isLoading) {
    return (
      <>
        <Header />
        <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
          <div className="pt-4 space-y-4">
            <Skeleton className="h-8 w-64" />
            <Skeleton className="h-48 w-full" />
            <Skeleton className="h-32 w-full" />
          </div>
        </div>
      </>
    );
  }

  if (isError || !deal) {
    return (
      <>
        <Header />
        <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
          <div className="pt-4">
            <Button variant="ghost" onClick={() => router.push("/bonds")}>
              <IconArrowLeft className="mr-2 size-4" />
              {t("bond.backToList")}
            </Button>
          </div>
          <Alert variant="destructive">
            <IconAlertCircle className="size-4" />
            <AlertDescription>
              {t("bond.loadError")}: {extractErrorMessage(error)}
            </AlertDescription>
          </Alert>
        </div>
      </>
    );
  }

  return (
    <>
      <Header />
      <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
        <div className="flex flex-col gap-4 pt-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex items-center gap-4">
            <Button variant="ghost" size="icon" onClick={() => router.push("/bonds")}>
              <IconArrowLeft className="size-4" />
            </Button>
            <div>
              <div className="flex items-center gap-3">
                <h1 className="text-2xl font-bold tracking-tight">
                  {deal.deal_number || id.slice(0, 8)}
                </h1>
                <BondStatusBadge status={deal.status} />
              </div>
              <p className="text-muted-foreground">{t("bond.detail")}</p>
            </div>
          </div>

          <BondActionButtons
            deal={deal}
            onApprove={(reason) =>
              approveMutation.mutate(
                { action: "APPROVE", reason },
                {
                  onSuccess: () => toast.success(t("bond.approveSuccess")),
                  onError: (err) => toast.error(extractErrorMessage(err)),
                }
              )
            }
            onReject={(reason) =>
              approveMutation.mutate(
                { action: "REJECT", reason },
                {
                  onSuccess: () => toast.success(t("bond.rejectSuccess")),
                  onError: (err) => toast.error(extractErrorMessage(err)),
                }
              )
            }
            onRecall={(reason) =>
              recallMutation.mutate(
                { reason },
                {
                  onSuccess: () => toast.success(t("bond.recallSuccess")),
                  onError: (err) => toast.error(extractErrorMessage(err)),
                }
              )
            }
            onCancel={(reason) =>
              cancelMutation.mutate(
                { reason },
                {
                  onSuccess: () => toast.success(t("bond.cancelSuccess")),
                  onError: (err) => toast.error(extractErrorMessage(err)),
                }
              )
            }
            onCancelApprove={() =>
              cancelApproveMutation.mutate(
                { action: "APPROVE" },
                {
                  onSuccess: () => toast.success(t("bond.cancelApproveSuccess")),
                  onError: (err) => toast.error(extractErrorMessage(err)),
                }
              )
            }
            onCancelReject={(reason) =>
              cancelApproveMutation.mutate(
                { action: "REJECT", reason },
                {
                  onSuccess: () => toast.success(t("bond.cancelRejectSuccess")),
                  onError: (err) => toast.error(extractErrorMessage(err)),
                }
              )
            }
            onClone={() =>
              cloneMutation.mutate(deal.id, {
                onSuccess: (res) => {
                  toast.success(t("bond.cloneSuccess"));
                  router.push(`/bonds/${res.id}`);
                },
                onError: (err) => toast.error(extractErrorMessage(err)),
              })
            }
            onEdit={() => router.push(`/bonds/${deal.id}/edit`)}
            onDelete={() =>
              deleteMutation.mutate(deal.id, {
                onSuccess: () => {
                  toast.success(t("bond.deleteSuccess"));
                  router.push("/bonds");
                },
                onError: (err) => toast.error(extractErrorMessage(err)),
              })
            }
            isLoading={isActionLoading}
          />
        </div>

        <Tabs defaultValue="info" className="w-full">
          <TabsList>
            <TabsTrigger value="info">{t("bond.detail")}</TabsTrigger>
            <TabsTrigger value="attachments" className="gap-1.5">
              <IconPaperclip className="size-3.5" />
              {t("attachment.title")}
            </TabsTrigger>
            <TabsTrigger value="history">{t("bond.history.title")}</TabsTrigger>
          </TabsList>

          <TabsContent value="info" className="space-y-4 mt-4">
            <div className="grid gap-4 md:grid-cols-2">
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">{t("bond.basicInfo")}</CardTitle>
                </CardHeader>
                <CardContent>
                  <dl className="grid grid-cols-2 gap-x-4 gap-y-3 text-sm">
                    <div>
                      <dt className="text-muted-foreground">{t("bond.dealNumber")}</dt>
                      <dd className="font-mono font-medium">{deal.deal_number || "—"}</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">{t("table.status")}</dt>
                      <dd><BondStatusBadge status={deal.status} /></dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">{t("bond.category")}</dt>
                      <dd className="font-medium">{t(`bond.category.${deal.bond_category}`)}</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">{t("bond.direction")}</dt>
                      <dd className="font-medium">{t(`bond.direction.${deal.direction}`)}</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">{t("bond.counterparty")}</dt>
                      <dd className="font-medium">
                        {deal.counterparty_code} — {deal.counterparty_name}
                      </dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">{t("bond.transactionType")}</dt>
                      <dd className="font-medium">
                        {deal.transaction_type === "OTHER"
                          ? deal.transaction_type_other || t("bond.txType.OTHER")
                          : t(`bond.txType.${deal.transaction_type}`)}
                      </dd>
                    </div>
                    {deal.portfolio_type && (
                      <div>
                        <dt className="text-muted-foreground">{t("bond.portfolioType")}</dt>
                        <dd className="font-medium">{deal.portfolio_type}</dd>
                      </div>
                    )}
                  </dl>
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle className="text-base">{t("bond.bondInfo")}</CardTitle>
                </CardHeader>
                <CardContent>
                  <dl className="grid grid-cols-2 gap-x-4 gap-y-3 text-sm">
                    <div>
                      <dt className="text-muted-foreground">{t("bond.bondCode")}</dt>
                      <dd className="font-mono font-medium">{deal.bond_code_display || "—"}</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">{t("bond.issuer")}</dt>
                      <dd className="font-medium">{deal.issuer}</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">{t("bond.couponRate")}</dt>
                      <dd className="font-medium tabular-nums">{deal.coupon_rate}%</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">{t("bond.faceValue")}</dt>
                      <dd className="font-medium tabular-nums">{formatAmount(deal.face_value)}</dd>
                    </div>
                    {deal.issue_date && (
                      <div>
                        <dt className="text-muted-foreground">{t("bond.issueDate")}</dt>
                        <dd className="font-medium">{formatDate(deal.issue_date)}</dd>
                      </div>
                    )}
                    <div>
                      <dt className="text-muted-foreground">{t("bond.maturityDate")}</dt>
                      <dd className="font-medium">{formatDate(deal.maturity_date)}</dd>
                    </div>
                  </dl>
                </CardContent>
              </Card>
            </div>

            <div className="grid gap-4 md:grid-cols-2">
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">{t("bond.pricingInfo")}</CardTitle>
                </CardHeader>
                <CardContent>
                  <dl className="grid grid-cols-2 gap-x-4 gap-y-3 text-sm">
                    <div>
                      <dt className="text-muted-foreground">{t("bond.quantity")}</dt>
                      <dd className="font-medium tabular-nums">{deal.quantity.toLocaleString("en-US")}</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">{t("bond.cleanPrice")}</dt>
                      <dd className="font-medium tabular-nums">{formatAmount(deal.clean_price)}</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">{t("bond.discountRate")}</dt>
                      <dd className="font-medium tabular-nums">{deal.discount_rate}%</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">{t("bond.settlementPrice")}</dt>
                      <dd className="font-medium tabular-nums">{formatAmount(deal.settlement_price)}</dd>
                    </div>
                    <div className="col-span-2">
                      <dt className="text-muted-foreground">{t("bond.totalValue")}</dt>
                      <dd className="text-lg font-bold tabular-nums">{formatAmount(deal.total_value)}</dd>
                    </div>
                  </dl>
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle className="text-base">{t("bond.settlementInfo")}</CardTitle>
                </CardHeader>
                <CardContent>
                  <dl className="grid grid-cols-2 gap-x-4 gap-y-3 text-sm">
                    <div>
                      <dt className="text-muted-foreground">{t("bond.tradeDate")}</dt>
                      <dd className="font-medium">{formatDate(deal.trade_date)}</dd>
                    </div>
                    {deal.order_date && (
                      <div>
                        <dt className="text-muted-foreground">{t("bond.orderDate")}</dt>
                        <dd className="font-medium">{formatDate(deal.order_date)}</dd>
                      </div>
                    )}
                    <div>
                      <dt className="text-muted-foreground">{t("bond.valueDate")}</dt>
                      <dd className="font-medium">{formatDate(deal.value_date)}</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">{t("bond.paymentDate")}</dt>
                      <dd className="font-medium">{formatDate(deal.payment_date)}</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">{t("bond.remainingTenor")}</dt>
                      <dd className="font-medium">{deal.remaining_tenor_days} {t("bond.days")}</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">{t("bond.confirmationMethod")}</dt>
                      <dd className="font-medium">
                        {deal.confirmation_method === "OTHER"
                          ? deal.confirmation_other || t("bond.confirmation.OTHER")
                          : t(`bond.confirmation.${deal.confirmation_method}`)}
                      </dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">{t("bond.contractPreparedBy")}</dt>
                      <dd className="font-medium">{t(`bond.contractBy.${deal.contract_prepared_by}`)}</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">{t("bond.createdBy")}</dt>
                      <dd className="font-medium">{deal.created_by_name}</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">{t("bond.createdAt")}</dt>
                      <dd className="font-medium">{formatDate(deal.created_at)}</dd>
                    </div>
                  </dl>
                  {deal.note && (
                    <>
                      <Separator className="my-4" />
                      <div className="text-sm">
                        <span className="text-muted-foreground">{t("bond.note")}:</span>{" "}
                        <span>{deal.note}</span>
                      </div>
                    </>
                  )}
                </CardContent>
              </Card>
            </div>
          </TabsContent>

          <TabsContent value="attachments" className="mt-4 space-y-4">
            <Card>
              <CardHeader>
                <CardTitle className="text-base">{t("attachment.title")}</CardTitle>
              </CardHeader>
              <CardContent>
                <BondAttachmentList
                  dealId={id}
                  canDelete={deal.status === "OPEN"}
                  currentUserId={user?.id}
                />
              </CardContent>
            </Card>
            {deal.status === "OPEN" && (
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">{t("attachment.upload")}</CardTitle>
                </CardHeader>
                <CardContent>
                  <BondAttachmentUpload dealId={id} />
                </CardContent>
              </Card>
            )}
          </TabsContent>

          <TabsContent value="history" className="mt-4">
            <Card>
              <CardHeader>
                <CardTitle className="text-base">{t("bond.history.title")}</CardTitle>
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
