"use client";

import { useState } from "react";
import { Header } from "@/components/header";
import { useTranslation } from "@/lib/i18n";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { DatePicker } from "@/components/ui/date-picker";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { PaginationBar } from "@/components/pagination";
import { IconAlertCircle, IconInbox, IconChevronDown } from "@tabler/icons-react";
import { extractErrorMessage } from "@/lib/utils";
import { useIsMobile } from "@/hooks/use-mobile";
import {
  useAuditLogs,
  useAuditStats,
  type AuditLog,
  type AuditLogFilters,
} from "@/hooks/use-audit-logs";

function AuditStatsCards() {
  const { t } = useTranslation();
  const { data: stats, isLoading } = useAuditStats();

  if (isLoading) {
    return (
      <div className="grid gap-4 grid-cols-2 sm:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="h-20 rounded-lg" />
        ))}
      </div>
    );
  }

  if (!stats?.length) return null;

  const total = stats.reduce((sum, s) => sum + s.total, 0);

  return (
    <div className="grid gap-4 grid-cols-2 sm:grid-cols-4">
      <Card>
        <CardContent className="pt-4 pb-3">
          <p className="text-sm text-muted-foreground">{t("audit.totalActions")}</p>
          <p className="text-2xl font-bold">{total}</p>
        </CardContent>
      </Card>
      {stats.slice(0, 3).map((stat) => (
        <Card key={stat.action}>
          <CardContent className="pt-4 pb-3">
            <p className="text-sm text-muted-foreground">{stat.action}</p>
            <p className="text-2xl font-bold">{stat.total}</p>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}

function AuditLogExpandable({ log }: { log: AuditLog }) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const hasChanges = log.old_values || log.new_values;

  return (
    <Collapsible open={open} onOpenChange={setOpen}>
      <CollapsibleTrigger className="w-full">
        <div className="flex items-start gap-3 p-3 rounded-lg border cursor-pointer hover:bg-accent/50 transition-colors text-left">
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 flex-wrap">
              <Badge variant="outline" className="text-xs shrink-0">
                {log.action}
              </Badge>
              <span className="text-sm font-medium">{log.user_full_name}</span>
              {log.deal_module && (
                <Badge variant="secondary" className="text-xs">
                  {log.deal_module}
                </Badge>
              )}
            </div>
            <div className="flex flex-wrap gap-x-4 gap-y-1 mt-1 text-xs text-muted-foreground">
              {log.deal_id && <span>{t("audit.dealId")}: {log.deal_id.slice(0, 8)}...</span>}
              {log.status_before && log.status_after && (
                <span>{log.status_before} → {log.status_after}</span>
              )}
              {log.reason && <span>{t("audit.reason")}: {log.reason}</span>}
              <span>{log.ip_address}</span>
              <span>{new Date(log.performed_at).toLocaleString("vi-VN")}</span>
            </div>
          </div>
          {hasChanges && (
            <IconChevronDown className={`size-4 text-muted-foreground shrink-0 mt-1 transition-transform ${open ? "rotate-180" : ""}`} />
          )}
        </div>
      </CollapsibleTrigger>
      {hasChanges && (
        <CollapsibleContent>
          <div className="mx-3 mb-3 mt-1 rounded-lg bg-muted/50 p-3 text-xs">
            <p className="font-medium mb-2">{t("audit.changes")}:</p>
            <div className="grid gap-2 sm:grid-cols-2">
              {log.old_values && (
                <div>
                  <p className="text-muted-foreground mb-1">{t("audit.oldValues")}:</p>
                  <pre className="rounded bg-background p-2 overflow-x-auto whitespace-pre-wrap break-all">
                    {JSON.stringify(log.old_values, null, 2)}
                  </pre>
                </div>
              )}
              {log.new_values && (
                <div>
                  <p className="text-muted-foreground mb-1">{t("audit.newValues")}:</p>
                  <pre className="rounded bg-background p-2 overflow-x-auto whitespace-pre-wrap break-all">
                    {JSON.stringify(log.new_values, null, 2)}
                  </pre>
                </div>
              )}
            </div>
          </div>
        </CollapsibleContent>
      )}
    </Collapsible>
  );
}

export default function AuditLogsPage() {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const [filters, setFilters] = useState<AuditLogFilters>({
    page: 1,
    page_size: 20,
  });

  const { data, isLoading, isError, error, refetch } = useAuditLogs(filters);
  const logs = data?.data ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.ceil(total / (filters.page_size || 20));

  return (
    <>
      <Header />
      <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
        <div className="pt-4">
          <h1 className="text-2xl font-bold tracking-tight">{t("audit.title")}</h1>
          <p className="text-muted-foreground">{t("audit.description")}</p>
        </div>

        <AuditStatsCards />

        {/* Filters */}
        <div className="flex flex-col gap-2 sm:flex-row">
          <Select
            value={filters.action || "all"}
            onValueChange={(v) => {
              const val = !v || v === "all" ? undefined : v;
              setFilters((f) => ({ ...f, action: val, page: 1 }));
            }}
          >
            <SelectTrigger className="w-full sm:w-auto">
              <SelectValue placeholder={t("audit.filterByAction")}>{(v: string) => v === "all" ? t("audit.allActions") : v}</SelectValue>
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all" label={t("audit.allActions")}>{t("audit.allActions")}</SelectItem>
              <SelectItem value="CREATE" label="CREATE">CREATE</SelectItem>
              <SelectItem value="UPDATE" label="UPDATE">UPDATE</SelectItem>
              <SelectItem value="APPROVE" label="APPROVE">APPROVE</SelectItem>
              <SelectItem value="REJECT" label="REJECT">REJECT</SelectItem>
              <SelectItem value="RECALL" label="RECALL">RECALL</SelectItem>
              <SelectItem value="CANCEL" label="CANCEL">CANCEL</SelectItem>
              <SelectItem value="DELETE" label="DELETE">DELETE</SelectItem>
              <SelectItem value="LOGIN" label="LOGIN">LOGIN</SelectItem>
              <SelectItem value="LOGOUT" label="LOGOUT">LOGOUT</SelectItem>
            </SelectContent>
          </Select>
          <Select
            value={filters.deal_module || "all"}
            onValueChange={(v) => {
              const val = !v || v === "all" ? undefined : v;
              setFilters((f) => ({ ...f, deal_module: val, page: 1 }));
            }}
          >
            <SelectTrigger className="w-full sm:w-auto">
              <SelectValue placeholder={t("audit.filterByModule")}>{(v: string) => v === "all" ? t("audit.allModules") : v}</SelectValue>
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all" label={t("audit.allModules")}>{t("audit.allModules")}</SelectItem>
              <SelectItem value="FX" label="FX">FX</SelectItem>
              <SelectItem value="GTCG" label="GTCG">GTCG</SelectItem>
              <SelectItem value="MM" label="MM">MM</SelectItem>
              <SelectItem value="AUTH" label="AUTH">AUTH</SelectItem>
              <SelectItem value="ADMIN" label="ADMIN">ADMIN</SelectItem>
            </SelectContent>
          </Select>
          <DatePicker
            value={filters.date_from || ""}
            onChange={(val) =>
              setFilters((f) => ({ ...f, date_from: val || undefined, page: 1 }))
            }
            className="w-full sm:w-auto"
            placeholder={t("audit.dateFrom")}
          />
          <DatePicker
            value={filters.date_to || ""}
            onChange={(val) =>
              setFilters((f) => ({ ...f, date_to: val || undefined, page: 1 }))
            }
            className="w-full sm:w-auto"
            placeholder={t("audit.dateTo")}
          />
        </div>

        {/* Content */}
        {isError ? (
          <Alert variant="destructive">
            <IconAlertCircle className="size-4" />
            <AlertDescription className="flex items-center justify-between">
              <span>{t("audit.loadError")}: {extractErrorMessage(error)}</span>
              <button onClick={() => refetch()} className="underline font-medium ml-2">
                {t("common.retry")}
              </button>
            </AlertDescription>
          </Alert>
        ) : isLoading ? (
          <div className="space-y-3">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} className="h-16 w-full rounded-lg" />
            ))}
          </div>
        ) : logs.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 text-center">
            <IconInbox className="size-12 text-muted-foreground/50 mb-4" />
            <h3 className="text-lg font-medium">{t("audit.noLogs")}</h3>
            <p className="text-sm text-muted-foreground mt-1">{t("audit.noLogsDescription")}</p>
          </div>
        ) : (
          <div className="space-y-4">
            {isMobile ? (
              <div className="space-y-2">
                {logs.map((log) => (
                  <AuditLogExpandable key={log.id} log={log} />
                ))}
              </div>
            ) : (
              <div className="space-y-2">
                {logs.map((log) => (
                  <AuditLogExpandable key={log.id} log={log} />
                ))}
              </div>
            )}

            {totalPages > 1 && (
              <PaginationBar
                page={filters.page || 1}
                total={total}
                pageSize={filters.page_size || 20}
                onPageChange={(p) => setFilters((f) => ({ ...f, page: p }))}
              />
            )}
          </div>
        )}
      </div>
    </>
  );
}
