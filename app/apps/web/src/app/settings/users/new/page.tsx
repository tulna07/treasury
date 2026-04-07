"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Header } from "@/components/header";
import { useTranslation } from "@/lib/i18n";
import { Button } from "@/components/ui/button";
import { IconArrowLeft } from "@tabler/icons-react";
import { toast } from "sonner";
import { extractErrorMessage } from "@/lib/utils";
import { useCreateAdminUser } from "@/hooks/use-admin-users";
import { UserForm } from "../components/user-form";

export default function UserNewPage() {
  const { t } = useTranslation();
  const router = useRouter();
  const createMutation = useCreateAdminUser();
  const [apiError, setApiError] = useState<string | null>(null);

  return (
    <>
      <Header />
      <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
        <div className="flex items-center gap-4 pt-4">
          <Button variant="ghost" size="icon" onClick={() => router.push("/settings/users")}>
            <IconArrowLeft className="size-4" />
          </Button>
          <div>
            <h1 className="text-2xl font-bold tracking-tight">{t("users.new")}</h1>
            <p className="text-muted-foreground">{t("users.description")}</p>
          </div>
        </div>

        <UserForm
          isCreate
          onSubmit={(data) => {
            setApiError(null);
            createMutation.mutate(
              {
                username: data.username!,
                password: data.password!,
                full_name: data.full_name,
                email: data.email,
                branch_id: data.branch_id,
                department: data.department,
                position: data.position,
              },
              {
                onSuccess: (res) => {
                  toast.success(t("users.createSuccess"));
                  router.push(`/settings/users/${res.id}`);
                },
                onError: (err) => {
                  setApiError(extractErrorMessage(err));
                },
              }
            );
          }}
          onCancel={() => router.push("/settings/users")}
          isSubmitting={createMutation.isPending}
          error={apiError}
          submitLabel={t("users.new")}
        />
      </div>
    </>
  );
}
