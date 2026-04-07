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
import { useFxDeal, useUpdateFxDeal } from "@/hooks/use-fx";
import { useCounterparties } from "@/hooks/use-counterparties";
import { FxDealForm } from "../../components/fx-deal-form";

export default function FxEditPage() {
  const { t } = useTranslation();
  const router = useRouter();
  const params = useParams();
  const id = params.id as string;
  const counterparties = useCounterparties();

  const { data, isLoading, isError, error } = useFxDeal(id);
  const updateMutation = useUpdateFxDeal(id);
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
            <Button variant="ghost" onClick={() => router.push("/fx")}>
              <IconArrowLeft className="mr-2 size-4" />
              {t("fx.backToList")}
            </Button>
          </div>
          <Alert variant="destructive">
            <IconAlertCircle className="size-4" />
            <AlertDescription>
              {t("fx.loadError")}: {extractErrorMessage(error)}
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
            <Button variant="ghost" onClick={() => router.push(`/fx/${id}`)}>
              <IconArrowLeft className="mr-2 size-4" />
              {t("common.back")}
            </Button>
          </div>
          <Alert>
            <IconAlertCircle className="size-4" />
            <AlertDescription>
              {t("fx.status." + deal.status)} — {t("fx.edit")}
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
          <Button variant="ghost" size="icon" onClick={() => router.push(`/fx/${id}`)}>
            <IconArrowLeft className="size-4" />
          </Button>
          <div>
            <h1 className="text-2xl font-bold tracking-tight">
              {t("fx.edit")} — {deal.ticket_number}
            </h1>
            <p className="text-muted-foreground">{t("fx.description")}</p>
          </div>
        </div>

        <FxDealForm
          initialData={deal}
          counterparties={counterparties}
          onSubmit={(data) => {
            setApiError(null);
            updateMutation.mutate(
              { ...data, version: deal.version },
              {
                onSuccess: () => {
                  toast.success(t("fx.updateSuccess"));
                  router.push(`/fx/${id}`);
                },
                onError: (err) => {
                  setApiError(
                    extractErrorMessage(err)
                  );
                },
              }
            );
          }}
          onCancel={() => router.push(`/fx/${id}`)}
          isSubmitting={updateMutation.isPending}
          error={apiError}
          submitLabel={t("fx.action.save")}
        />
      </div>
    </>
  );
}
