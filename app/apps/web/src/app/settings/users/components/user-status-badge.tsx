"use client";

import { Badge } from "@/components/ui/badge";
import { useTranslation } from "@/lib/i18n";

export function UserStatusBadge({ isActive }: { isActive: boolean }) {
  const { t } = useTranslation();

  return isActive ? (
    <Badge variant="outline" className="border-green-500/50 bg-green-500/10 text-green-700 dark:text-green-400">
      {t("common.active")}
    </Badge>
  ) : (
    <Badge variant="outline" className="border-red-500/50 bg-red-500/10 text-red-700 dark:text-red-400">
      {t("common.locked")}
    </Badge>
  );
}
