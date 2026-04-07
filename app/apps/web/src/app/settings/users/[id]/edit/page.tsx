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
import { useAdminUser, useUpdateAdminUser } from "@/hooks/use-admin-users";
import { UserForm } from "../../components/user-form";

export default function UserEditPage() {
  const { t } = useTranslation();
  const router = useRouter();
  const params = useParams();
  const id = params.id as string;

  const { data: user, isLoading, isError, error } = useAdminUser(id);
  const updateMutation = useUpdateAdminUser(id);
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

  if (isError || !user) {
    return (
      <>
        <Header />
        <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
          <div className="pt-4">
            <Button variant="ghost" onClick={() => router.push("/settings/users")}>
              <IconArrowLeft className="mr-2 size-4" />
              {t("users.backToList")}
            </Button>
          </div>
          <Alert variant="destructive">
            <IconAlertCircle className="size-4" />
            <AlertDescription>
              {t("users.loadError")}: {extractErrorMessage(error)}
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
          <Button variant="ghost" size="icon" onClick={() => router.push(`/settings/users/${id}`)}>
            <IconArrowLeft className="size-4" />
          </Button>
          <div>
            <h1 className="text-2xl font-bold tracking-tight">{t("users.edit")}</h1>
            <p className="text-muted-foreground">{user.full_name}</p>
          </div>
        </div>

        <UserForm
          initialData={{
            full_name: user.full_name,
            email: user.email,
            branch_id: user.branch_id,
            department: user.department,
            position: user.position,
          }}
          onSubmit={(data) => {
            setApiError(null);
            updateMutation.mutate(
              {
                full_name: data.full_name,
                email: data.email,
                department: data.department,
                position: data.position,
              },
              {
                onSuccess: () => {
                  toast.success(t("users.updateSuccess"));
                  router.push(`/settings/users/${id}`);
                },
                onError: (err) => {
                  setApiError(extractErrorMessage(err));
                },
              }
            );
          }}
          onCancel={() => router.push(`/settings/users/${id}`)}
          isSubmitting={updateMutation.isPending}
          error={apiError}
          submitLabel={t("common.save")}
        />
      </div>
    </>
  );
}
