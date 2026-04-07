"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Header } from "@/components/header";
import { useTranslation } from "@/lib/i18n";
import { useAbility } from "@/hooks/use-ability";
import { IconPlus } from "@tabler/icons-react";
import { Button } from "@/components/ui/button";
import { toast } from "sonner";
import { extractErrorMessage } from "@/lib/utils";
import { useBondDeals, useCloneBondDeal, useDeleteBondDeal } from "@/hooks/use-bonds";
import type { BondDeal, BondFilters } from "@/hooks/use-bonds";
import { BondFiltersBar } from "./components/bond-filters";
import { BondDealList } from "./components/bond-deal-list";
import { BondExportDialog } from "./components/bond-export-dialog";

export default function BondsPage() {
  const { t } = useTranslation();
  const ability = useAbility();
  const router = useRouter();
  const [filters, setFilters] = useState<BondFilters>({
    page: 1,
    page_size: 20,
  });

  const { data, isLoading, isError, error, refetch } = useBondDeals(filters);
  const cloneMutation = useCloneBondDeal();
  const deleteMutation = useDeleteBondDeal();

  const deals = data?.data ?? [];
  const total = data?.total ?? 0;
  const totalPages = data?.total_pages ?? 0;

  function handleView(deal: BondDeal) {
    router.push(`/bonds/${deal.id}`);
  }

  function handleEdit(deal: BondDeal) {
    router.push(`/bonds/${deal.id}/edit`);
  }

  function handleClone(deal: BondDeal) {
    cloneMutation.mutate(deal.id, {
      onSuccess: () => toast.success(t("bond.cloneSuccess")),
      onError: (err) => toast.error(extractErrorMessage(err)),
    });
  }

  function handleDelete(deal: BondDeal) {
    deleteMutation.mutate(deal.id, {
      onSuccess: () => toast.success(t("bond.deleteSuccess")),
      onError: (err) => toast.error(extractErrorMessage(err)),
    });
  }

  return (
    <>
      <Header />
      <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
        <div className="flex flex-col gap-4 pt-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight">{t("bond.title")}</h1>
            <p className="text-muted-foreground">{t("bond.description")}</p>
          </div>
          <div className="flex gap-2">
            <BondExportDialog />
            {ability.can("create", "GTCGTransaction") && (
              <Button onClick={() => router.push("/bonds/new")} className="shrink-0">
                <IconPlus className="mr-2 size-4" />
                {t("bond.new")}
              </Button>
            )}
          </div>
        </div>

        <BondFiltersBar filters={filters} onFiltersChange={setFilters} />

        <BondDealList
          deals={deals}
          total={total}
          totalPages={totalPages}
          filters={filters}
          onFiltersChange={setFilters}
          onView={handleView}
          onEdit={handleEdit}
          onClone={handleClone}
          onDelete={handleDelete}
          isLoading={isLoading}
          isError={isError}
          error={error}
          onRetry={() => refetch()}
        />
      </div>
    </>
  );
}
