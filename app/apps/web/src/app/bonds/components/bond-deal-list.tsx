"use client";

import {
  useReactTable,
  getCoreRowModel,
  getSortedRowModel,
  type SortingState,
  flexRender,
} from "@tanstack/react-table";
import { useState } from "react";
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
import { extractErrorMessage } from "@/lib/utils";
import { useIsMobile } from "@/hooks/use-mobile";
import { getBondColumns } from "./bond-table-columns";
import { BondCardView } from "./bond-card-view";
import type { BondDeal, BondFilters } from "@/hooks/use-bonds";
import { useAbility } from "@/hooks/use-ability";

interface BondDealListProps {
  deals: BondDeal[];
  total: number;
  totalPages: number;
  filters: BondFilters;
  onFiltersChange: (filters: BondFilters) => void;
  onView: (deal: BondDeal) => void;
  onEdit: (deal: BondDeal) => void;
  onClone: (deal: BondDeal) => void;
  onDelete: (deal: BondDeal) => void;
  isLoading: boolean;
  isError: boolean;
  error: unknown;
  onRetry: () => void;
}

export function BondDealList({
  deals,
  total,
  totalPages,
  filters,
  onFiltersChange,
  onView,
  onEdit,
  onClone,
  onDelete,
  isLoading,
  isError,
  error,
  onRetry,
}: BondDealListProps) {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const ability = useAbility();
  const [sorting, setSorting] = useState<SortingState>([]);

  const canEdit = ability.can("update", "GTCGTransaction");
  const canDelete =
    ability.can("delete", "GTCGTransaction") || ability.can("update", "GTCGTransaction");

  const columns = getBondColumns({
    t,
    onView,
    onEdit,
    onClone,
    onDelete,
    canEdit,
    canDelete,
  });

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
          <span>{t("bond.loadError")}: {extractErrorMessage(error)}</span>
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
        <h3 className="text-lg font-medium">{t("bond.noDeals")}</h3>
        <p className="text-sm text-muted-foreground mt-1">{t("bond.noDealsDescription")}</p>
      </div>
    );
  }

  const currentPage = filters.page || 1;
  const pageSize = filters.page_size || 20;

  return (
    <div className="space-y-4">
      {isMobile ? (
        <BondCardView deals={deals} onView={onView} />
      ) : (
        <div className="rounded-lg border overflow-x-auto">
          <Table>
            <TableHeader>
              {table.getHeaderGroups().map((headerGroup) => (
                <TableRow key={headerGroup.id}>
                  {headerGroup.headers.map((header) => (
                    <TableHead
                      key={header.id}
                      className={
                        header.column.getCanSort()
                          ? "cursor-pointer select-none"
                          : ""
                      }
                      onClick={header.column.getToggleSortingHandler()}
                    >
                      {header.isPlaceholder ? null : (
                        <div className="flex items-center gap-1">
                          {flexRender(
                            header.column.columnDef.header,
                            header.getContext()
                          )}
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
          itemLabel={t("bond.dealList").toLowerCase()}
        />
      )}
    </div>
  );
}
