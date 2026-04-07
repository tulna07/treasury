"use client";

import { useState } from "react";
import {
  useReactTable,
  getCoreRowModel,
  getSortedRowModel,
  type SortingState,
  type ColumnDef,
  flexRender,
} from "@tanstack/react-table";
import Link from "next/link";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { PaginationBar } from "@/components/pagination";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { IconAlertCircle, IconInbox } from "@tabler/icons-react";
import { useTranslation } from "@/lib/i18n";
import { extractErrorMessage, formatDate } from "@/lib/utils";
import { useIsMobile } from "@/hooks/use-mobile";
import { MMStatusBadge } from "./mm-status-badge";
import type { MMOMORepoDeal, MMFilters } from "@/hooks/use-mm";

interface Props {
  deals: MMOMORepoDeal[];
  total: number;
  totalPages: number;
  filters: MMFilters;
  onFiltersChange: (filters: MMFilters) => void;
  isLoading: boolean;
  isError: boolean;
  error: unknown;
  onRetry: () => void;
}

export function MMOMOList({
  deals,
  total,
  totalPages,
  filters,
  onFiltersChange,
  isLoading,
  isError,
  error,
  onRetry,
}: Props) {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const [sorting, setSorting] = useState<SortingState>([]);

  const columns: ColumnDef<MMOMORepoDeal>[] = [
    {
      accessorKey: "deal_number",
      header: t("mm.col.dealNumber"),
      cell: ({ row }) => (
        <Link
          href={`/mm/omo/${row.original.id}`}
          className="font-mono text-sm text-primary hover:underline"
        >
          {row.original.deal_number}
        </Link>
      ),
    },
    {
      accessorKey: "session_name",
      header: t("mm.col.session"),
    },
    {
      accessorKey: "counterparty_name",
      header: t("mm.col.counterparty"),
      cell: ({ row }) => (
        <span>{row.original.counterparty_code} – {row.original.counterparty_name}</span>
      ),
    },
    {
      accessorKey: "bond_code",
      header: t("mm.col.bondCode"),
      cell: ({ row }) => (
        <span className="font-mono text-sm">{row.original.bond_code}</span>
      ),
    },
    {
      accessorKey: "notional_amount",
      header: () => <div className="text-right">{t("mm.col.notional")}</div>,
      cell: ({ row }) => (
        <div className="text-right font-medium">
          {Number(row.original.notional_amount).toLocaleString("en-US")}
        </div>
      ),
    },
    {
      accessorKey: "winning_rate",
      header: () => <div className="text-right">{t("mm.col.winningRate")}</div>,
      cell: ({ row }) => (
        <div className="text-right">{row.original.winning_rate}%</div>
      ),
    },
    {
      accessorKey: "tenor_days",
      header: t("mm.col.tenor"),
      cell: ({ row }) => `${row.original.tenor_days}D`,
    },
    {
      accessorKey: "settlement_date_1",
      header: t("mm.col.settlementDate1"),
      cell: ({ row }) => formatDate(row.original.settlement_date_1),
    },
    {
      accessorKey: "settlement_date_2",
      header: t("mm.col.settlementDate2"),
      cell: ({ row }) => formatDate(row.original.settlement_date_2),
    },
    {
      accessorKey: "status",
      header: t("mm.col.status"),
      cell: ({ row }) => <MMStatusBadge status={row.original.status} />,
    },
  ];

  const table = useReactTable({
    data: deals,
    columns,
    state: { sorting },
    onSortingChange: setSorting,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    manualPagination: true,
    pageCount: totalPages,
  });

  if (isError) {
    return (
      <Alert variant="destructive">
        <IconAlertCircle className="size-4" />
        <AlertDescription className="flex items-center justify-between">
          <span>{t("mm.loadError")}: {extractErrorMessage(error)}</span>
          <button onClick={onRetry} className="underline font-medium ml-2">
            {t("common.retry")}
          </button>
        </AlertDescription>
      </Alert>
    );
  }

  if (isLoading) {
    return (
      <div className="space-y-3">
        {Array.from({ length: 5 }).map((_, i) => (
          <Skeleton key={i} className="h-14 w-full rounded-lg" />
        ))}
      </div>
    );
  }

  if (deals.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-center">
        <IconInbox className="size-12 text-muted-foreground/50 mb-4" />
        <h3 className="text-lg font-medium">{t("mm.noOMODeals")}</h3>
      </div>
    );
  }

  const currentPage = filters.page || 1;
  const pageSize = filters.page_size || 20;

  return (
    <div className="space-y-4">
      {isMobile ? (
        <div className="space-y-3">
          {deals.map((deal) => (
            <Link
              key={deal.id}
              href={`/mm/omo/${deal.id}`}
              className="block rounded-lg border p-4 hover:bg-accent/50 transition-colors"
            >
              <div className="flex items-center justify-between mb-2">
                <span className="font-mono text-sm font-medium">{deal.deal_number}</span>
                <MMStatusBadge status={deal.status} />
              </div>
              <p className="text-sm text-muted-foreground">
                {deal.session_name} · {deal.bond_code} · {Number(deal.notional_amount).toLocaleString("en-US")}
              </p>
              <p className="text-xs text-muted-foreground mt-1">
                {deal.winning_rate}% · {deal.tenor_days}D · {formatDate(deal.settlement_date_1)}
              </p>
            </Link>
          ))}
        </div>
      ) : (
        <div className="rounded-lg border overflow-x-auto">
          <Table>
            <TableHeader>
              {table.getHeaderGroups().map((headerGroup) => (
                <TableRow key={headerGroup.id}>
                  {headerGroup.headers.map((header) => (
                    <TableHead
                      key={header.id}
                      className={header.column.getCanSort() ? "cursor-pointer select-none" : ""}
                      onClick={header.column.getToggleSortingHandler()}
                    >
                      {header.isPlaceholder ? null : (
                        <div className="flex items-center gap-1">
                          {flexRender(header.column.columnDef.header, header.getContext())}
                          {header.column.getIsSorted() === "asc" && " ↑"}
                          {header.column.getIsSorted() === "desc" && " ↓"}
                        </div>
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
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </TableCell>
                  ))}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      {totalPages > 1 && (
        <PaginationBar
          page={currentPage}
          total={total}
          pageSize={pageSize}
          onPageChange={(p) => onFiltersChange({ ...filters, page: p })}
          itemLabel={t("mm.dealLabel").toLowerCase()}
        />
      )}
    </div>
  );
}
