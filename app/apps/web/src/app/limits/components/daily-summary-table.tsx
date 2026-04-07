"use client";

import { useState } from "react";
import {
  useReactTable,
  getCoreRowModel,
  getSortedRowModel,
  type ColumnDef,
  type SortingState,
  flexRender,
} from "@tanstack/react-table";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogClose,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import {
  IconAlertCircle,
  IconDownload,
  IconInfinity,
} from "@tabler/icons-react";
import { useTranslation } from "@/lib/i18n";
import { useDailySummary, useExportDailySummary } from "@/hooks/use-limits";
import type { DailySummaryRow } from "@/hooks/use-limits";
import { toast } from "sonner";
import { extractErrorMessage } from "@/lib/utils";

function formatVND(value: number): string {
  return Number(value).toLocaleString("en-US", {
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  });
}

function todayISO(): string {
  const d = new Date();
  return d.toISOString().split("T")[0];
}

export function DailySummaryTable() {
  const { t } = useTranslation();
  const [date, setDate] = useState(todayISO());
  const [exportOpen, setExportOpen] = useState(false);
  const [exportPassword, setExportPassword] = useState("");

  const { data, isLoading, isError, error, refetch } = useDailySummary(date);
  const exportMutation = useExportDailySummary();

  const rows = data?.data ?? [];
  const fxRate = data?.fx_rate_usd_vnd;

  const [sorting, setSorting] = useState<SortingState>([]);

  const columns: ColumnDef<DailySummaryRow>[] = [
    {
      accessorKey: "counterparty_name",
      header: t("limit.counterparty"),
      cell: ({ row }) => (
        <div>
          <div className="font-medium">{row.original.counterparty_name}</div>
          <div className="text-xs text-muted-foreground">
            {row.original.counterparty_code}
          </div>
        </div>
      ),
    },
    {
      accessorKey: "cif",
      header: "CIF",
      cell: ({ row }) => (
        <span className="font-mono text-xs">{row.original.cif}</span>
      ),
    },
    {
      accessorKey: "limit_type",
      header: t("limit.type"),
      cell: ({ row }) => (
        <Badge variant="secondary">
          {row.original.limit_type === "COLLATERALIZED"
            ? t("limit.collateralized")
            : t("limit.uncollateralized")}
        </Badge>
      ),
    },
    {
      id: "granted_limit",
      header: () => <div className="text-right">{t("limit.granted")}</div>,
      cell: ({ row }) => (
        <div className="text-right tabular-nums">
          {row.original.granted_unlimited ? (
            <span className="inline-flex items-center gap-1 text-muted-foreground">
              <IconInfinity className="size-3" />
              {t("limit.unlimited")}
            </span>
          ) : (
            <span className="font-medium">
              {formatVND(row.original.granted_limit ?? 0)}
            </span>
          )}
        </div>
      ),
    },
    {
      accessorKey: "mm_utilized",
      header: () => <div className="text-right">{t("limit.mmUtilized")}</div>,
      cell: ({ row }) => (
        <div className="text-right tabular-nums">
          {formatVND(row.original.mm_utilized)}
        </div>
      ),
    },
    {
      accessorKey: "bond_utilized",
      header: () => (
        <div className="text-right">{t("limit.bondUtilized")}</div>
      ),
      cell: ({ row }) => (
        <div className="text-right tabular-nums">
          {formatVND(row.original.bond_utilized)}
        </div>
      ),
    },
    {
      accessorKey: "fx_utilized",
      header: () => <div className="text-right">{t("limit.fxUtilized")}</div>,
      cell: ({ row }) => (
        <div className="text-right tabular-nums">
          {formatVND(row.original.fx_utilized)}
        </div>
      ),
    },
    {
      accessorKey: "total_utilized",
      header: () => (
        <div className="text-right">{t("limit.totalUtilized")}</div>
      ),
      cell: ({ row }) => (
        <div className="text-right tabular-nums font-semibold">
          {formatVND(row.original.total_utilized)}
        </div>
      ),
    },
    {
      id: "remaining",
      header: () => <div className="text-right">{t("limit.remaining")}</div>,
      cell: ({ row }) => {
        if (row.original.granted_unlimited) {
          return (
            <div className="text-right text-muted-foreground">
              {t("limit.unlimited")}
            </div>
          );
        }
        const remaining = row.original.remaining ?? 0;
        const isNegative = remaining < 0;
        return (
          <div
            className={`text-right tabular-nums font-semibold ${isNegative ? "text-red-600 dark:text-red-400" : ""}`}
          >
            {formatVND(remaining)}
          </div>
        );
      },
    },
    {
      accessorKey: "fx_rate",
      header: () => <div className="text-right">{t("limit.fxRate")}</div>,
      cell: ({ row }) => (
        <div className="text-right tabular-nums text-sm text-muted-foreground">
          {row.original.fx_rate
            ? Number(row.original.fx_rate).toLocaleString("en-US")
            : "—"}
        </div>
      ),
    },
  ];

  const table = useReactTable({
    data: rows,
    columns,
    state: { sorting },
    onSortingChange: setSorting,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
  });

  function handleExport() {
    exportMutation.mutate(
      { date, password: exportPassword },
      {
        onSuccess: () => {
          toast.success(t("limit.exportSuccess"));
          setExportOpen(false);
          setExportPassword("");
        },
        onError: (err) => toast.error(extractErrorMessage(err)),
      }
    );
  }

  return (
    <div className="space-y-4">
      {/* Controls */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-3">
          <Input
            type="date"
            value={date}
            onChange={(e) => setDate(e.target.value)}
            className="w-44"
          />
          {fxRate && (
            <Badge variant="outline" className="text-xs">
              USD/VND: {Number(fxRate).toLocaleString("en-US")}
            </Badge>
          )}
        </div>
        <Button
          variant="outline"
          onClick={() => setExportOpen(true)}
          className="shrink-0"
        >
          <IconDownload className="mr-2 size-4" />
          {t("limit.exportExcel")}
        </Button>
      </div>

      {/* Loading */}
      {isLoading && (
        <div className="space-y-3">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </div>
      )}

      {/* Error */}
      {isError && (
        <Alert variant="destructive">
          <IconAlertCircle className="size-4" />
          <AlertDescription className="flex items-center gap-2">
            {(error as { error?: string })?.error || t("limit.loadError")}
            <Button variant="outline" size="sm" onClick={() => refetch()}>
              {t("common.retry")}
            </Button>
          </AlertDescription>
        </Alert>
      )}

      {/* Table */}
      {!isLoading && !isError && (
        <div className="overflow-x-auto rounded-md border">
          <Table>
            <TableHeader>
              {table.getHeaderGroups().map((headerGroup) => (
                <TableRow key={headerGroup.id}>
                  {headerGroup.headers.map((header) => (
                    <TableHead key={header.id} className="whitespace-nowrap">
                      {header.isPlaceholder
                        ? null
                        : flexRender(
                            header.column.columnDef.header,
                            header.getContext()
                          )}
                    </TableHead>
                  ))}
                </TableRow>
              ))}
            </TableHeader>
            <TableBody>
              {table.getRowModel().rows.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={columns.length}
                    className="h-24 text-center text-muted-foreground"
                  >
                    {t("limit.noData")}
                  </TableCell>
                </TableRow>
              ) : (
                table.getRowModel().rows.map((row) => (
                  <TableRow key={row.id}>
                    {row.getVisibleCells().map((cell) => (
                      <TableCell
                        key={cell.id}
                        className="whitespace-nowrap"
                      >
                        {flexRender(
                          cell.column.columnDef.cell,
                          cell.getContext()
                        )}
                      </TableCell>
                    ))}
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      )}

      {/* Export dialog */}
      <Dialog open={exportOpen} onOpenChange={setExportOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{t("limit.exportExcel")}</DialogTitle>
          </DialogHeader>
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground">
              {t("limit.exportDescription")}
            </p>
            <div className="space-y-2">
              <Label>{t("limit.exportPassword")}</Label>
              <Input
                type="password"
                value={exportPassword}
                onChange={(e) => setExportPassword(e.target.value)}
                placeholder={t("limit.exportPasswordPlaceholder")}
              />
            </div>
          </div>
          <DialogFooter>
            <DialogClose render={<Button variant="outline" />}>
              {t("common.cancel")}
            </DialogClose>
            <Button
              onClick={handleExport}
              disabled={!exportPassword.trim() || exportMutation.isPending}
            >
              {exportMutation.isPending
                ? t("common.exporting")
                : t("limit.exportExcel")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
