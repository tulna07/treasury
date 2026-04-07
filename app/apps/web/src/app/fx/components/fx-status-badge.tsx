"use client";

import { Badge } from "@/components/ui/badge";
import { useTranslation } from "@/lib/i18n";
import type { FxStatus } from "@/hooks/use-fx";

const statusStyles: Record<string, string> = {
  OPEN: "bg-blue-100 text-blue-800 dark:bg-blue-900/50 dark:text-blue-300",
  PENDING_L1: "bg-amber-100 text-amber-800 dark:bg-amber-900/50 dark:text-amber-300",
  PENDING_L2: "bg-orange-100 text-orange-800 dark:bg-orange-900/50 dark:text-orange-300",
  PENDING_L2_APPROVAL: "bg-orange-100 text-orange-800 dark:bg-orange-900/50 dark:text-orange-300",
  PENDING_CHIEF_ACCOUNTANT: "bg-violet-100 text-violet-800 dark:bg-violet-900/50 dark:text-violet-300",
  PENDING_BOOKING: "bg-indigo-100 text-indigo-800 dark:bg-indigo-900/50 dark:text-indigo-300",
  APPROVED: "bg-emerald-100 text-emerald-800 dark:bg-emerald-900/50 dark:text-emerald-300",
  BOOKED_L1: "bg-teal-100 text-teal-800 dark:bg-teal-900/50 dark:text-teal-300",
  BOOKED_L2: "bg-cyan-100 text-cyan-800 dark:bg-cyan-900/50 dark:text-cyan-300",
  COMPLETED: "bg-green-100 text-green-800 dark:bg-green-900/50 dark:text-green-300",
  SETTLED: "bg-green-100 text-green-800 dark:bg-green-900/50 dark:text-green-300",
  REJECTED: "bg-red-100 text-red-800 dark:bg-red-900/50 dark:text-red-300",
  CANCELLED: "bg-gray-100 text-gray-800 dark:bg-gray-700/50 dark:text-gray-300",
  PENDING_CANCEL_L1: "bg-rose-100 text-rose-800 dark:bg-rose-900/50 dark:text-rose-300",
  PENDING_CANCEL_L2: "bg-rose-100 text-rose-800 dark:bg-rose-900/50 dark:text-rose-300",
};

export function FxStatusBadge({ status }: { status: FxStatus }) {
  const { t } = useTranslation();
  return (
    <Badge variant="secondary" className={statusStyles[status] || ""}>
      {t(`fx.status.${status}`)}
    </Badge>
  );
}
