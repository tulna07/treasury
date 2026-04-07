"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Header } from "@/components/header";
import { useTranslation } from "@/lib/i18n";
import { useAbility } from "@/hooks/use-ability";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { PaginationBar } from "@/components/pagination";
import {
  IconPlus,
  IconAlertCircle,
  IconInbox,
  IconSearch,
  IconTrash,
} from "@tabler/icons-react";
import { useIsMobile } from "@/hooks/use-mobile";
import { toast } from "sonner";
import { extractErrorMessage } from "@/lib/utils";
import {
  useMasterCounterparties,
  useDeleteCounterparty,
  type MasterCounterparty,
  type CounterpartyFilters,
} from "@/hooks/use-master-data";

export default function CounterpartiesPage() {
  const { t } = useTranslation();
  const ability = useAbility();
  const router = useRouter();
  const isMobile = useIsMobile();
  const [filters, setFilters] = useState<CounterpartyFilters>({
    page: 1,
    page_size: 20,
  });
  const [search, setSearch] = useState("");
  const deleteMutation = useDeleteCounterparty();

  const { data, isLoading, isError, error, refetch } = useMasterCounterparties(filters);
  const counterparties = data?.data ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.ceil(total / (filters.page_size || 20));

  function handleSearch() {
    setFilters((f) => ({ ...f, search: search || undefined, page: 1 }));
  }

  function handleDelete(cp: MasterCounterparty) {
    if (!confirm(t("counterparties.deleteConfirm"))) return;
    deleteMutation.mutate(cp.id, {
      onSuccess: () => toast.success(t("counterparties.deleteSuccess")),
      onError: (err) => toast.error(extractErrorMessage(err)),
    });
  }

  return (
    <>
      <Header />
      <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
        <div className="flex flex-col gap-4 pt-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight">{t("counterparties.title")}</h1>
            <p className="text-muted-foreground">{t("counterparties.description")}</p>
          </div>
          {ability.can("manage", "Settings") && (
            <Button onClick={() => router.push("/settings/counterparties/new")} className="shrink-0">
              <IconPlus className="mr-2 size-4" />
              {t("counterparties.new")}
            </Button>
          )}
        </div>

        {/* Search */}
        <div className="relative">
          <IconSearch className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder={t("counterparties.search")}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && handleSearch()}
            className="pl-9"
          />
        </div>

        {/* Content */}
        {isError ? (
          <Alert variant="destructive">
            <IconAlertCircle className="size-4" />
            <AlertDescription className="flex items-center justify-between">
              <span>{t("counterparties.loadError")}: {extractErrorMessage(error)}</span>
              <button onClick={() => refetch()} className="underline font-medium ml-2">
                {t("common.retry")}
              </button>
            </AlertDescription>
          </Alert>
        ) : isLoading ? (
          <div className="space-y-3">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} className="h-14 w-full rounded-lg" />
            ))}
          </div>
        ) : counterparties.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 text-center">
            <IconInbox className="size-12 text-muted-foreground/50 mb-4" />
            <h3 className="text-lg font-medium">{t("counterparties.noCounterparties")}</h3>
            <p className="text-sm text-muted-foreground mt-1">{t("counterparties.noCounterpartiesDescription")}</p>
          </div>
        ) : (
          <div className="space-y-4">
            {isMobile ? (
              <div className="space-y-3">
                {counterparties.map((cp) => (
                  <div
                    key={cp.id}
                    className="rounded-lg border p-4 space-y-2 cursor-pointer hover:bg-accent/50 transition-colors"
                    onClick={() => router.push(`/settings/counterparties/${cp.id}/edit`)}
                  >
                    <div className="flex items-center justify-between">
                      <span className="font-medium">{cp.full_name}</span>
                      <Badge variant={cp.is_active ? "outline" : "destructive"} className={cp.is_active ? "border-green-500/50 bg-green-500/10 text-green-700 dark:text-green-400" : ""}>
                        {cp.is_active ? t("common.active") : t("common.locked")}
                      </Badge>
                    </div>
                    <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-sm">
                      <div><span className="text-muted-foreground">{t("counterparties.code")}:</span> <span className="font-mono">{cp.code}</span></div>
                      <div><span className="text-muted-foreground">{t("counterparties.swiftCode")}:</span> <span className="font-mono">{cp.swift_code || "-"}</span></div>
                      <div><span className="text-muted-foreground">{t("counterparties.country")}:</span> {cp.country_code || "-"}</div>
                      <div><span className="text-muted-foreground">{t("counterparties.cif")}:</span> {cp.cif || "-"}</div>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="rounded-lg border overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{t("counterparties.code")}</TableHead>
                      <TableHead>{t("counterparties.name")}</TableHead>
                      <TableHead>{t("counterparties.swiftCode")}</TableHead>
                      <TableHead>{t("counterparties.country")}</TableHead>
                      <TableHead>{t("counterparties.cif")}</TableHead>
                      <TableHead>{t("common.status")}</TableHead>
                      {ability.can("manage", "Settings") && (
                        <TableHead className="w-[60px]">{t("common.actions")}</TableHead>
                      )}
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {counterparties.map((cp) => (
                      <TableRow
                        key={cp.id}
                        className="cursor-pointer"
                        onClick={() => router.push(`/settings/counterparties/${cp.id}/edit`)}
                      >
                        <TableCell className="font-mono font-medium">{cp.code}</TableCell>
                        <TableCell className="font-medium">{cp.full_name}</TableCell>
                        <TableCell className="font-mono">{cp.swift_code || "-"}</TableCell>
                        <TableCell>{cp.country_code || "-"}</TableCell>
                        <TableCell>{cp.cif || "-"}</TableCell>
                        <TableCell>
                          <Badge variant="outline" className={cp.is_active ? "border-green-500/50 bg-green-500/10 text-green-700 dark:text-green-400" : "border-red-500/50 bg-red-500/10 text-red-700 dark:text-red-400"}>
                            {cp.is_active ? t("common.active") : t("common.locked")}
                          </Badge>
                        </TableCell>
                        {ability.can("manage", "Settings") && (
                          <TableCell>
                            <button
                              className="text-muted-foreground hover:text-destructive transition-colors"
                              onClick={(e) => {
                                e.stopPropagation();
                                handleDelete(cp);
                              }}
                            >
                              <IconTrash className="size-4" />
                            </button>
                          </TableCell>
                        )}
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
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
