"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Header } from "@/components/header";
import { useTranslation } from "@/lib/i18n";
import { Button } from "@/components/ui/button";
import { IconPlus } from "@tabler/icons-react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { MMInterbankList } from "./components/mm-interbank-list";
import { MMOMOList } from "./components/mm-omo-list";
import { MMRepoList } from "./components/mm-repo-list";
import { MMExportDialog } from "./components/mm-export-dialog";
import { useMMInterbankDeals, useMMOMODeals, useMMRepoDeals, type MMFilters } from "@/hooks/use-mm";

export default function MmPage() {
  const { t } = useTranslation();
  const router = useRouter();
  const [activeTab, setActiveTab] = useState("interbank");
  const [filters, setFilters] = useState<MMFilters>({
    page: 1,
    page_size: 20,
  });
  const [omoFilters, setOmoFilters] = useState<MMFilters>({
    page: 1,
    page_size: 20,
  });
  const [repoFilters, setRepoFilters] = useState<MMFilters>({
    page: 1,
    page_size: 20,
  });

  const { data, isLoading, isError, error, refetch } = useMMInterbankDeals(filters);
  const omoQuery = useMMOMODeals(omoFilters);
  const repoQuery = useMMRepoDeals(repoFilters);

  return (
    <>
      <Header />
      <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
        <div className="flex flex-col gap-4 pt-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight">{t("mm.title")}</h1>
            <p className="text-muted-foreground">{t("mm.description")}</p>
          </div>
          <div className="flex gap-2">
            <Button onClick={() => {
              const routes: Record<string, string> = {
                interbank: "/mm/interbank/new",
                omo: "/mm/omo/new",
                repo: "/mm/repo/new",
              };
              router.push(routes[activeTab] || "/mm/interbank/new");
            }}>
              <IconPlus className="mr-2 size-4" />
              {t("mm.new")}
            </Button>
            <MMExportDialog />
          </div>
        </div>

        <Tabs value={activeTab} onValueChange={setActiveTab}>
          <TabsList>
            <TabsTrigger value="interbank">{t("mm.interbank")}</TabsTrigger>
            <TabsTrigger value="omo">{t("mm.omo")}</TabsTrigger>
            <TabsTrigger value="repo">{t("mm.repoKbnn")}</TabsTrigger>
          </TabsList>

          <TabsContent value="interbank">
            <MMInterbankList
              deals={data?.data ?? []}
              total={data?.total ?? 0}
              totalPages={data?.total_pages ?? 0}
              filters={filters}
              onFiltersChange={setFilters}
              isLoading={isLoading}
              isError={isError}
              error={error}
              onRetry={refetch}
            />
          </TabsContent>

          <TabsContent value="omo">
            <MMOMOList
              deals={omoQuery.data?.data ?? []}
              total={omoQuery.data?.total ?? 0}
              totalPages={omoQuery.data?.total_pages ?? 0}
              filters={omoFilters}
              onFiltersChange={setOmoFilters}
              isLoading={omoQuery.isLoading}
              isError={omoQuery.isError}
              error={omoQuery.error}
              onRetry={omoQuery.refetch}
            />
          </TabsContent>

          <TabsContent value="repo">
            <MMRepoList
              deals={repoQuery.data?.data ?? []}
              total={repoQuery.data?.total ?? 0}
              totalPages={repoQuery.data?.total_pages ?? 0}
              filters={repoFilters}
              onFiltersChange={setRepoFilters}
              isLoading={repoQuery.isLoading}
              isError={repoQuery.isError}
              error={repoQuery.error}
              onRetry={repoQuery.refetch}
            />
          </TabsContent>
        </Tabs>
      </div>
    </>
  );
}
