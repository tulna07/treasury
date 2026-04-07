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
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { PaginationBar } from "@/components/pagination";
import {
  IconAlertCircle,
  IconEdit,
  IconInfinity,
} from "@tabler/icons-react";
import { useTranslation } from "@/lib/i18n";
import { useAbility } from "@/hooks/use-ability";
import type { CreditLimit, LimitFilters } from "@/hooks/use-limits";

function formatAmount(value: number | null, unlimited: boolean): string {
  if (unlimited) return "";
  if (value === null || value === undefined || isNaN(Number(value))) return "—";
  return Number(value).toLocaleString("en-US", {
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  });
}

interface LimitCounterpartyListProps {
  limits: CreditLimit[];
  total: number;
  totalPages: number;
  filters: LimitFilters;
  onFiltersChange: (filters: LimitFilters) => void;
  onEdit: (limit: CreditLimit) => void;
  isLoading: boolean;
  isError: boolean;
  error: Error | null;
  onRetry: () => void;
}

export function LimitCounterpartyList({
  limits,
  total,
  totalPages,
  filters,
  onFiltersChange,
  onEdit,
  isLoading,
  isError,
  error,
  onRetry,
}: LimitCounterpartyListProps) {
  const { t } = useTranslation();
  const ability = useAbility();
  const [sorting, setSorting] = useState<SortingState>([]);
  const canManage = ability.can("manage", "Limit");

  const columns: ColumnDef<CreditLimit>[] = [
    {
      accessorKey: "counterparty_name",
      header: t("limit.counterparty"),
      cell: ({ row }) => (
        <div>
          <span className="font-medium">{row.original.counterparty_name}</span>
          <span className="ml-2 text-xs text-muted-foreground">
            {row.original.counterparty_code}
          </span>
        </div>
      ),
    },
    {
      accessorKey: "cif",
      header: "CIF",
      cell: ({ row }) => (
        <span className="font-mono text-sm">{row.original.cif}</span>
      ),
    },
    {
      id: "uncollateralized",
      header: () => (
        <div className="text-right">{t("limit.uncollateralized")}</div>
      ),
      cell: ({ row }) => (
        <div className="text-right tabular-nums">
          {row.original.uncollateralized_unlimited ? (
            <Badge variant="secondary" className="gap-1">
              <IconInfinity className="size-3" />
              {t("limit.unlimited")}
            </Badge>
          ) : (
            <span className="font-medium">
              {formatAmount(
                row.original.uncollateralized_limit,
                false
              )}
            </span>
          )}
        </div>
      ),
    },
    {
      id: "collateralized",
      header: () => (
        <div className="text-right">{t("limit.collateralized")}</div>
      ),
      cell: ({ row }) => (
        <div className="text-right tabular-nums">
          {row.original.collateralized_unlimited ? (
            <Badge variant="secondary" className="gap-1">
              <IconInfinity className="size-3" />
              {t("limit.unlimited")}
            </Badge>
          ) : (
            <span className="font-medium">
              {formatAmount(row.original.collateralized_limit, false)}
            </span>
          )}
        </div>
      ),
    },
    {
      id: "uncollateralized_remaining",
      header: () => (
        <div className="text-right">{t("limit.remainingUncollateralized")}</div>
      ),
      cell: ({ row }) => {
        const remaining = row.original.uncollateralized_remaining;
        const isNegative = remaining !== null && remaining < 0;
        return (
          <div
            className={`text-right tabular-nums font-medium ${isNegative ? "text-red-600 dark:text-red-400" : ""}`}
          >
            {row.original.uncollateralized_unlimited
              ? t("limit.unlimited")
              : formatAmount(remaining, false)}
          </div>
        );
      },
    },
    {
      id: "collateralized_remaining",
      header: () => (
        <div className="text-right">{t("limit.remainingCollateralized")}</div>
      ),
      cell: ({ row }) => {
        const remaining = row.original.collateralized_remaining;
        const isNegative = remaining !== null && remaining < 0;
        return (
          <div
            className={`text-right tabular-nums font-medium ${isNegative ? "text-red-600 dark:text-red-400" : ""}`}
          >
            {row.original.collateralized_unlimited
              ? t("limit.unlimited")
              : formatAmount(remaining, false)}
          </div>
        );
      },
    },
  ];

  if (canManage) {
    columns.push({
      id: "actions",
      cell: ({ row }) => (
        <Button
          variant="ghost"
          size="icon"
          onClick={() => onEdit(row.original)}
        >
          <IconEdit className="size-4" />
        </Button>
      ),
    });
  }

  const table = useReactTable({
    data: limits,
    columns,
    state: { sorting },
    onSortingChange: setSorting,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    manualPagination: true,
    pageCount: totalPages,
  });

  if (isLoading) {
    return (
      <div className="space-y-3">
        {Array.from({ length: 5 }).map((_, i) => (
          <Skeleton key={i} className="h-12 w-full" />
        ))}
      </div>
    );
  }

  if (isError) {
    return (
      <Alert variant="destructive">
        <IconAlertCircle className="size-4" />
        <AlertDescription className="flex items-center gap-2">
          {(error as { error?: string })?.error || t("limit.loadError")}
          <Button variant="outline" size="sm" onClick={onRetry}>
            {t("common.retry")}
          </Button>
        </AlertDescription>
      </Alert>
    );
  }

  if (limits.length === 0) {
    return (
      <div className="py-12 text-center text-muted-foreground">
        {t("limit.noLimits")}
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Desktop table */}
      <div className="hidden md:block">
        <div className="overflow-x-auto rounded-md border">
          <Table>
            <TableHeader>
              {table.getHeaderGroups().map((headerGroup) => (
                <TableRow key={headerGroup.id}>
                  {headerGroup.headers.map((header) => (
                    <TableHead key={header.id}>
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
              {table.getRowModel().rows.map((row) => (
                <TableRow key={row.id}>
                  {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id}>
                      {flexRender(
                        cell.column.columnDef.cell,
                        cell.getContext()
                      )}
                    </TableCell>
                  ))}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </div>

      {/* Mobile cards */}
      <div className="flex flex-col gap-3 md:hidden">
        {limits.map((limit) => (
          <div
            key={limit.counterparty_id}
            className="rounded-lg border bg-card p-4"
          >
            <div className="flex items-center justify-between">
              <div>
                <div className="font-medium">
                  {limit.counterparty_name}
                </div>
                <div className="text-xs text-muted-foreground">
                  {limit.counterparty_code} · {limit.cif}
                </div>
              </div>
              {canManage && (
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={(e) => {
                    e.preventDefault();
                    onEdit(limit);
                  }}
                >
                  <IconEdit className="size-4" />
                </Button>
              )}
            </div>
            <div className="mt-2 space-y-1 text-sm">
              <div className="flex justify-between">
                <span className="text-muted-foreground">
                  {t("limit.uncollateralized")}
                </span>
                <span className="font-medium tabular-nums">
                  {limit.uncollateralized_unlimited
                    ? t("limit.unlimited")
                    : formatAmount(limit.uncollateralized_limit, false)}
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">
                  {t("limit.collateralized")}
                </span>
                <span className="font-medium tabular-nums">
                  {limit.collateralized_unlimited
                    ? t("limit.unlimited")
                    : formatAmount(limit.collateralized_limit, false)}
                </span>
              </div>
            </div>
          </div>
        ))}
      </div>

      {totalPages > 1 && (
        <PaginationBar
          page={filters.page ?? 1}
          total={total}
          pageSize={filters.page_size ?? 20}
          onPageChange={(page) =>
            onFiltersChange({ ...filters, page })
          }
        />
      )}
    </div>
  );
}
