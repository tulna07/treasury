"use client";

import Link from "next/link";
import { Header } from "@/components/header";
import { useTranslation } from "@/lib/i18n";
import { useAbility } from "@/hooks/use-ability";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  IconUsers,
  IconShieldCog,
  IconFileText,
  IconBuildingBank,
  IconChevronRight,
} from "@tabler/icons-react";

interface SettingsCard {
  titleKey: string;
  descriptionKey: string;
  href: string;
  icon: React.ElementType;
  visible: boolean;
}

export default function SettingsPage() {
  const { t } = useTranslation();
  const ability = useAbility();

  const cards: SettingsCard[] = [
    {
      titleKey: "settings.users",
      descriptionKey: "settings.usersDescription",
      href: "/settings/users",
      icon: IconUsers,
      visible: ability.can("manage", "Settings"),
    },
    {
      titleKey: "settings.roles",
      descriptionKey: "settings.rolesDescription",
      href: "/settings/roles",
      icon: IconShieldCog,
      visible: ability.can("manage", "Settings"),
    },
    {
      titleKey: "settings.auditLogs",
      descriptionKey: "settings.auditLogsDescription",
      href: "/settings/audit-logs",
      icon: IconFileText,
      visible: ability.can("view", "AuditLog") || ability.can("manage", "Settings"),
    },
    {
      titleKey: "settings.counterparties",
      descriptionKey: "settings.counterpartiesDescription",
      href: "/settings/counterparties",
      icon: IconBuildingBank,
      visible: ability.can("manage", "Settings"),
    },
  ];

  const visibleCards = cards.filter((c) => c.visible);

  return (
    <>
      <Header />
      <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
        <div className="pt-4">
          <h1 className="text-2xl font-bold tracking-tight">
            {t("settings.title")}
          </h1>
          <p className="text-muted-foreground">{t("settings.description")}</p>
        </div>

        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {visibleCards.map((card) => (
            <Link key={card.href} href={card.href}>
              <Card className="group cursor-pointer transition-colors hover:border-primary/50 hover:bg-accent/50 h-full">
                <CardHeader className="flex flex-row items-center justify-between pb-2">
                  <card.icon className="size-8 text-muted-foreground group-hover:text-primary transition-colors" />
                  <IconChevronRight className="size-4 text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity" />
                </CardHeader>
                <CardContent>
                  <CardTitle className="text-base mb-1">
                    {t(card.titleKey)}
                  </CardTitle>
                  <p className="text-sm text-muted-foreground">
                    {t(card.descriptionKey)}
                  </p>
                </CardContent>
              </Card>
            </Link>
          ))}
        </div>
      </div>
    </>
  );
}
