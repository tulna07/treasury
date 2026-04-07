"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Header } from "@/components/header";
import { useTranslation } from "@/lib/i18n";
import { Button } from "@/components/ui/button";
import { IconArrowLeft } from "@tabler/icons-react";
import { toast } from "sonner";
import { extractErrorMessage } from "@/lib/utils";
import { useCreateCounterparty } from "@/hooks/use-master-data";
import { CounterpartyForm } from "../components/counterparty-form";

export default function CounterpartyNewPage() {
  const { t } = useTranslation();
  const router = useRouter();
  const createMutation = useCreateCounterparty();
  const [apiError, setApiError] = useState<string | null>(null);

  return (
    <>
      <Header />
      <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
        <div className="flex items-center gap-4 pt-4">
          <Button variant="ghost" size="icon" onClick={() => router.push("/settings/counterparties")}>
            <IconArrowLeft className="size-4" />
          </Button>
          <div>
            <h1 className="text-2xl font-bold tracking-tight">{t("counterparties.new")}</h1>
            <p className="text-muted-foreground">{t("counterparties.description")}</p>
          </div>
        </div>

        <CounterpartyForm
          onSubmit={(data) => {
            setApiError(null);
            createMutation.mutate(data, {
              onSuccess: () => {
                toast.success(t("counterparties.createSuccess"));
                router.push("/settings/counterparties");
              },
              onError: (err) => {
                setApiError(extractErrorMessage(err));
              },
            });
          }}
          onCancel={() => router.push("/settings/counterparties")}
          isSubmitting={createMutation.isPending}
          error={apiError}
          submitLabel={t("counterparties.new")}
        />
      </div>
    </>
  );
}
