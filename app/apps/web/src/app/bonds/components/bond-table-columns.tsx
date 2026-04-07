"use client";

import { type ColumnDef } from "@tanstack/react-table";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { IconDots, IconEye, IconEdit, IconCopy, IconTrash } from "@tabler/icons-react";
import Link from "next/link";
import { BondStatusBadge } from "./bond-status-badge";
import type { BondDeal } from "@/hooks/use-bonds";
import { formatDate } from "@/lib/utils";

interface ColumnOptions {
  t: (key: string) => string;
  onView: (deal: BondDeal) => void;
  onEdit?: (deal: BondDeal) => void;
  onClone?: (deal: BondDeal) => void;
  onDelete?: (deal: BondDeal) => void;
  canEdit: boolean;
  canDelete: boolean;
}

export function getBondColumns({
  t,
  onView,
  onEdit,
  onClone,
  onDelete,
  canEdit,
  canDelete,
}: ColumnOptions): ColumnDef<BondDeal>[] {
  return [
    {
      accessorKey: "deal_number",
      header: t("bond.dealNumber"),
      cell: ({ row }) => (
        <Link
          href={`/bonds/${row.original.id}`}
          className="font-mono text-sm text-primary hover:underline"
        >
          {row.original.deal_number || row.original.id.slice(0, 8)}
        </Link>
      ),
    },
    {
      accessorKey: "bond_category",
      header: t("bond.category"),
      cell: ({ row }) => t(`bond.category.${row.original.bond_category}`),
    },
    {
      accessorKey: "direction",
      header: t("bond.direction"),
      cell: ({ row }) => t(`bond.direction.${row.original.direction}`),
    },
    {
      accessorKey: "counterparty_name",
      header: t("bond.counterparty"),
    },
    {
      accessorKey: "transaction_type",
      header: t("bond.transactionType"),
      cell: ({ row }) => {
        if (row.original.transaction_type === "OTHER") {
          return row.original.transaction_type_other || t("bond.txType.OTHER");
        }
        return t(`bond.txType.${row.original.transaction_type}`);
      },
    },
    {
      accessorKey: "bond_code_display",
      header: t("bond.bondCode"),
      cell: ({ row }) => (
        <span className="font-mono text-sm">{row.original.bond_code_display}</span>
      ),
    },
    {
      accessorKey: "quantity",
      header: () => <div className="text-right">{t("bond.quantity")}</div>,
      cell: ({ row }) => (
        <div className="text-right font-medium tabular-nums">
          {Number(row.original.quantity).toLocaleString("en-US")}
        </div>
      ),
    },
    {
      accessorKey: "total_value",
      header: () => <div className="text-right">{t("bond.totalValue")}</div>,
      cell: ({ row }) => (
        <div className="text-right font-medium tabular-nums">
          {Number(row.original.total_value).toLocaleString("en-US", {
            minimumFractionDigits: 0,
            maximumFractionDigits: 0,
          })}
        </div>
      ),
    },
    {
      accessorKey: "status",
      header: t("table.status"),
      cell: ({ row }) => <BondStatusBadge status={row.original.status} />,
    },
    {
      accessorKey: "trade_date",
      header: t("bond.tradeDate"),
      cell: ({ row }) => formatDate(row.original.trade_date),
    },
    {
      accessorKey: "created_by_name",
      header: t("bond.createdBy"),
    },
    {
      id: "actions",
      header: "",
      cell: ({ row }) => {
        const deal = row.original;
        const isEditable = deal.status === "OPEN";
        return (
          <DropdownMenu>
            <DropdownMenuTrigger
              render={<Button variant="ghost" size="icon" className="size-8" />}
            >
              <IconDots className="size-4" />
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => onView(deal)}>
                <IconEye className="mr-2 size-4" />
                {t("bond.action.view")}
              </DropdownMenuItem>
              {canEdit && isEditable && onEdit && (
                <DropdownMenuItem onClick={() => onEdit(deal)}>
                  <IconEdit className="mr-2 size-4" />
                  {t("bond.action.edit")}
                </DropdownMenuItem>
              )}
              {onClone && (
                <DropdownMenuItem onClick={() => onClone(deal)}>
                  <IconCopy className="mr-2 size-4" />
                  {t("bond.action.clone")}
                </DropdownMenuItem>
              )}
              {canDelete && isEditable && onDelete && (
                <DropdownMenuItem
                  onClick={() => onDelete(deal)}
                  className="text-destructive focus:text-destructive"
                >
                  <IconTrash className="mr-2 size-4" />
                  {t("bond.action.delete")}
                </DropdownMenuItem>
              )}
            </DropdownMenuContent>
          </DropdownMenu>
        );
      },
    },
  ];
}
