"use client";

import { Input } from "@/components/ui/input";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { DatePicker } from "@/components/ui/date-picker";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Button } from "@/components/ui/button";
import { IconSearch, IconX } from "@tabler/icons-react";
import { useTranslation } from "@/lib/i18n";
import type { FxFilters } from "@/hooks/use-fx";

interface FxFiltersBarProps {
  filters: FxFilters;
  onFiltersChange: (filters: FxFilters) => void;
}

const FX_STATUSES = [
  "OPEN",
  "PENDING_L2_APPROVAL",
  "PENDING_CHIEF_ACCOUNTANT",
  "PENDING_BOOKING",
  "APPROVED",
  "COMPLETED",
  "REJECTED",
  "CANCELLED",
] as const;

const FX_DEAL_TYPES = ["SPOT", "FORWARD", "SWAP"] as const;

const STATUS_LABELS: Record<string, string> = Object.fromEntries(
  ["ALL", ...FX_STATUSES].map((s) => [s, s === "ALL" ? "fx.allStatuses" : `fx.status.${s}`])
);
const TYPE_LABELS: Record<string, string> = Object.fromEntries(
  ["ALL", ...FX_DEAL_TYPES].map((s) => [s, s === "ALL" ? "fx.allTypes" : `fx.type.${s}`])
);

export function FxFiltersBar({ filters, onFiltersChange }: FxFiltersBarProps) {
  const { t } = useTranslation();

  const renderStatus = (v: string) => t(STATUS_LABELS[v] ?? `fx.status.${v}`);
  const renderType = (v: string) => t(TYPE_LABELS[v] ?? `fx.type.${v}`);

  const hasActiveFilters =
    filters.status || filters.deal_type || filters.ticket_number || filters.from_date || filters.to_date;

  function clearFilters() {
    onFiltersChange({
      page: 1,
      page_size: filters.page_size,
      sort_by: filters.sort_by,
      sort_dir: filters.sort_dir,
    });
  }

  return (
    <div className="flex flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-center">
      <div className="relative flex-1 min-w-[200px] max-w-sm">
        <IconSearch className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
        <Input
          placeholder={t("fx.search")}
          value={filters.ticket_number || ""}
          onChange={(e) =>
            onFiltersChange({ ...filters, ticket_number: e.target.value, page: 1 })
          }
          className="pl-9"
        />
      </div>

      <Select
        value={filters.status || "__ALL__"}
        onValueChange={(v) =>
          onFiltersChange({ ...filters, status: v === "__ALL__" || !v ? undefined : v, page: 1 })
        }
      >
        <SelectTrigger className="w-full sm:w-auto">
          <SelectValue placeholder={t("fx.allStatuses")}>
            {(v: string) => v === "__ALL__" ? t("fx.allStatuses") : t(`fx.status.${v}`)}
          </SelectValue>
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="__ALL__" label={t("fx.allStatuses")}>{t("fx.allStatuses")}</SelectItem>
          {FX_STATUSES.map((s) => (
            <SelectItem key={s} value={s} label={t(`fx.status.${s}`)}>
              {t(`fx.status.${s}`)}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      <Select
        value={filters.deal_type || "__ALL__"}
        onValueChange={(v) =>
          onFiltersChange({ ...filters, deal_type: v === "__ALL__" || !v ? undefined : v, page: 1 })
        }
      >
        <SelectTrigger className="w-full sm:w-auto">
          <SelectValue placeholder={t("fx.allTypes")}>
            {(v: string) => v === "__ALL__" ? t("fx.allTypes") : t(`fx.type.${v}`)}
          </SelectValue>
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="__ALL__" label={t("fx.allTypes")}>{t("fx.allTypes")}</SelectItem>
          {FX_DEAL_TYPES.map((dt) => (
            <SelectItem key={dt} value={dt} label={t(`fx.type.${dt}`)}>
              {t(`fx.type.${dt}`)}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      <DatePicker
        className="w-full sm:w-[180px]"
        placeholder={t("fx.fromDate")}
        value={filters.from_date || ""}
        onChange={(val) =>
          onFiltersChange({ ...filters, from_date: val || undefined, page: 1 })
        }
      />

      <DatePicker
        className="w-full sm:w-[180px]"
        placeholder={t("fx.toDate")}
        value={filters.to_date || ""}
        onChange={(val) =>
          onFiltersChange({ ...filters, to_date: val || undefined, page: 1 })
        }
      />

      <div className="flex items-center gap-2">
        <Checkbox
          id="show_cancelled"
          checked={filters.exclude_cancelled === false}
          onCheckedChange={(checked) =>
            onFiltersChange({
              ...filters,
              exclude_cancelled: checked ? false : undefined,
              page: 1,
            })
          }
        />
        <Label htmlFor="show_cancelled" className="text-sm cursor-pointer">
          {t("fx.showCancelled")}
        </Label>
      </div>

      {hasActiveFilters && (
        <Button variant="ghost" size="sm" onClick={clearFilters}>
          <IconX className="mr-1 size-4" />
          {t("common.cancel")}
        </Button>
      )}
    </div>
  );
}
