"use client";

import { useRouter } from "next/navigation";
import { Header } from "@/components/header";
import { useTranslation } from "@/lib/i18n";
import { Button } from "@/components/ui/button";
import { IconArrowLeft } from "@tabler/icons-react";
import { toast } from "sonner";
import { extractErrorMessage } from "@/lib/utils";
import { useCreateFxDeal } from "@/hooks/use-fx";
import { useCounterparties } from "@/hooks/use-counterparties";
import { FxDealForm } from "../components/fx-deal-form";
import { useState } from "react";

export default function FxNewPage() {
  const { t } = useTranslation();
  const router = useRouter();
  const counterparties = useCounterparties();
  const createMutation = useCreateFxDeal();
  const [apiError, setApiError] = useState<string | null>(null);

  return (
    <>
      <Header />
      <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
        <div className="flex items-center gap-4 pt-4">
          <Button variant="ghost" size="icon" onClick={() => router.push("/fx")}>
            <IconArrowLeft className="size-4" />
          </Button>
          <div>
            <h1 className="text-2xl font-bold tracking-tight">{t("fx.new")}</h1>
            <p className="text-muted-foreground">{t("fx.description")}</p>
          </div>
        </div>

        <FxDealForm
          counterparties={counterparties}
          onSubmit={(data) => {
            setApiError(null);
            createMutation.mutate(data, {
              onSuccess: (res) => {
                toast.success(t("fx.createSuccess"));
                router.push(`/fx/${res.id}`);
              },
              onError: (err) => {
                setApiError(
                  extractErrorMessage(err)
                );
              },
            });
          }}
          onCancel={() => router.push("/fx")}
          isSubmitting={createMutation.isPending}
          error={apiError}
          submitLabel={t("fx.action.create")}
        />
      </div>
    </>
  );
}
