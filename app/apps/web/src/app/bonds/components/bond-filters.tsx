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
import type { BondFilters } from "@/hooks/use-bonds";

interface BondFiltersBarProps {
  filters: BondFilters;
  onFiltersChange: (filters: BondFilters) => void;
}

const BOND_STATUSES = [
  "OPEN",
  "PENDING_L2_APPROVAL",
  "PENDING_BOOKING",
  "PENDING_CHIEF_ACCOUNTANT",
  "COMPLETED",
  "REJECTED",
  "VOIDED_BY_ACCOUNTING",
  "CANCELLED",
] as const;

const BOND_CATEGORIES = [
  "GOVERNMENT",
  "FINANCIAL_INSTITUTION",
  "CERTIFICATE_OF_DEPOSIT",
] as const;

const BOND_DIRECTIONS = ["BUY", "SELL"] as const;

export function BondFiltersBar({ filters, onFiltersChange }: BondFiltersBarProps) {
  const { t } = useTranslation();

  const renderStatus = (v: string) => v === "ALL" ? t("bond.allStatuses") : t(`bond.status.${v}`);
  const renderCategory = (v: string) => v === "ALL" ? t("bond.allCategories") : t(`bond.category.${v}`);
  const renderDirection = (v: string) => v === "ALL" ? t("bond.allDirections") : t(`bond.direction.${v}`);

  const hasActiveFilters =
    filters.status ||
    filters.bond_category ||
    filters.direction ||
    filters.deal_number ||
    filters.from_date ||
    filters.to_date;

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
          placeholder={t("bond.search")}
          value={filters.deal_number || ""}
          onChange={(e) =>
            onFiltersChange({ ...filters, deal_number: e.target.value, page: 1 })
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
          <SelectValue placeholder={t("bond.allStatuses")}>
            {(v: string) => v === "__ALL__" ? t("bond.allStatuses") : t(`bond.status.${v}`)}
          </SelectValue>
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="__ALL__" label={t("bond.allStatuses")}>{t("bond.allStatuses")}</SelectItem>
          {BOND_STATUSES.map((s) => (
            <SelectItem key={s} value={s} label={t(`bond.status.${s}`)}>
              {t(`bond.status.${s}`)}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      <Select
        value={filters.bond_category || "__ALL__"}
        onValueChange={(v) =>
          onFiltersChange({ ...filters, bond_category: v === "__ALL__" || !v ? undefined : v, page: 1 })
        }
      >
        <SelectTrigger className="w-full sm:w-auto">
          <SelectValue placeholder={t("bond.allCategories")}>
            {(v: string) => v === "__ALL__" ? t("bond.allCategories") : t(`bond.category.${v}`)}
          </SelectValue>
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="__ALL__" label={t("bond.allCategories")}>{t("bond.allCategories")}</SelectItem>
          {BOND_CATEGORIES.map((c) => (
            <SelectItem key={c} value={c} label={t(`bond.category.${c}`)}>
              {t(`bond.category.${c}`)}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      <Select
        value={filters.direction || "__ALL__"}
        onValueChange={(v) =>
          onFiltersChange({ ...filters, direction: v === "__ALL__" || !v ? undefined : v, page: 1 })
        }
      >
        <SelectTrigger className="w-full sm:w-auto">
          <SelectValue placeholder={t("bond.allDirections")}>
            {(v: string) => v === "__ALL__" ? t("bond.allDirections") : t(`bond.direction.${v}`)}
          </SelectValue>
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="__ALL__" label={t("bond.allDirections")}>{t("bond.allDirections")}</SelectItem>
          {BOND_DIRECTIONS.map((d) => (
            <SelectItem key={d} value={d} label={t(`bond.direction.${d}`)}>
              {t(`bond.direction.${d}`)}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      <DatePicker
        className="w-full sm:w-[180px]"
        placeholder={t("bond.fromDate")}
        value={filters.from_date || ""}
        onChange={(val) =>
          onFiltersChange({ ...filters, from_date: val || undefined, page: 1 })
        }
      />

      <DatePicker
        className="w-full sm:w-[180px]"
        placeholder={t("bond.toDate")}
        value={filters.to_date || ""}
        onChange={(val) =>
          onFiltersChange({ ...filters, to_date: val || undefined, page: 1 })
        }
      />

      <div className="flex items-center gap-2">
        <Checkbox
          id="show_cancelled_bonds"
          checked={filters.exclude_cancelled === false}
          onCheckedChange={(checked) =>
            onFiltersChange({
              ...filters,
              exclude_cancelled: checked ? false : undefined,
              page: 1,
            })
          }
        />
        <Label htmlFor="show_cancelled_bonds" className="text-sm cursor-pointer">
          {t("bond.showCancelled")}
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
