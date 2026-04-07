"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Header } from "@/components/header";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import {
  IconAlertCircle,
  IconCheckbox,
  IconBell,
} from "@tabler/icons-react";
import { useTranslation } from "@/lib/i18n";
import { formatDate, extractErrorMessage } from "@/lib/utils";
import {
  useNotifications,
  useMarkRead,
  useMarkAllRead,
} from "@/hooks/use-notifications";
import type { Notification } from "@/hooks/use-notifications";
import { PaginationBar } from "@/components/pagination";

function getNotificationHref(notif: Notification): string {
  if (notif.related_id && notif.related_module) {
    const moduleMap: Record<string, string> = {
      FX: "/fx",
      GTCG: "/gtcg",
      MM: "/mm",
      SETTLEMENT: "/settlements",
    };
    const base = moduleMap[notif.related_module] || "/fx";
    return `${base}/${notif.related_id}`;
  }
  return "#";
}

export default function NotificationsPage() {
  const { t } = useTranslation();
  const router = useRouter();
  const [page, setPage] = useState(1);
  const [unreadOnly, setUnreadOnly] = useState(false);

  const { data, isLoading, isError, error } = useNotifications(page);
  const markRead = useMarkRead();
  const markAllRead = useMarkAllRead();

  const notifications = data?.data ?? [];
  const total = data?.total ?? 0;

  const displayed = unreadOnly
    ? notifications.filter((n) => !n.is_read)
    : notifications;

  function handleClick(notif: Notification) {
    if (!notif.is_read) {
      markRead.mutate(notif.id);
    }
    const href = getNotificationHref(notif);
    if (href !== "#") {
      router.push(href);
    }
  }

  return (
    <>
      <Header />
      <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
        <div className="flex flex-col gap-4 pt-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight">
              {t("notifications.title")}
            </h1>
            <p className="text-muted-foreground">
              {t("notifications.description")}
            </p>
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={() => markAllRead.mutate()}
            disabled={markAllRead.isPending}
          >
            <IconCheckbox className="mr-1.5 size-4" />
            {t("notifications.markAllRead")}
          </Button>
        </div>

        <div className="flex items-center gap-2">
          <Checkbox
            id="unread_only"
            checked={unreadOnly}
            onCheckedChange={(v) => setUnreadOnly(!!v)}
          />
          <Label htmlFor="unread_only" className="text-sm cursor-pointer">
            {t("notifications.unreadOnly")}
          </Label>
        </div>

        {isError && (
          <Alert variant="destructive">
            <IconAlertCircle className="size-4" />
            <AlertDescription>{extractErrorMessage(error)}</AlertDescription>
          </Alert>
        )}

        {isLoading ? (
          <div className="space-y-3">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} className="h-16 w-full" />
            ))}
          </div>
        ) : displayed.length === 0 ? (
          <Card>
            <CardContent className="py-12 text-center">
              <IconBell className="mx-auto size-10 text-muted-foreground/50 mb-3" />
              <p className="text-muted-foreground">{t("notifications.empty")}</p>
            </CardContent>
          </Card>
        ) : (
          <Card>
            <CardContent className="p-0">
              {displayed.map((notif, idx) => (
                <div key={notif.id}>
                  <button
                    onClick={() => handleClick(notif)}
                    className={`w-full text-left p-4 hover:bg-accent transition-colors ${
                      !notif.is_read ? "bg-accent/50" : ""
                    }`}
                  >
                    <div className="flex items-start justify-between gap-3">
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          {!notif.is_read && (
                            <div className="size-2 rounded-full bg-primary shrink-0" />
                          )}
                          <p className="text-sm font-medium">{notif.title}</p>
                        </div>
                        <p className="text-sm text-muted-foreground mt-0.5">
                          {notif.message}
                        </p>
                      </div>
                      <div className="text-right shrink-0">
                        <p className="text-xs text-muted-foreground">
                          {formatDate(notif.created_at)}
                        </p>
                        {notif.related_module && (
                          <span className="text-xs font-mono text-muted-foreground">
                            {notif.related_module}
                          </span>
                        )}
                      </div>
                    </div>
                  </button>
                  {idx < displayed.length - 1 && (
                    <div className="border-b" />
                  )}
                </div>
              ))}
            </CardContent>
          </Card>
        )}

        <PaginationBar
          page={page}
          total={total}
          pageSize={20}
          onPageChange={setPage}
        />
      </div>
    </>
  );
}
