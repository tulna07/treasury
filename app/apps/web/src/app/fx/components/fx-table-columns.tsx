"use client";

import { type ColumnDef } from "@tanstack/react-table";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Badge } from "@/components/ui/badge";
import { IconDots, IconEye, IconEdit, IconCopy, IconTrash } from "@tabler/icons-react";
import Link from "next/link";
import { FxStatusBadge } from "./fx-status-badge";
import type { FxDeal } from "@/hooks/use-fx";
import { formatDate } from "@/lib/utils";

interface ColumnOptions {
  t: (key: string) => string;
  onView: (deal: FxDeal) => void;
  onEdit?: (deal: FxDeal) => void;
  onClone?: (deal: FxDeal) => void;
  onDelete?: (deal: FxDeal) => void;
  canEdit: boolean;
  canDelete: boolean;
}

export function getFxColumns({
  t,
  onView,
  onEdit,
  onClone,
  onDelete,
  canEdit,
  canDelete,
}: ColumnOptions): ColumnDef<FxDeal>[] {
  return [
    {
      accessorKey: "ticket_number",
      header: t("fx.ticketNumber"),
      cell: ({ row }) => (
        <Link
          href={`/fx/${row.original.id}`}
          className="font-mono text-sm text-primary hover:underline"
        >
          {row.original.ticket_number || row.original.id.slice(0, 8)}
        </Link>
      ),
    },
    {
      accessorKey: "deal_type",
      header: t("fx.dealType"),
      cell: ({ row }) => t(`fx.type.${row.original.deal_type}`),
    },
    {
      accessorKey: "direction",
      header: t("fx.direction"),
      cell: ({ row }) => t(`fx.direction.${row.original.direction}`),
    },
    {
      accessorKey: "counterparty_name",
      header: t("fx.counterparty"),
    },
    {
      accessorKey: "notional_amount",
      header: () => <div className="text-right">{t("fx.notionalAmount")}</div>,
      cell: ({ row }) => (
        <div className="text-right font-medium tabular-nums">
          {Number(row.original.notional_amount).toLocaleString("en-US", {
            minimumFractionDigits: 2,
            maximumFractionDigits: 2,
          })}
        </div>
      ),
    },
    {
      accessorKey: "currency_code",
      header: t("fx.currency"),
      cell: ({ row }) => (
        <span className="font-mono text-sm">{row.original.currency_code}</span>
      ),
    },
    {
      accessorKey: "trade_date",
      header: t("fx.tradeDate"),
      cell: ({ row }) => formatDate(row.original.trade_date),
    },
    {
      accessorKey: "status",
      header: t("table.status"),
      cell: ({ row }) => <FxStatusBadge status={row.original.status} />,
    },
    {
      id: "is_international",
      header: "Intl Settlement",
      cell: ({ row }) => (
        <Badge variant={row.original.is_international ? "default" : "secondary"}>
          {row.original.is_international ? t("common.yes") : t("common.no")}
        </Badge>
      ),
      enableHiding: true,
    },
    {
      id: "settlement_amount",
      header: () => <div className="text-right">{t("fx.settlementAmount")}</div>,
      cell: ({ row }) =>
        row.original.settlement_amount ? (
          <div className="text-right font-medium tabular-nums">
            {Number(row.original.settlement_amount).toLocaleString("en-US", {
              minimumFractionDigits: 2,
              maximumFractionDigits: 2,
            })}{" "}
            <span className="font-mono text-xs">{row.original.settlement_currency}</span>
          </div>
        ) : (
          <div className="text-right text-muted-foreground">—</div>
        ),
      enableHiding: true,
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
                {t("fx.action.view")}
              </DropdownMenuItem>
              {canEdit && isEditable && onEdit && (
                <DropdownMenuItem onClick={() => onEdit(deal)}>
                  <IconEdit className="mr-2 size-4" />
                  {t("fx.action.edit")}
                </DropdownMenuItem>
              )}
              {onClone && (
                <DropdownMenuItem onClick={() => onClone(deal)}>
                  <IconCopy className="mr-2 size-4" />
                  {t("fx.action.clone")}
                </DropdownMenuItem>
              )}
              {canDelete && isEditable && onDelete && (
                <DropdownMenuItem
                  onClick={() => onDelete(deal)}
                  className="text-destructive focus:text-destructive"
                >
                  <IconTrash className="mr-2 size-4" />
                  {t("fx.action.delete")}
                </DropdownMenuItem>
              )}
            </DropdownMenuContent>
          </DropdownMenu>
        );
      },
    },
  ];
}
