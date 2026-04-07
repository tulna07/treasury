"use client";

import { usePathname } from "next/navigation";
import { Separator } from "@/components/ui/separator";
import { SidebarTrigger } from "@/components/ui/sidebar";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import { ThemeToggle } from "@/components/theme-toggle";
import { LanguageToggle } from "@/components/language-toggle";
import { NotificationBell } from "@/components/notification-bell";
import { useTranslation } from "@/lib/i18n";

const pathMap: Record<string, { labelKey: string; parent?: { labelKey: string; href: string } }> = {
  "/": { labelKey: "menu.dashboard" },
  "/fx": { labelKey: "menu.fx" },
  "/fx/new": {
    labelKey: "fx.new",
    parent: { labelKey: "menu.fx", href: "/fx" },
  },
  "/bonds": { labelKey: "menu.bonds" },
  "/bonds/new": {
    labelKey: "bond.new",
    parent: { labelKey: "menu.bonds", href: "/bonds" },
  },
  "/mm": { labelKey: "menu.mm" },
  "/limits": { labelKey: "menu.limits" },
  "/settlements": { labelKey: "menu.settlements" },
  "/notifications": { labelKey: "notifications.title" },
  "/exports": { labelKey: "exports.title" },
  "/profile": { labelKey: "profile.title" },
  "/settings": {
    labelKey: "settings.title",
  },
  "/settings/users": {
    labelKey: "users.title",
    parent: { labelKey: "settings.title", href: "/settings" },
  },
  "/settings/users/new": {
    labelKey: "users.new",
    parent: { labelKey: "users.title", href: "/settings/users" },
  },
  "/settings/roles": {
    labelKey: "roles.title",
    parent: { labelKey: "settings.title", href: "/settings" },
  },
  "/settings/audit-logs": {
    labelKey: "audit.title",
    parent: { labelKey: "settings.title", href: "/settings" },
  },
  "/settings/counterparties": {
    labelKey: "counterparties.title",
    parent: { labelKey: "settings.title", href: "/settings" },
  },
  "/settings/counterparties/new": {
    labelKey: "counterparties.new",
    parent: { labelKey: "counterparties.title", href: "/settings/counterparties" },
  },
};

function resolveDynamicPath(pathname: string): { labelKey: string; parent?: { labelKey: string; href: string } } | undefined {
  if (pathname.match(/^\/fx\/[^/]+\/edit$/)) {
    return { labelKey: "fx.edit", parent: { labelKey: "menu.fx", href: "/fx" } };
  }
  if (pathname.match(/^\/fx\/[^/]+$/)) {
    return { labelKey: "fx.detail", parent: { labelKey: "menu.fx", href: "/fx" } };
  }
  if (pathname.match(/^\/settings\/users\/[^/]+\/edit$/)) {
    return { labelKey: "users.edit", parent: { labelKey: "users.title", href: "/settings/users" } };
  }
  if (pathname.match(/^\/settings\/users\/[^/]+$/)) {
    return { labelKey: "users.detail", parent: { labelKey: "users.title", href: "/settings/users" } };
  }
  if (pathname.match(/^\/settings\/counterparties\/[^/]+\/edit$/)) {
    return { labelKey: "counterparties.edit", parent: { labelKey: "counterparties.title", href: "/settings/counterparties" } };
  }
  if (pathname.match(/^\/settings\/counterparties\/[^/]+$/)) {
    return { labelKey: "counterparties.detail", parent: { labelKey: "counterparties.title", href: "/settings/counterparties" } };
  }
  return undefined;
}

export function Header() {
  const pathname = usePathname();
  const { t } = useTranslation();
  const current = pathMap[pathname] ?? resolveDynamicPath(pathname);

  return (
    <header className="flex h-14 shrink-0 items-center justify-between border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 transition-[width,height] ease-linear group-has-data-[collapsible=icon]/sidebar-wrapper:h-12">
      <div className="flex items-center gap-2 px-4">
        <SidebarTrigger className="-ml-1" />
        <Separator
          orientation="vertical"
          className="mr-2 data-[orientation=vertical]:h-4"
        />
        <Breadcrumb>
          <BreadcrumbList>
            {current?.parent && (
              <>
                <BreadcrumbItem className="hidden md:block">
                  <BreadcrumbLink href={current.parent.href}>
                    {t(current.parent.labelKey)}
                  </BreadcrumbLink>
                </BreadcrumbItem>
                <BreadcrumbSeparator className="hidden md:block" />
              </>
            )}
            <BreadcrumbItem>
              <BreadcrumbPage>
                {current ? t(current.labelKey) : "Dashboard"}
              </BreadcrumbPage>
            </BreadcrumbItem>
          </BreadcrumbList>
        </Breadcrumb>
      </div>
      <div className="flex items-center gap-1 px-4">
        <NotificationBell />
        <LanguageToggle />
        <ThemeToggle />
      </div>
    </header>
  );
}
