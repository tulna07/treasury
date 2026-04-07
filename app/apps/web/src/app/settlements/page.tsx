"use client";

import { useState } from "react";
import { Header } from "@/components/header";
import { AccessDenied } from "@/components/access-denied";
import { useTranslation } from "@/lib/i18n";
import { useAbility } from "@/hooks/use-ability";
import { useSettlements, type SettlementFilters, type InternationalPayment } from "@/hooks/use-settlements";
import { useApproveSettlement, useRejectSettlement } from "@/hooks/use-settlements";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table";
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from "@/components/ui/dialog";
import { Textarea } from "@/components/ui/textarea";
import { IconAlertCircle, IconCheck, IconX, IconInbox } from "@tabler/icons-react";
import { extractErrorMessage, formatDate } from "@/lib/utils";
import { toast } from "sonner";

const statusStyles: Record<string, string> = {
  PENDING: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300",
  APPROVED: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300",
  REJECTED: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300",
};

const statusLabels: Record<string, string> = {
  PENDING: "Chờ duyệt",
  APPROVED: "Đã duyệt",
  REJECTED: "Từ chối",
};

export default function SettlementsPage() {
  const { t } = useTranslation();
  const ability = useAbility();
  const [filters, setFilters] = useState<SettlementFilters>({ page: 1, page_size: 20 });
  const { data, isLoading, isError, error } = useSettlements(filters);
  const approveMutation = useApproveSettlement();
  const rejectMutation = useRejectSettlement();
  const [rejectDialog, setRejectDialog] = useState<{ open: boolean; id: string }>({ open: false, id: "" });
  const [rejectReason, setRejectReason] = useState("");

  const payments: InternationalPayment[] = data?.data ?? [];

  // Permission guard
  if (!ability.can("view", "Settlement") && !ability.can("approve", "Settlement")) {
    return (
      <>
        <Header />
        <AccessDenied />
      </>
    );
  }

  function handleApprove(id: string) {
    approveMutation.mutate(id, {
      onSuccess: () => toast.success(t("settlement.approved")),
      onError: (e) => toast.error(extractErrorMessage(e)),
    });
  }

  function handleReject() {
    if (!rejectReason.trim()) return;
    rejectMutation.mutate(
      { id: rejectDialog.id, reason: rejectReason },
      {
        onSuccess: () => {
          toast.success(t("settlement.rejected"));
          setRejectDialog({ open: false, id: "" });
          setRejectReason("");
        },
        onError: (e) => toast.error(extractErrorMessage(e)),
      }
    );
  }

  return (
    <>
      <Header />
      <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
        <div className="flex flex-col gap-4 pt-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight">{t("settlement.title")}</h1>
            <p className="text-muted-foreground">{t("settlement.description")}</p>
          </div>
          <div className="flex gap-2">
            <Select
              value={filters.status ?? "ALL"}
              onValueChange={(v) => setFilters({ ...filters, status: v === "ALL" ? undefined : v ?? undefined })}
            >
              <SelectTrigger className="w-40"><SelectValue>{(v: string) => v === "ALL" ? t("common.all") : (statusLabels[v] ?? v)}</SelectValue></SelectTrigger>
              <SelectContent>
                <SelectItem value="ALL" label={t("common.all")}>{t("common.all")}</SelectItem>
                <SelectItem value="PENDING" label={statusLabels.PENDING}>{statusLabels.PENDING}</SelectItem>
                <SelectItem value="APPROVED" label={statusLabels.APPROVED}>{statusLabels.APPROVED}</SelectItem>
                <SelectItem value="REJECTED" label={statusLabels.REJECTED}>{statusLabels.REJECTED}</SelectItem>
              </SelectContent>
            </Select>
            <Select
              value={filters.source_module ?? "ALL"}
              onValueChange={(v) => setFilters({ ...filters, source_module: v === "ALL" ? undefined : v ?? undefined })}
            >
              <SelectTrigger className="w-32"><SelectValue>{(v: string) => v === "ALL" ? t("common.all") : v}</SelectValue></SelectTrigger>
              <SelectContent>
                <SelectItem value="ALL" label={t("common.all")}>{t("common.all")}</SelectItem>
                <SelectItem value="FX" label="FX">FX</SelectItem>
                <SelectItem value="MM" label="MM">MM</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        {isLoading && <div className="space-y-2">{Array.from({ length: 5 }).map((_, i) => <Skeleton key={i} className="h-12 w-full" />)}</div>}
        {isError && <Alert variant="destructive"><IconAlertCircle className="size-4" /><AlertDescription>{extractErrorMessage(error)}</AlertDescription></Alert>}

        {!isLoading && !isError && (
          <Card>
            <CardContent className="p-0">
              <div className="rounded-lg border overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{t("settlement.ticket")}</TableHead>
                      <TableHead>{t("settlement.module")}</TableHead>
                      <TableHead>{t("settlement.counterparty")}</TableHead>
                      <TableHead>{t("settlement.currency")}</TableHead>
                      <TableHead className="text-right">{t("settlement.amount")}</TableHead>
                      <TableHead>{t("settlement.transferDate")}</TableHead>
                      <TableHead>{t("settlement.status")}</TableHead>
                      <TableHead>{t("common.actions")}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {payments.length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={8} className="h-32 text-center text-muted-foreground">
                          <IconInbox className="mx-auto size-8 mb-2" />
                          {t("settlement.empty")}
                        </TableCell>
                      </TableRow>
                    ) : (
                      payments.map((p) => (
                        <TableRow key={p.id}>
                          <TableCell className="font-mono font-medium">{p.ticket_display}</TableCell>
                          <TableCell><Badge variant="outline">{p.source_module}</Badge></TableCell>
                          <TableCell>{p.counterparty_name}</TableCell>
                          <TableCell>{p.currency_code}</TableCell>
                          <TableCell className="text-right tabular-nums">{Number(p.amount).toLocaleString("en-US", { minimumFractionDigits: 2 })}</TableCell>
                          <TableCell>{formatDate(p.transfer_date)}</TableCell>
                          <TableCell><Badge className={statusStyles[p.settlement_status]}>{statusLabels[p.settlement_status]}</Badge></TableCell>
                          <TableCell>
                            {p.settlement_status === "PENDING" && (
                              <div className="flex gap-1">
                                <Button size="sm" variant="ghost" className="text-green-600" onClick={() => handleApprove(p.id)}>
                                  <IconCheck className="size-4" />
                                </Button>
                                <Button size="sm" variant="ghost" className="text-red-600" onClick={() => setRejectDialog({ open: true, id: p.id })}>
                                  <IconX className="size-4" />
                                </Button>
                              </div>
                            )}
                          </TableCell>
                        </TableRow>
                      ))
                    )}
                  </TableBody>
                </Table>
              </div>
            </CardContent>
          </Card>
        )}
      </div>

      <Dialog open={rejectDialog.open} onOpenChange={(o) => !o && setRejectDialog({ open: false, id: "" })}>
        <DialogContent>
          <DialogHeader><DialogTitle>{t("settlement.rejectTitle")}</DialogTitle></DialogHeader>
          <Textarea
            placeholder={t("settlement.rejectReason")}
            value={rejectReason}
            onChange={(e) => setRejectReason(e.target.value)}
            rows={3}
          />
          <DialogFooter>
            <Button variant="outline" onClick={() => setRejectDialog({ open: false, id: "" })}>{t("common.cancel")}</Button>
            <Button variant="destructive" onClick={handleReject} disabled={!rejectReason.trim()}>{t("settlement.confirmReject")}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
