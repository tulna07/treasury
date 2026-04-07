"use client";

import { useState } from "react";
import { Header } from "@/components/header";
import { useTranslation } from "@/lib/i18n";
import { useAbility } from "@/hooks/use-ability";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Input } from "@/components/ui/input";
import { IconSearch } from "@tabler/icons-react";
import { useCreditLimits } from "@/hooks/use-limits";
import type { CreditLimit, LimitFilters } from "@/hooks/use-limits";
import { LimitCounterpartyList } from "./components/limit-counterparty-list";
import { LimitEditDialog } from "./components/limit-edit-dialog";
import { DailySummaryTable } from "./components/daily-summary-table";
import { LimitApprovalList } from "./components/limit-approval-list";

export default function LimitsPage() {
  const { t } = useTranslation();
  const ability = useAbility();
  const [filters, setFilters] = useState<LimitFilters>({
    page: 1,
    page_size: 20,
  });
  const [editLimit, setEditLimit] = useState<CreditLimit | null>(null);
  const [editOpen, setEditOpen] = useState(false);

  const { data, isLoading, isError, error, refetch } =
    useCreditLimits(filters);

  const limits = data?.data ?? [];
  const total = data?.total ?? 0;
  const totalPages = data?.total_pages ?? 0;

  function handleEdit(limit: CreditLimit) {
    setEditLimit(limit);
    setEditOpen(true);
  }

  return (
    <>
      <Header />
      <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
        <div className="flex flex-col gap-4 pt-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight">
              {t("limit.title")}
            </h1>
            <p className="text-muted-foreground">{t("limit.description")}</p>
          </div>
        </div>

        <Tabs defaultValue="counterparties">
          <TabsList>
            <TabsTrigger value="counterparties">
              {t("limit.tabCounterparties")}
            </TabsTrigger>
            <TabsTrigger value="daily-summary">
              {t("limit.tabDailySummary")}
            </TabsTrigger>
            <TabsTrigger value="approvals">
              {t("limit.tabApprovals")}
            </TabsTrigger>
          </TabsList>

          <TabsContent value="counterparties">
            <div className="space-y-4">
              {/* Search */}
              <div className="relative max-w-sm">
                <IconSearch className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  placeholder={t("limit.searchPlaceholder")}
                  className="pl-9"
                  value={filters.search ?? ""}
                  onChange={(e) =>
                    setFilters({
                      ...filters,
                      search: e.target.value,
                      page: 1,
                    })
                  }
                />
              </div>

              <LimitCounterpartyList
                limits={limits}
                total={total}
                totalPages={totalPages}
                filters={filters}
                onFiltersChange={setFilters}
                onEdit={handleEdit}
                isLoading={isLoading}
                isError={isError}
                error={error}
                onRetry={() => refetch()}
              />
            </div>
          </TabsContent>

          <TabsContent value="daily-summary">
            <DailySummaryTable />
          </TabsContent>

          <TabsContent value="approvals">
            <LimitApprovalList />
          </TabsContent>
        </Tabs>
      </div>

      <LimitEditDialog
        open={editOpen}
        onOpenChange={setEditOpen}
        limit={editLimit}
      />
    </>
  );
}
