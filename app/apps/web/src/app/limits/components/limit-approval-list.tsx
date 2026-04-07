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
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { PaginationBar } from "@/components/pagination";
import { IconAlertCircle } from "@tabler/icons-react";
import { useTranslation } from "@/lib/i18n";
import { useLimitApprovals } from "@/hooks/use-limits";
import type { LimitApproval, ApprovalFilters } from "@/hooks/use-limits";
import { formatDate } from "@/lib/utils";

const approvalStatusColors: Record<string, string> = {
  PENDING:
    "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300",
  APPROVED_RM:
    "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300",
  APPROVED_HEAD:
    "bg-emerald-100 text-emerald-800 dark:bg-emerald-900 dark:text-emerald-300",
  REJECTED_RM: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300",
  REJECTED_HEAD:
    "bg-rose-100 text-rose-800 dark:bg-rose-900 dark:text-rose-300",
};

function formatVND(value: number): string {
  return Number(value).toLocaleString("en-US", {
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  });
}

export function LimitApprovalList() {
  const { t } = useTranslation();
  const [filters, setFilters] = useState<ApprovalFilters>({
    page: 1,
    page_size: 20,
  });
  const [sorting, setSorting] = useState<SortingState>([]);

  const { data, isLoading, isError, error, refetch } =
    useLimitApprovals(filters);

  const approvals = data?.data ?? [];
  const total = data?.total ?? 0;
  const totalPages = data?.total_pages ?? 0;

  const columns: ColumnDef<LimitApproval>[] = [
    {
      accessorKey: "deal_module",
      header: t("limit.dealModule"),
      cell: ({ row }) => (
        <Badge variant="outline">{row.original.deal_module}</Badge>
      ),
    },
    {
      accessorKey: "deal_number",
      header: t("limit.dealNumber"),
      cell: ({ row }) => (
        <span className="font-mono text-sm">{row.original.deal_number}</span>
      ),
    },
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
      accessorKey: "amount_vnd",
      header: () => <div className="text-right">{t("limit.amountVND")}</div>,
      cell: ({ row }) => (
        <div className="text-right tabular-nums font-medium">
          {formatVND(row.original.amount_vnd)}
        </div>
      ),
    },
    {
      id: "rm_status",
      header: t("limit.rmApproval"),
      cell: ({ row }) => (
        <div>
          <Badge
            variant="secondary"
            className={approvalStatusColors[row.original.rm_status] ?? ""}
          >
            {t(`limit.approvalStatus.${row.original.rm_status}`)}
          </Badge>
          {row.original.rm_approved_by && (
            <div className="mt-1 text-xs text-muted-foreground">
              {row.original.rm_approved_by}
            </div>
          )}
        </div>
      ),
    },
    {
      id: "head_status",
      header: t("limit.headApproval"),
      cell: ({ row }) => (
        <div>
          <Badge
            variant="secondary"
            className={approvalStatusColors[row.original.head_status] ?? ""}
          >
            {t(`limit.approvalStatus.${row.original.head_status}`)}
          </Badge>
          {row.original.head_approved_by && (
            <div className="mt-1 text-xs text-muted-foreground">
              {row.original.head_approved_by}
            </div>
          )}
        </div>
      ),
    },
    {
      accessorKey: "created_at",
      header: t("limit.createdAt"),
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">
          {formatDate(row.original.created_at)}
        </span>
      ),
    },
  ];

  const table = useReactTable({
    data: approvals,
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
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            {t("common.retry")}
          </Button>
        </AlertDescription>
      </Alert>
    );
  }

  if (approvals.length === 0) {
    return (
      <div className="py-12 text-center text-muted-foreground">
        {t("limit.noApprovals")}
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
              {table.getRowModel().rows.map((row) => (
                <TableRow key={row.id}>
                  {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id} className="whitespace-nowrap">
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
        {approvals.map((approval) => (
          <div
            key={approval.id}
            className="rounded-lg border bg-card p-4 space-y-2"
          >
            <div className="flex items-center justify-between">
              <span className="font-mono text-sm font-medium">
                {approval.deal_number}
              </span>
              <Badge variant="outline">{approval.deal_module}</Badge>
            </div>
            <div className="text-sm">
              {approval.counterparty_name} · {approval.counterparty_code}
            </div>
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted-foreground">
                {t("limit.amountVND")}
              </span>
              <span className="font-medium tabular-nums">
                {formatVND(approval.amount_vnd)}
              </span>
            </div>
            <div className="flex gap-2">
              <Badge
                variant="secondary"
                className={approvalStatusColors[approval.rm_status] ?? ""}
              >
                CV: {t(`limit.approvalStatus.${approval.rm_status}`)}
              </Badge>
              <Badge
                variant="secondary"
                className={approvalStatusColors[approval.head_status] ?? ""}
              >
                TPB: {t(`limit.approvalStatus.${approval.head_status}`)}
              </Badge>
            </div>
          </div>
        ))}
      </div>

      {totalPages > 1 && (
        <PaginationBar
          page={filters.page ?? 1}
          total={total}
          pageSize={filters.page_size ?? 20}
          onPageChange={(page) => setFilters({ ...filters, page })}
        />
      )}
    </div>
  );
}
