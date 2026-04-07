"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Header } from "@/components/header";
import { useTranslation } from "@/lib/i18n";
import { Button } from "@/components/ui/button";
import { IconArrowLeft } from "@tabler/icons-react";
import { toast } from "sonner";
import { extractErrorMessage } from "@/lib/utils";
import { useCreateBondDeal } from "@/hooks/use-bonds";
import { useCounterparties } from "@/hooks/use-counterparties";
import { BondDealForm } from "../components/bond-deal-form";

export default function BondNewPage() {
  const { t } = useTranslation();
  const router = useRouter();
  const counterparties = useCounterparties();
  const createMutation = useCreateBondDeal();
  const [apiError, setApiError] = useState<string | null>(null);

  return (
    <>
      <Header />
      <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
        <div className="flex items-center gap-4 pt-4">
          <Button variant="ghost" size="icon" onClick={() => router.push("/bonds")}>
            <IconArrowLeft className="size-4" />
          </Button>
          <div>
            <h1 className="text-2xl font-bold tracking-tight">{t("bond.new")}</h1>
            <p className="text-muted-foreground">{t("bond.description")}</p>
          </div>
        </div>

        <BondDealForm
          counterparties={counterparties}
          onSubmit={(data) => {
            setApiError(null);
            createMutation.mutate(data, {
              onSuccess: (res) => {
                toast.success(t("bond.createSuccess"));
                router.push(`/bonds/${res.id}`);
              },
              onError: (err) => {
                setApiError(extractErrorMessage(err));
              },
            });
          }}
          onCancel={() => router.push("/bonds")}
          isSubmitting={createMutation.isPending}
          error={apiError}
          submitLabel={t("bond.new")}
        />
      </div>
    </>
  );
}
