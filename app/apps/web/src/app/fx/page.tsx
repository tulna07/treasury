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
import { useFxDeals, useCloneFxDeal, useDeleteFxDeal } from "@/hooks/use-fx";
import type { FxDeal, FxFilters } from "@/hooks/use-fx";
import { FxFiltersBar } from "./components/fx-filters";
import { FxDealList } from "./components/fx-deal-list";
import { FxExportDialog } from "./components/fx-export-dialog";

export default function FxPage() {
  const { t } = useTranslation();
  const ability = useAbility();
  const router = useRouter();
  const [filters, setFilters] = useState<FxFilters>({
    page: 1,
    page_size: 20,
  });

  const { data, isLoading, isError, error, refetch } = useFxDeals(filters);
  const cloneMutation = useCloneFxDeal();
  const deleteMutation = useDeleteFxDeal();

  const deals = data?.data ?? [];
  const total = data?.total ?? 0;
  const totalPages = data?.total_pages ?? 0;

  function handleView(deal: FxDeal) {
    router.push(`/fx/${deal.id}`);
  }

  function handleEdit(deal: FxDeal) {
    router.push(`/fx/${deal.id}/edit`);
  }

  function handleClone(deal: FxDeal) {
    cloneMutation.mutate(deal.id, {
      onSuccess: () => toast.success(t("fx.cloneSuccess")),
      onError: (err) =>
        toast.error(extractErrorMessage(err)),
    });
  }

  function handleDelete(deal: FxDeal) {
    deleteMutation.mutate(deal.id, {
      onSuccess: () => toast.success(t("fx.deleteSuccess")),
      onError: (err) =>
        toast.error(extractErrorMessage(err)),
    });
  }

  return (
    <>
      <Header />
      <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
        <div className="flex flex-col gap-4 pt-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight">{t("fx.title")}</h1>
            <p className="text-muted-foreground">{t("fx.description")}</p>
          </div>
          <div className="flex gap-2">
            {ability.can("create", "FXTransaction") && (
              <Button onClick={() => router.push("/fx/new")} className="shrink-0">
                <IconPlus className="mr-2 size-4" />
                {t("fx.new")}
              </Button>
            )}
            <FxExportDialog />
          </div>
        </div>

        <FxFiltersBar filters={filters} onFiltersChange={setFilters} />

        <FxDealList
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
