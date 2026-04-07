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
import { Badge } from "@/components/ui/badge";
import { IconAlertCircle, IconInbox } from "@tabler/icons-react";
import { useTranslation } from "@/lib/i18n";
import { extractErrorMessage, formatDate } from "@/lib/utils";
import { useIsMobile } from "@/hooks/use-mobile";
import { MMStatusBadge } from "./mm-status-badge";
import type { MMInterbankDeal, MMFilters } from "@/hooks/use-mm";

interface Props {
  deals: MMInterbankDeal[];
  total: number;
  totalPages: number;
  filters: MMFilters;
  onFiltersChange: (filters: MMFilters) => void;
  isLoading: boolean;
  isError: boolean;
  error: unknown;
  onRetry: () => void;
}

const directionStyles: Record<string, string> = {
  PLACE: "bg-emerald-100 text-emerald-800 dark:bg-emerald-900/50 dark:text-emerald-300",
  TAKE: "bg-orange-100 text-orange-800 dark:bg-orange-900/50 dark:text-orange-300",
  LEND: "bg-emerald-100 text-emerald-800 dark:bg-emerald-900/50 dark:text-emerald-300",
  BORROW: "bg-orange-100 text-orange-800 dark:bg-orange-900/50 dark:text-orange-300",
};

export function MMInterbankList({
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

  const columns: ColumnDef<MMInterbankDeal>[] = [
    {
      accessorKey: "deal_number",
      header: t("mm.col.dealNumber"),
      cell: ({ row }) => (
        <Link
          href={`/mm/interbank/${row.original.id}`}
          className="font-mono text-sm text-primary hover:underline"
        >
          {row.original.deal_number}
        </Link>
      ),
    },
    {
      accessorKey: "counterparty_name",
      header: t("mm.col.counterparty"),
    },
    {
      accessorKey: "direction",
      header: t("mm.col.direction"),
      cell: ({ row }) => (
        <Badge variant="secondary" className={directionStyles[row.original.direction] || ""}>
          {t(`mm.direction.${row.original.direction}`)}
        </Badge>
      ),
    },
    {
      accessorKey: "currency_code",
      header: t("mm.col.currency"),
    },
    {
      accessorKey: "principal_amount",
      header: () => <div className="text-right">{t("mm.col.principal")}</div>,
      cell: ({ row }) => (
        <div className="text-right font-medium">
          {Number(row.original.principal_amount).toLocaleString("en-US")}
        </div>
      ),
    },
    {
      accessorKey: "interest_rate",
      header: () => <div className="text-right">{t("mm.col.rate")}</div>,
      cell: ({ row }) => (
        <div className="text-right">{row.original.interest_rate}%</div>
      ),
    },
    {
      accessorKey: "tenor_days",
      header: t("mm.col.tenor"),
      cell: ({ row }) => `${row.original.tenor_days}D`,
    },
    {
      accessorKey: "maturity_date",
      header: t("mm.col.maturityDate"),
      cell: ({ row }) => formatDate(row.original.maturity_date),
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
        <h3 className="text-lg font-medium">{t("mm.noDeals")}</h3>
        <p className="text-sm text-muted-foreground mt-1">{t("mm.noDealsDescription")}</p>
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
              href={`/mm/interbank/${deal.id}`}
              className="block rounded-lg border p-4 hover:bg-accent/50 transition-colors"
            >
              <div className="flex items-center justify-between mb-2">
                <span className="font-mono text-sm font-medium">{deal.deal_number}</span>
                <MMStatusBadge status={deal.status} />
              </div>
              <p className="text-sm text-muted-foreground">
                {t(`mm.direction.${deal.direction}`)} · {deal.counterparty_name} · {Number(deal.principal_amount).toLocaleString("en-US")} {deal.currency_code}
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
