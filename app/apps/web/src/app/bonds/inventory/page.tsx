"use client";

import { useState } from "react";
import { Header } from "@/components/header";
import { useTranslation } from "@/lib/i18n";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { IconAlertCircle, IconPackage } from "@tabler/icons-react";
import { useBondInventory } from "@/hooks/use-bonds";
import { formatDate, extractErrorMessage } from "@/lib/utils";

export default function BondInventoryPage() {
  const { t } = useTranslation();
  const { data: inventory, isLoading, isError, error } = useBondInventory();
  const [categoryFilter, setCategoryFilter] = useState<string>("ALL");

  const filtered = inventory?.filter(
    (item: any) => categoryFilter === "ALL" || item.portfolio_type === categoryFilter
  ) ?? [];

  function formatAmount(value: string | number) {
    return Number(value).toLocaleString("en-US", {
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    });
  }

  return (
    <>
      <Header />
      <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
        <div className="flex flex-col gap-4 pt-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight">{t("bond.inventory.title")}</h1>
            <p className="text-muted-foreground">{t("bond.inventory.description")}</p>
          </div>
          <Select value={categoryFilter} onValueChange={(val) => setCategoryFilter(val ?? "ALL")}>
            <SelectTrigger className="w-full sm:w-48">
              <SelectValue>{(v: string) => v === "ALL" ? t("bond.allCategories") : v}</SelectValue>
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="ALL" label={t("bond.allCategories")}>{t("bond.allCategories")}</SelectItem>
              <SelectItem value="HTM" label="HTM">HTM</SelectItem>
              <SelectItem value="AFS" label="AFS">AFS</SelectItem>
              <SelectItem value="HFT" label="HFT">HFT</SelectItem>
            </SelectContent>
          </Select>
        </div>

        {isLoading && (
          <div className="space-y-2">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} className="h-12 w-full" />
            ))}
          </div>
        )}

        {isError && (
          <Alert variant="destructive">
            <IconAlertCircle className="size-4" />
            <AlertDescription>{extractErrorMessage(error)}</AlertDescription>
          </Alert>
        )}

        {!isLoading && !isError && (
          <>
            {/* Desktop table */}
            <div className="hidden md:block">
              <Card>
                <CardContent className="p-0">
                  <div className="rounded-lg border overflow-x-auto">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>{t("bond.bondCode")}</TableHead>
                          <TableHead>{t("bond.category")}</TableHead>
                          <TableHead>{t("bond.portfolioType")}</TableHead>
                          <TableHead>{t("bond.issuer")}</TableHead>
                          <TableHead className="text-right">{t("bond.quantity")}</TableHead>
                          <TableHead className="text-right">{t("bond.faceValue")}</TableHead>
                          <TableHead className="text-right">{t("bond.totalValue")}</TableHead>
                          <TableHead>{t("bond.maturityDate")}</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {filtered.length === 0 ? (
                          <TableRow>
                            <TableCell colSpan={8} className="h-32 text-center text-muted-foreground">
                              <IconPackage className="mx-auto size-8 mb-2" />
                              {t("bond.inventory.empty")}
                            </TableCell>
                          </TableRow>
                        ) : (
                          filtered.map((item: any, idx: number) => (
                            <TableRow key={`${item.bond_code}-${item.portfolio_type}-${idx}`}>
                              <TableCell className="font-mono font-medium">{item.bond_code}</TableCell>
                              <TableCell>{t(`bond.category.${item.bond_category}`)}</TableCell>
                              <TableCell>{item.portfolio_type}</TableCell>
                              <TableCell>{item.catalog_issuer || "—"}</TableCell>
                              <TableCell className="text-right tabular-nums">{(item.available_quantity ?? 0).toLocaleString("en-US")}</TableCell>
                              <TableCell className="text-right tabular-nums">{formatAmount(item.catalog_face_value || item.acquisition_price || 0)}</TableCell>
                              <TableCell className="text-right tabular-nums font-medium">{formatAmount(item.nominal_value || 0)}</TableCell>
                              <TableCell>{item.catalog_maturity_date ? formatDate(item.catalog_maturity_date) : "—"}</TableCell>
                            </TableRow>
                          ))
                        )}
                      </TableBody>
                    </Table>
                  </div>
                </CardContent>
              </Card>
            </div>

            {/* Mobile cards */}
            <div className="md:hidden space-y-3">
              {filtered.length === 0 ? (
                <div className="text-center py-8 text-muted-foreground">
                  <IconPackage className="mx-auto size-8 mb-2" />
                  {t("bond.inventory.empty")}
                </div>
              ) : (
                filtered.map((item: any, idx: number) => (
                  <Card key={`${item.bond_code}-${item.portfolio_type}-${idx}`}>
                    <CardHeader className="pb-2">
                      <CardTitle className="text-sm font-mono">{item.bond_code}</CardTitle>
                    </CardHeader>
                    <CardContent className="text-sm space-y-1">
                      <div className="text-muted-foreground">
                        {t(`bond.category.${item.bond_category}`)} · {item.portfolio_type} · {item.catalog_issuer || "—"}
                      </div>
                      <div className="flex justify-between">
                        <span className="text-muted-foreground">{t("bond.quantity")}</span>
                        <span className="tabular-nums font-medium">{(item.available_quantity ?? 0).toLocaleString("en-US")}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-muted-foreground">{t("bond.totalValue")}</span>
                        <span className="tabular-nums font-medium">{formatAmount(item.nominal_value || 0)}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-muted-foreground">{t("bond.maturityDate")}</span>
                        <span>{item.catalog_maturity_date ? formatDate(item.catalog_maturity_date) : "—"}</span>
                      </div>
                    </CardContent>
                  </Card>
                ))
              )}
            </div>
          </>
        )}
      </div>
    </>
  );
}
