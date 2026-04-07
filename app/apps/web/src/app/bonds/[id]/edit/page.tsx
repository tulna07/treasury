"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { Header } from "@/components/header";
import { useTranslation } from "@/lib/i18n";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { IconArrowLeft, IconAlertCircle } from "@tabler/icons-react";
import { toast } from "sonner";
import { extractErrorMessage } from "@/lib/utils";
import { useBondDeal, useUpdateBondDeal } from "@/hooks/use-bonds";
import { useCounterparties } from "@/hooks/use-counterparties";
import { BondDealForm } from "../../components/bond-deal-form";

export default function BondEditPage() {
  const { t } = useTranslation();
  const router = useRouter();
  const params = useParams();
  const id = params.id as string;
  const counterparties = useCounterparties();

  const { data, isLoading, isError, error } = useBondDeal(id);
  const updateMutation = useUpdateBondDeal(id);
  const [apiError, setApiError] = useState<string | null>(null);

  const deal = data;

  if (isLoading) {
    return (
      <>
        <Header />
        <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
          <div className="pt-4 space-y-4">
            <Skeleton className="h-8 w-64" />
            <Skeleton className="h-48 w-full" />
          </div>
        </div>
      </>
    );
  }

  if (isError || !deal) {
    return (
      <>
        <Header />
        <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
          <div className="pt-4">
            <Button variant="ghost" onClick={() => router.push("/bonds")}>
              <IconArrowLeft className="mr-2 size-4" />
              {t("bond.backToList")}
            </Button>
          </div>
          <Alert variant="destructive">
            <IconAlertCircle className="size-4" />
            <AlertDescription>
              {t("bond.loadError")}: {extractErrorMessage(error)}
            </AlertDescription>
          </Alert>
        </div>
      </>
    );
  }

  if (deal.status !== "OPEN") {
    return (
      <>
        <Header />
        <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
          <div className="pt-4">
            <Button variant="ghost" onClick={() => router.push(`/bonds/${id}`)}>
              <IconArrowLeft className="mr-2 size-4" />
              {t("common.back")}
            </Button>
          </div>
          <Alert>
            <IconAlertCircle className="size-4" />
            <AlertDescription>
              {t("bond.status." + deal.status)} — {t("bond.edit")}
            </AlertDescription>
          </Alert>
        </div>
      </>
    );
  }

  return (
    <>
      <Header />
      <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
        <div className="flex items-center gap-4 pt-4">
          <Button variant="ghost" size="icon" onClick={() => router.push(`/bonds/${id}`)}>
            <IconArrowLeft className="size-4" />
          </Button>
          <div>
            <h1 className="text-2xl font-bold tracking-tight">
              {t("bond.edit")} — {deal.deal_number}
            </h1>
            <p className="text-muted-foreground">{t("bond.description")}</p>
          </div>
        </div>

        <BondDealForm
          initialData={deal}
          counterparties={counterparties}
          onSubmit={(data) => {
            setApiError(null);
            updateMutation.mutate(
              { ...data, version: deal.version },
              {
                onSuccess: () => {
                  toast.success(t("bond.updateSuccess"));
                  router.push(`/bonds/${id}`);
                },
                onError: (err) => {
                  setApiError(extractErrorMessage(err));
                },
              }
            );
          }}
          onCancel={() => router.push(`/bonds/${id}`)}
          isSubmitting={updateMutation.isPending}
          error={apiError}
          submitLabel={t("bond.action.edit")}
        />
      </div>
    </>
  );
}
