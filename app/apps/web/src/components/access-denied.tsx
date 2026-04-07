"use client";

import { useTranslation } from "@/lib/i18n";
import { IconShieldLock, IconArrowLeft } from "@tabler/icons-react";
import { Button } from "@/components/ui/button";
import { useRouter } from "next/navigation";

interface AccessDeniedProps {
  /** Optional custom message override */
  message?: string;
}

export function AccessDenied({ message }: AccessDeniedProps) {
  const { t } = useTranslation();
  const router = useRouter();

  return (
    <div className="flex flex-1 items-center justify-center p-8">
      <div className="mx-auto max-w-md text-center">
        <div className="mx-auto mb-6 flex size-20 items-center justify-center rounded-full bg-muted">
          <IconShieldLock className="size-10 text-muted-foreground" />
        </div>
        <h2 className="mb-2 text-2xl font-semibold tracking-tight">
          {t("access.denied.title")}
        </h2>
        <p className="mb-6 text-muted-foreground leading-relaxed">
          {message || t("access.denied.description")}
        </p>
        <div className="flex flex-col gap-3 sm:flex-row sm:justify-center">
          <Button variant="outline" onClick={() => router.back()}>
            <IconArrowLeft className="mr-2 size-4" />
            {t("access.denied.goBack")}
          </Button>
          <Button onClick={() => router.push("/")}>
            {t("access.denied.goHome")}
          </Button>
        </div>
        <p className="mt-8 text-xs text-muted-foreground">
          {t("access.denied.contact")}
        </p>
      </div>
    </div>
  );
}
