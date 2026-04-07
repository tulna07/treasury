"use client";

import * as React from "react";
import { IconChevronLeft, IconChevronRight } from "@tabler/icons-react";
import { Button } from "@/components/ui/button";

interface PaginationBarProps {
  /** Current page (1-based) */
  page: number;
  /** Total number of items */
  total: number;
  /** Items per page */
  pageSize: number;
  /** Callback when page changes (1-based) */
  onPageChange: (page: number) => void;
  /** Label for items (default: "mục") */
  itemLabel?: string;
  /** Max page buttons to show (default: 7 desktop, 5 mobile) */
  maxButtons?: number;
}

export function PaginationBar({
  page,
  total,
  pageSize,
  onPageChange,
  itemLabel = "mục",
  maxButtons = 7,
}: PaginationBarProps) {
  const totalPages = Math.ceil(total / pageSize);
  if (totalPages <= 1) return null;

  const from = (page - 1) * pageSize + 1;
  const to = Math.min(page * pageSize, total);

  const mobileMax = Math.min(maxButtons, 5);

  function getPageNumbers(max: number) {
    const pages: number[] = [];
    const count = Math.min(totalPages, max);
    for (let i = 0; i < count; i++) {
      let pageNum: number;
      if (totalPages <= max) {
        pageNum = i + 1;
      } else if (page <= Math.ceil(max / 2)) {
        pageNum = i + 1;
      } else if (page >= totalPages - Math.floor(max / 2)) {
        pageNum = totalPages - count + i + 1;
      } else {
        pageNum = page - Math.floor(max / 2) + i;
      }
      pages.push(pageNum);
    }
    return pages;
  }

  const desktopPages = getPageNumbers(maxButtons);
  const mobilePages = getPageNumbers(mobileMax);

  return (
    <div className="flex flex-col items-center gap-2 pt-2 sm:flex-row sm:justify-between">
      {/* Desktop: show range */}
      <p className="text-sm text-muted-foreground hidden sm:block">
        Hiển thị {from}–{to} / {total} {itemLabel}
      </p>
      {/* Mobile: show page number */}
      <p className="text-xs text-muted-foreground sm:hidden">
        Trang {page}/{totalPages}
      </p>

      <div className="flex items-center gap-1">
        <Button
          variant="outline"
          size="icon"
          disabled={page <= 1}
          onClick={() => onPageChange(page - 1)}
        >
          <IconChevronLeft className="size-4" />
        </Button>

        {/* Desktop page buttons */}
        <div className="hidden sm:flex items-center gap-1">
          {desktopPages.map((p) => (
            <Button
              key={p}
              variant={p === page ? "default" : "outline"}
              size="icon"
              onClick={() => onPageChange(p)}
            >
              {p}
            </Button>
          ))}
        </div>

        {/* Mobile page buttons */}
        <div className="flex sm:hidden items-center gap-1">
          {mobilePages.map((p) => (
            <Button
              key={p}
              variant={p === page ? "default" : "outline"}
              size="icon"
              className="size-8 text-xs"
              onClick={() => onPageChange(p)}
            >
              {p}
            </Button>
          ))}
        </div>

        <Button
          variant="outline"
          size="icon"
          disabled={page >= totalPages}
          onClick={() => onPageChange(page + 1)}
        >
          <IconChevronRight className="size-4" />
        </Button>
      </div>
    </div>
  );
}
