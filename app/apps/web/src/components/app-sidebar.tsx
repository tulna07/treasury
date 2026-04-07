"use client";

import * as React from "react";
import {
  IconLayoutDashboard,
  IconArrowsExchange,
  IconReceiptTax,
  IconCoins,
  IconScale,
  IconGlobe,
  IconSettings,
  IconChevronRight,
  IconUsersGroup,
  IconShieldLock,
  IconClipboardList,
  IconBuildingBank,
} from "@tabler/icons-react";
import Link from "next/link";
import { usePathname } from "next/navigation";

import { NavUser } from "@/components/nav-user";
import { useTranslation } from "@/lib/i18n";
import { useAbility } from "@/hooks/use-ability";
import type { Actions, Subjects } from "@/lib/ability";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarGroup,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
  SidebarRail,
} from "@/components/ui/sidebar";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";

interface MenuItem {
  titleKey: string;
  url: string;
  icon: React.ElementType;
  permission?: [Actions, Subjects];
}

interface MenuGroup {
  labelKey: string;
  items: MenuItem[];
  collapsible?: boolean;
  pathPrefix?: string;
  icon?: React.ElementType;
}

function KienlongLogo({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 48 48" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path d="M42.7165 27.6486C45.3313 27.6486 47.451 25.5443 47.451 22.9485C47.451 20.3527 45.3313 18.2484 42.7165 18.2484C40.1017 18.2484 37.9819 20.3527 37.9819 22.9485C37.9819 25.5443 40.1017 27.6486 42.7165 27.6486Z" fill="currentColor"/>
      <path d="M6.2575 27.6486C8.87232 27.6486 10.9921 25.5443 10.9921 22.9485C10.9921 20.3527 8.87232 18.2484 6.2575 18.2484C3.64268 18.2484 1.52295 20.3527 1.52295 22.9485C1.52295 25.5443 3.64268 27.6486 6.2575 27.6486Z" fill="currentColor"/>
      <path fillRule="evenodd" clipRule="evenodd" d="M38.2696 32.1104C38.1134 29.7842 36.2148 27.8875 33.8595 27.7324C33.3788 27.6966 32.9222 27.7443 32.4776 27.8397C31.7205 28.0068 30.9395 27.7682 30.3867 27.2194L20.509 17.4136C19.9442 16.8052 19.764 15.9583 20.0043 15.159C20.1365 14.7296 20.2086 14.2763 20.2086 13.7991C20.2086 13.5247 20.1846 13.2503 20.1365 12.9879C20.0284 12.0574 20.3408 11.1031 21.0498 10.3873C21.7708 9.67159 22.7321 9.37336 23.6694 9.48072C23.7295 9.49265 24.126 9.5523 24.4865 9.5523C24.859 9.5523 25.2436 9.49265 25.3037 9.48072C26.241 9.37336 27.2023 9.68352 27.9233 10.3873C28.6443 11.1031 28.9447 12.0574 28.8365 12.9879C28.7885 13.2503 28.7644 13.5247 28.7644 13.7991C28.7644 16.507 31.0716 18.6781 33.8355 18.4873C36.1667 18.3322 38.0653 16.4474 38.2215 14.1331C38.4138 11.3894 36.2268 9.09899 33.499 9.09899C33.2226 9.09899 32.9462 9.12285 32.6819 9.17057C31.7446 9.27793 30.7832 8.96777 30.0622 8.26395C29.3412 7.5482 29.0408 6.59386 29.149 5.66338C29.2571 5.15043 29.2091 4.53011 29.197 4.37503C28.9807 2.23971 27.3225 0.521904 25.2075 0.211746C24.9672 0.175958 24.7269 0.1521 24.4865 0.1521C24.2462 0.1521 24.0059 0.175958 23.7655 0.211746C21.6506 0.533834 19.9923 2.25164 19.776 4.37503C19.764 4.53011 19.7159 5.24586 19.8241 5.66338C19.9322 6.59386 19.6198 7.5482 18.9108 8.26395C18.1898 8.9797 17.2285 9.27793 16.2912 9.17057C16.0268 9.12285 15.7504 9.09899 15.4741 9.09899C12.7463 9.09899 10.5593 11.3775 10.7515 14.1212C10.9077 16.4474 12.8064 18.3441 15.1616 18.4992C15.6423 18.535 16.0989 18.4873 16.5435 18.3918C17.3006 18.2248 18.0817 18.4634 18.6344 19.0121L28.5121 28.8179C29.0769 29.4263 29.2571 30.2733 29.0168 31.0726C28.8846 31.502 28.8125 31.9553 28.8125 32.4325C28.8125 32.7069 28.8365 32.9812 28.8846 33.2437C28.9928 34.1741 28.6803 35.1285 27.9713 35.8442C27.2504 36.56 26.289 36.8582 25.3517 36.7509C25.2916 36.7389 24.8951 36.6793 24.5346 36.6793C24.1621 36.6793 23.7775 36.7389 23.7175 36.7509C22.7802 36.8582 21.8188 36.5481 21.0978 35.8442C20.3768 35.1285 20.0764 34.1741 20.1846 33.2437C20.2326 32.9812 20.2567 32.7069 20.2567 32.4325C20.2567 29.7246 17.9495 27.5534 15.1857 27.7443C12.8544 27.8994 10.9558 29.7842 10.7996 32.0985C10.6073 34.8422 12.7943 37.1326 15.5221 37.1326C15.7985 37.1326 16.0749 37.1087 16.3393 37.061C17.2766 36.9536 18.2379 37.2638 18.9589 37.9676C19.6799 38.6834 19.9803 39.6377 19.8721 40.5682C19.764 41.0811 19.8121 41.7015 19.8241 41.8565C20.0404 43.9919 21.6987 45.7097 23.8136 46.0198C24.0539 46.0556 24.2943 46.0795 24.5346 46.0795C24.7749 46.0795 25.0153 46.0556 25.2556 46.0198C27.3705 45.6977 29.0288 43.9799 29.2451 41.8565C29.2571 41.7015 29.3052 40.9857 29.197 40.5682C29.0889 39.6377 29.4013 38.6834 30.1103 37.9676C30.8313 37.2519 31.7926 36.9536 32.7299 37.061C32.9943 37.1087 33.2707 37.1326 33.5471 37.1326C36.2748 37.1326 38.4619 34.8541 38.2696 32.1104Z" fill="currentColor"/>
    </svg>
  );
}

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  const pathname = usePathname();
  const { t } = useTranslation();
  const ability = useAbility();

  const topItems: MenuItem[] = [
    { titleKey: "menu.dashboard", url: "/", icon: IconLayoutDashboard },
  ];

  const menuGroups: MenuGroup[] = [
    {
      labelKey: "sidebar.transactions",
      items: [
        { titleKey: "menu.fx", url: "/fx", icon: IconArrowsExchange, permission: ["view", "FXTransaction"] },
        { titleKey: "menu.bonds", url: "/bonds", icon: IconBuildingBank, permission: ["view", "GTCGTransaction"] },
        { titleKey: "menu.bondInventory", url: "/bonds/inventory", icon: IconReceiptTax, permission: ["view", "GTCGTransaction"] },
        { titleKey: "menu.mm", url: "/mm", icon: IconCoins, permission: ["view", "MMTransaction"] },
      ],
    },
    {
      labelKey: "sidebar.management",
      items: [
        { titleKey: "menu.limits", url: "/limits", icon: IconScale, permission: ["view", "Limit"] },
        { titleKey: "menu.settlements", url: "/settlements", icon: IconGlobe, permission: ["view", "Settlement"] },
        { titleKey: "menu.counterparties", url: "/settings/counterparties", icon: IconBuildingBank },
      ],
    },
    {
      labelKey: "sidebar.admin",
      items: [
        { titleKey: "menu.users", url: "/settings/users", icon: IconUsersGroup, permission: ["manage", "Settings"] },
        { titleKey: "menu.roles", url: "/settings/roles", icon: IconShieldLock, permission: ["manage", "Settings"] },
        { titleKey: "menu.auditLogs", url: "/settings/audit-logs", icon: IconClipboardList, permission: ["view", "AuditLog"] },
        { titleKey: "menu.settings", url: "/settings", icon: IconSettings, permission: ["manage", "Settings"] },
      ],
    },
  ];

  // Filter items by permission
  const filterItems = (items: MenuItem[]) =>
    items.filter((item) => {
      if (!item.permission) return true;
      return ability.can(item.permission[0], item.permission[1]);
    });

  // Filter groups: remove groups with no visible items
  const visibleGroups = menuGroups
    .map((group) => ({
      ...group,
      items: filterItems(group.items),
    }))
    .filter((group) => group.items.length > 0);

  return (
    <Sidebar collapsible="icon" {...props}>
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton size="lg" render={<Link href="/" />}>
              <div className="flex aspect-square size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
                <KienlongLogo className="size-5" />
              </div>
              <div className="grid flex-1 text-left text-sm leading-tight">
                <span className="truncate font-bold">KienlongBank</span>
                <span className="truncate text-xs text-muted-foreground">
                  {t("common.treasury")}
                </span>
              </div>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>

      <SidebarContent>
        {/* Top items — Dashboard */}
        <SidebarGroup>
          <SidebarGroupLabel>{t("sidebar.overview")}</SidebarGroupLabel>
          <SidebarMenu>
            {topItems.map((item) => (
              <SidebarMenuItem key={item.url}>
                <SidebarMenuButton
                  tooltip={t(item.titleKey)}
                  isActive={pathname === item.url}
                  render={<Link href={item.url} />}
                >
                  <item.icon className="size-4" />
                  <span>{t(item.titleKey)}</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            ))}
          </SidebarMenu>
        </SidebarGroup>

        {/* Menu groups */}
        {visibleGroups.map((group) =>
          group.collapsible ? (
            <CollapsibleMenuGroup
              key={group.labelKey}
              group={group}
              pathname={pathname}
              t={t}
            />
          ) : (
            <SidebarGroup key={group.labelKey}>
              <SidebarGroupLabel>{t(group.labelKey)}</SidebarGroupLabel>
              <SidebarMenu>
                {group.items.map((item) => (
                  <SidebarMenuItem key={item.url}>
                    <SidebarMenuButton
                      tooltip={t(item.titleKey)}
                      isActive={pathname === item.url}
                      render={<Link href={item.url} />}
                    >
                      <item.icon className="size-4" />
                      <span>{t(item.titleKey)}</span>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                ))}
              </SidebarMenu>
            </SidebarGroup>
          )
        )}
      </SidebarContent>

      <SidebarFooter>
        <NavUser />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  );
}

function CollapsibleMenuGroup({
  group,
  pathname,
  t,
}: {
  group: MenuGroup;
  pathname: string;
  t: (key: string) => string;
}) {
  const isActive = group.pathPrefix
    ? pathname.startsWith(group.pathPrefix)
    : false;
  const [open, setOpen] = React.useState(isActive);

  React.useEffect(() => {
    if (isActive) setOpen(true);
  }, [isActive]);

  const GroupIcon = group.icon ?? IconSettings;

  return (
    <SidebarGroup>
      <SidebarGroupLabel>{t(group.labelKey)}</SidebarGroupLabel>
      <SidebarMenu>
        <Collapsible open={open} onOpenChange={setOpen} className="group/collapsible">
          <SidebarMenuItem>
            <CollapsibleTrigger
              render={
                <SidebarMenuButton tooltip={t(group.labelKey)} isActive={isActive} />
              }
            >
              <GroupIcon className="size-4" />
              <span>{t(group.labelKey)}</span>
              <IconChevronRight className="ml-auto size-4 transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90" />
            </CollapsibleTrigger>
            <CollapsibleContent>
              <SidebarMenuSub>
                {group.items.map((item) => (
                  <SidebarMenuSubItem key={item.url}>
                    <SidebarMenuSubButton
                      isActive={pathname === item.url}
                      render={<Link href={item.url} />}
                    >
                      <item.icon className="size-3.5" />
                      <span>{t(item.titleKey)}</span>
                    </SidebarMenuSubButton>
                  </SidebarMenuSubItem>
                ))}
              </SidebarMenuSub>
            </CollapsibleContent>
          </SidebarMenuItem>
        </Collapsible>
      </SidebarMenu>
    </SidebarGroup>
  );
}
