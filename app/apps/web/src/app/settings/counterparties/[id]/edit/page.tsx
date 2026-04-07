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
import { useMasterCounterparty, useUpdateCounterparty } from "@/hooks/use-master-data";
import { CounterpartyForm } from "../../components/counterparty-form";

export default function CounterpartyEditPage() {
  const { t } = useTranslation();
  const router = useRouter();
  const params = useParams();
  const id = params.id as string;

  const { data: cp, isLoading, isError, error } = useMasterCounterparty(id);
  const updateMutation = useUpdateCounterparty(id);
  const [apiError, setApiError] = useState<string | null>(null);

  if (isLoading) {
    return (
      <>
        <Header />
        <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
          <div className="pt-4 space-y-4">
            <Skeleton className="h-8 w-64" />
            <Skeleton className="h-64 w-full" />
          </div>
        </div>
      </>
    );
  }

  if (isError || !cp) {
    return (
      <>
        <Header />
        <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
          <div className="pt-4">
            <Button variant="ghost" onClick={() => router.push("/settings/counterparties")}>
              <IconArrowLeft className="mr-2 size-4" />
              {t("counterparties.backToList")}
            </Button>
          </div>
          <Alert variant="destructive">
            <IconAlertCircle className="size-4" />
            <AlertDescription>
              {t("counterparties.loadError")}: {extractErrorMessage(error)}
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
          <Button variant="ghost" size="icon" onClick={() => router.push("/settings/counterparties")}>
            <IconArrowLeft className="size-4" />
          </Button>
          <div>
            <h1 className="text-2xl font-bold tracking-tight">{t("counterparties.edit")}</h1>
            <p className="text-muted-foreground">{cp.full_name}</p>
          </div>
        </div>

        <CounterpartyForm
          initialData={{
            code: cp.code,
            full_name: cp.full_name,
            short_name: cp.short_name || "",
            swift_code: cp.swift_code || "",
            country_code: cp.country_code || "",
            cif: cp.cif || "",
          }}
          onSubmit={(data) => {
            setApiError(null);
            updateMutation.mutate({
              full_name: data.full_name,
              short_name: data.short_name || undefined,
              swift_code: data.swift_code || undefined,
              country_code: data.country_code || undefined,
            }, {
              onSuccess: () => {
                toast.success(t("counterparties.updateSuccess"));
                router.push("/settings/counterparties");
              },
              onError: (err) => {
                setApiError(extractErrorMessage(err));
              },
            });
          }}
          onCancel={() => router.push("/settings/counterparties")}
          isSubmitting={updateMutation.isPending}
          error={apiError}
          submitLabel={t("common.save")}
        />
      </div>
    </>
  );
}
