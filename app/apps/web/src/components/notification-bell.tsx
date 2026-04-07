"use client";

import { useState, useRef, useEffect } from "react";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { IconBell, IconCheck } from "@tabler/icons-react";
import { useTranslation } from "@/lib/i18n";
import {
  useSSENotifications,
  useUnreadNotifications,
  useMarkRead,
  useMarkAllRead,
} from "@/hooks/use-notifications";
import type { Notification } from "@/hooks/use-notifications";

function timeAgo(dateStr: string, t: (key: string) => string): string {
  const diff = Date.now() - new Date(dateStr).getTime();
  const minutes = Math.floor(diff / 60000);
  if (minutes < 1) return t("notifications.justNow");
  if (minutes < 60) return `${minutes}m`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h`;
  const days = Math.floor(hours / 24);
  return `${days}d`;
}

export function NotificationBell() {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  const { unreadCount } = useSSENotifications();
  const { data: recentData } = useUnreadNotifications();
  const markRead = useMarkRead();
  const markAllRead = useMarkAllRead();

  const notifications = recentData?.data ?? [];
  const displayCount = unreadCount || recentData?.total || 0;

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  function handleNotificationClick(notif: Notification) {
    if (!notif.is_read) {
      markRead.mutate(notif.id);
    }
    setOpen(false);
  }

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
    return "/notifications";
  }

  return (
    <div className="relative" ref={ref}>
      <Button
        variant="ghost"
        size="icon"
        className="relative size-8"
        onClick={() => setOpen(!open)}
      >
        <IconBell className="size-4" />
        {displayCount > 0 && (
          <Badge className="absolute -top-1 -right-1 size-4 rounded-full p-0 text-[10px] flex items-center justify-center">
            {displayCount > 99 ? "99+" : displayCount}
          </Badge>
        )}
      </Button>

      {open && (
        <div className="absolute right-0 top-full mt-2 z-50 w-80 rounded-lg border bg-popover shadow-lg">
          <div className="flex items-center justify-between p-3">
            <h3 className="text-sm font-semibold">{t("notifications.title")}</h3>
            {displayCount > 0 && (
              <Button
                variant="ghost"
                size="sm"
                className="h-7 text-xs"
                onClick={() => markAllRead.mutate()}
              >
                <IconCheck className="mr-1 size-3" />
                {t("notifications.markAllRead")}
              </Button>
            )}
          </div>
          <Separator />
          <div className="max-h-80 overflow-y-auto">
            {notifications.length === 0 ? (
              <p className="p-4 text-center text-sm text-muted-foreground">
                {t("notifications.empty")}
              </p>
            ) : (
              notifications.map((notif) => (
                <Link
                  key={notif.id}
                  href={getNotificationHref(notif)}
                  onClick={() => handleNotificationClick(notif)}
                  className={`block p-3 hover:bg-accent transition-colors ${
                    !notif.is_read ? "bg-accent/50" : ""
                  }`}
                >
                  <div className="flex items-start justify-between gap-2">
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium truncate">{notif.title}</p>
                      <p className="text-xs text-muted-foreground line-clamp-2">
                        {notif.message}
                      </p>
                    </div>
                    <span className="text-xs text-muted-foreground whitespace-nowrap">
                      {timeAgo(notif.created_at, t)}
                    </span>
                  </div>
                </Link>
              ))
            )}
          </div>
          <Separator />
          <div className="p-2">
            <Link
              href="/notifications"
              onClick={() => setOpen(false)}
              className="block text-center text-sm text-primary hover:underline py-1"
            >
              {t("notifications.viewAll")}
            </Link>
          </div>
        </div>
      )}
    </div>
  );
}
