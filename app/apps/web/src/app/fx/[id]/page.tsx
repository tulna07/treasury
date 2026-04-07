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
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
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
  useFxDeal,
  useApproveFxDeal,
  useRecallFxDeal,
  useCancelFxDeal,
  useCancelApproveFxDeal,
  useCloneFxDeal,
  useDeleteFxDeal,
  useApprovalHistory,
} from "@/hooks/use-fx";
import type { ApprovalHistoryEntry } from "@/hooks/use-fx";
import { FxStatusBadge } from "../components/fx-status-badge";
import { FxActionButtons } from "../components/fx-action-buttons";
import { FxAttachmentUpload } from "../components/fx-attachment-upload";
import { FxAttachmentList } from "../components/fx-attachment-list";
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
        {t("fx.history.noHistory")}
      </p>
    );
  }

  return (
    <div className="space-y-0">
      {entries.map((entry, idx) => (
        <div key={entry.id} className="flex gap-3">
          {/* Timeline line */}
          <div className="flex flex-col items-center">
            <div className="flex size-8 items-center justify-center rounded-full border bg-background">
              <HistoryIcon actionType={entry.action_type} />
            </div>
            {idx < entries.length - 1 && (
              <div className="w-px flex-1 bg-border" />
            )}
          </div>

          {/* Content */}
          <div className="pb-6 pt-0.5 flex-1 min-w-0">
            <p className="text-sm font-medium">
              {t(`fx.history.action.${entry.action_type}`) !== `fx.history.action.${entry.action_type}`
                ? t(`fx.history.action.${entry.action_type}`)
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

export default function FxDetailPage() {
  const { t } = useTranslation();
  const router = useRouter();
  const params = useParams();
  const id = params.id as string;

  const { data, isLoading, isError, error } = useFxDeal(id);
  const approveMutation = useApproveFxDeal(id);
  const recallMutation = useRecallFxDeal(id);
  const cancelMutation = useCancelFxDeal(id);
  const cancelApproveMutation = useCancelApproveFxDeal(id);
  const cloneMutation = useCloneFxDeal();
  const deleteMutation = useDeleteFxDeal();
  const { data: historyData } = useApprovalHistory(id);
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
            <Button variant="ghost" onClick={() => router.push("/fx")}>
              <IconArrowLeft className="mr-2 size-4" />
              {t("fx.backToList")}
            </Button>
          </div>
          <Alert variant="destructive">
            <IconAlertCircle className="size-4" />
            <AlertDescription>
              {t("fx.loadError")}: {extractErrorMessage(error)}
            </AlertDescription>
          </Alert>
        </div>
      </>
    );
  }

  function formatAmount(value: string) {
    return Number(value).toLocaleString("en-US", {
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    });
  }

  return (
    <>
      <Header />
      <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
        <div className="flex flex-col gap-4 pt-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex items-center gap-4">
            <Button variant="ghost" size="icon" onClick={() => router.push("/fx")}>
              <IconArrowLeft className="size-4" />
            </Button>
            <div>
              <div className="flex items-center gap-3">
                <h1 className="text-2xl font-bold tracking-tight">
                  {deal.ticket_number || id.slice(0, 8)}
                </h1>
                <FxStatusBadge status={deal.status} />
              </div>
              <p className="text-muted-foreground">{t("fx.detail")}</p>
            </div>
          </div>

          <FxActionButtons
            deal={deal}
            onApprove={(reason) =>
              approveMutation.mutate(
                { action: "APPROVE", reason },
                {
                  onSuccess: () => toast.success(t("fx.approveSuccess")),
                  onError: (err) =>
                    toast.error(extractErrorMessage(err)),
                }
              )
            }
            onReject={(reason) =>
              approveMutation.mutate(
                { action: "REJECT", reason },
                {
                  onSuccess: () => toast.success(t("fx.rejectSuccess")),
                  onError: (err) =>
                    toast.error(extractErrorMessage(err)),
                }
              )
            }
            onRecall={(reason) =>
              recallMutation.mutate(
                { reason },
                {
                  onSuccess: () => toast.success(t("fx.recallSuccess")),
                  onError: (err) =>
                    toast.error(extractErrorMessage(err)),
                }
              )
            }
            onCancel={(reason) =>
              cancelMutation.mutate(
                { reason },
                {
                  onSuccess: () => toast.success(t("fx.cancelSuccess")),
                  onError: (err) =>
                    toast.error(extractErrorMessage(err)),
                }
              )
            }
            onCancelApprove={() =>
              cancelApproveMutation.mutate(
                { action: "APPROVE" },
                {
                  onSuccess: () => toast.success(t("fx.cancelApproveSuccess")),
                  onError: (err) =>
                    toast.error(extractErrorMessage(err)),
                }
              )
            }
            onCancelReject={(reason) =>
              cancelApproveMutation.mutate(
                { action: "REJECT", reason },
                {
                  onSuccess: () => toast.success(t("fx.cancelRejectSuccess")),
                  onError: (err) =>
                    toast.error(extractErrorMessage(err)),
                }
              )
            }
            onClone={() =>
              cloneMutation.mutate(deal.id, {
                onSuccess: (res) => {
                  toast.success(t("fx.cloneSuccess"));
                  router.push(`/fx/${res.id}`);
                },
                onError: (err) =>
                  toast.error(extractErrorMessage(err)),
              })
            }
            onEdit={() => router.push(`/fx/${deal.id}/edit`)}
            onDelete={() =>
              deleteMutation.mutate(deal.id, {
                onSuccess: () => {
                  toast.success(t("fx.deleteSuccess"));
                  router.push("/fx");
                },
                onError: (err) =>
                  toast.error(extractErrorMessage(err)),
              })
            }
            isLoading={isActionLoading}
          />
        </div>

        <Tabs defaultValue="info" className="w-full">
          <TabsList>
            <TabsTrigger value="info">{t("fx.history.dealInfo")}</TabsTrigger>
            <TabsTrigger value="attachments" className="gap-1.5">
              <IconPaperclip className="size-3.5" />
              {t("attachment.title")}
            </TabsTrigger>
            <TabsTrigger value="history">{t("fx.history.title")}</TabsTrigger>
          </TabsList>

          <TabsContent value="info" className="space-y-4 mt-4">
            <div className="grid gap-4 md:grid-cols-2">
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">{t("fx.detail")}</CardTitle>
                </CardHeader>
                <CardContent>
                  <dl className="grid grid-cols-2 gap-x-4 gap-y-3 text-sm">
                    <div>
                      <dt className="text-muted-foreground">{t("fx.ticketNumber")}</dt>
                      <dd className="font-mono font-medium">{deal.ticket_number || "—"}</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">{t("table.status")}</dt>
                      <dd><FxStatusBadge status={deal.status} /></dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">{t("fx.dealType")}</dt>
                      <dd className="font-medium">{t(`fx.type.${deal.deal_type}`)}</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">{t("fx.direction")}</dt>
                      <dd className="font-medium">{t(`fx.direction.${deal.direction}`)}</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">{t("fx.counterparty")}</dt>
                      <dd className="font-medium">
                        {deal.counterparty_code} — {deal.counterparty_name}
                      </dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">{t("fx.currency")}</dt>
                      <dd className="font-mono font-medium">{deal.currency_code}</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">{t("fx.notionalAmount")}</dt>
                      <dd className="font-medium tabular-nums">{formatAmount(deal.notional_amount)}</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">{t("fx.tradeDate")}</dt>
                      <dd className="font-medium">{formatDate(deal.trade_date)}</dd>
                    </div>
                    {deal.execution_date && (
                      <div>
                        <dt className="text-muted-foreground">{t("fx.executionDate")}</dt>
                        <dd className="font-medium">{formatDate(deal.execution_date)}</dd>
                      </div>
                    )}
                    {deal.pay_code_klb && (
                      <div>
                        <dt className="text-muted-foreground">{t("fx.payCodeKlb")}</dt>
                        <dd className="font-mono text-sm">{deal.pay_code_klb}</dd>
                      </div>
                    )}
                    {deal.pay_code_counterparty && (
                      <div>
                        <dt className="text-muted-foreground">{t("fx.payCodeCounterparty")}</dt>
                        <dd className="font-mono text-sm">{deal.pay_code_counterparty}</dd>
                      </div>
                    )}
                    <div>
                      <dt className="text-muted-foreground">Intl Settlement</dt>
                      <dd>
                        <Badge variant={deal.is_international ? "default" : "secondary"}>
                          {deal.is_international ? t("common.yes") : t("common.no")}
                        </Badge>
                      </dd>
                    </div>
                    {deal.settlement_amount && (
                      <div>
                        <dt className="text-muted-foreground">{t("fx.settlementAmount")}</dt>
                        <dd className="font-medium tabular-nums">
                          {formatAmount(deal.settlement_amount)}{" "}
                          <span className="font-mono">{deal.settlement_currency}</span>
                        </dd>
                      </div>
                    )}
                  </dl>
                  {deal.note && (
                    <>
                      <Separator className="my-4" />
                      <div className="text-sm">
                        <span className="text-muted-foreground">{t("fx.note")}:</span>{" "}
                        <span>{deal.note}</span>
                      </div>
                    </>
                  )}
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle className="text-base">{t("fx.createdAt")}</CardTitle>
                </CardHeader>
                <CardContent>
                  <dl className="grid grid-cols-2 gap-x-4 gap-y-3 text-sm">
                    <div>
                      <dt className="text-muted-foreground">{t("fx.createdAt")}</dt>
                      <dd className="font-medium">{formatDate(deal.created_at)}</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">{t("fx.updatedAt")}</dt>
                      <dd className="font-medium">{formatDate(deal.updated_at)}</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">{t("fx.version")}</dt>
                      <dd className="font-medium">{deal.version}</dd>
                    </div>
                  </dl>
                </CardContent>
              </Card>
            </div>

            {deal.legs && deal.legs.length > 0 && (
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">{t("fx.legs")}</CardTitle>
                </CardHeader>
                <CardContent>
                  {/* Desktop table */}
                  <div className="hidden sm:block rounded-lg border overflow-x-auto">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>{t("fx.legNumber")}</TableHead>
                          <TableHead>{t("fx.valueDate")}</TableHead>
                          <TableHead className="text-right">{t("fx.exchangeRate")}</TableHead>
                          <TableHead>{t("fx.buyCurrency")}</TableHead>
                          <TableHead className="text-right">{t("fx.buyAmount")}</TableHead>
                          <TableHead>{t("fx.sellCurrency")}</TableHead>
                          <TableHead className="text-right">{t("fx.sellAmount")}</TableHead>
                          {deal.deal_type === "SWAP" && (
                            <>
                              <TableHead>{t("fx.payCodeKlb")}</TableHead>
                              <TableHead>{t("fx.payCodeCounterparty")}</TableHead>
                            </>
                          )}
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {deal.legs.map((leg) => (
                          <TableRow key={leg.leg_number}>
                            <TableCell>{leg.leg_number}</TableCell>
                            <TableCell>{formatDate(leg.value_date)}</TableCell>
                            <TableCell className="text-right font-medium tabular-nums">
                              {Number(leg.exchange_rate).toLocaleString("en-US", { minimumFractionDigits: 2 })}
                            </TableCell>
                            <TableCell className="font-mono">{leg.buy_currency}</TableCell>
                            <TableCell className="text-right font-medium tabular-nums">
                              {formatAmount(leg.buy_amount)}
                            </TableCell>
                            <TableCell className="font-mono">{leg.sell_currency}</TableCell>
                            <TableCell className="text-right font-medium tabular-nums">
                              {formatAmount(leg.sell_amount)}
                            </TableCell>
                            {deal.deal_type === "SWAP" && (
                              <>
                                <TableCell className="font-mono text-sm">{leg.pay_code_klb || "—"}</TableCell>
                                <TableCell className="font-mono text-sm">{leg.pay_code_counterparty || "—"}</TableCell>
                              </>
                            )}
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </div>

                  {/* Mobile cards */}
                  <div className="sm:hidden space-y-3">
                    {deal.legs.map((leg) => (
                      <div key={leg.leg_number} className="rounded-lg border p-3 space-y-2 text-sm">
                        <div className="font-medium">{t("fx.legNumber")} {leg.leg_number}</div>
                        <div className="grid grid-cols-2 gap-x-4 gap-y-1">
                          <div>
                            <span className="text-muted-foreground">{t("fx.valueDate")}:</span>{" "}
                            {formatDate(leg.value_date)}
                          </div>
                          <div>
                            <span className="text-muted-foreground">{t("fx.exchangeRate")}:</span>{" "}
                            <span className="tabular-nums">{Number(leg.exchange_rate).toLocaleString("en-US", { minimumFractionDigits: 2 })}</span>
                          </div>
                          <div>
                            <span className="text-muted-foreground">{t("fx.buyCurrency")}:</span>{" "}
                            <span className="font-mono">{leg.buy_currency}</span>{" "}
                            <span className="tabular-nums">{formatAmount(leg.buy_amount)}</span>
                          </div>
                          <div>
                            <span className="text-muted-foreground">{t("fx.sellCurrency")}:</span>{" "}
                            <span className="font-mono">{leg.sell_currency}</span>{" "}
                            <span className="tabular-nums">{formatAmount(leg.sell_amount)}</span>
                          </div>
                          {deal.deal_type === "SWAP" && leg.pay_code_klb && (
                            <div className="col-span-2">
                              <span className="text-muted-foreground">{t("fx.payCodeKlb")}:</span>{" "}
                              <span className="font-mono text-sm">{leg.pay_code_klb}</span>
                            </div>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>
            )}
          </TabsContent>

          <TabsContent value="attachments" className="mt-4 space-y-4">
            <Card>
              <CardHeader>
                <CardTitle className="text-base">{t("attachment.title")}</CardTitle>
              </CardHeader>
              <CardContent>
                <FxAttachmentList
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
                  <FxAttachmentUpload dealId={id} />
                </CardContent>
              </Card>
            )}
          </TabsContent>

          <TabsContent value="history" className="mt-4">
            <Card>
              <CardHeader>
                <CardTitle className="text-base">{t("fx.history.title")}</CardTitle>
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
