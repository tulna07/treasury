"use client";

import Link from "next/link";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { FxStatusBadge } from "./fx-status-badge";
import { useTranslation } from "@/lib/i18n";
import { formatDate } from "@/lib/utils";
import type { FxDeal } from "@/hooks/use-fx";
import { IconChevronRight } from "@tabler/icons-react";

interface FxCardViewProps {
  deals: FxDeal[];
  onView: (deal: FxDeal) => void;
}

export function FxCardView({ deals, onView }: FxCardViewProps) {
  const { t } = useTranslation();

  if (deals.length === 0) return null;

  return (
    <div className="space-y-3">
      {deals.map((deal) => (
        <Link key={deal.id} href={`/fx/${deal.id}`} className="block">
        <Card
          className="cursor-pointer hover:bg-accent/50 transition-colors"
        >
          <CardContent className="p-4">
            <div className="flex items-start justify-between gap-2">
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-1">
                  <span className="font-mono text-sm font-medium text-primary">
                    {deal.ticket_number}
                  </span>
                  <FxStatusBadge status={deal.status} />
                </div>
                <p className="text-sm text-muted-foreground truncate">
                  {deal.counterparty_name}
                </p>
              </div>
              <Button variant="ghost" size="icon" className="size-8 shrink-0">
                <IconChevronRight className="size-4" />
              </Button>
            </div>

            <div className="flex flex-wrap items-center gap-x-3 gap-y-1 mt-2 text-sm">
              <span className="font-medium">{t(`fx.type.${deal.deal_type}`)}</span>
              <span className="text-muted-foreground">·</span>
              <span className="font-medium">{t(`fx.direction.${deal.direction}`)}</span>
              <span className="text-muted-foreground">·</span>
              <span className="font-medium tabular-nums">
                {Number(deal.notional_amount).toLocaleString("en-US", {
                  minimumFractionDigits: 0,
                  maximumFractionDigits: 0,
                })}
              </span>
              <span className="font-mono text-xs text-muted-foreground">{deal.currency_code}</span>
            </div>
            <div className="text-xs text-muted-foreground mt-1">
              {formatDate(deal.trade_date)}
            </div>
          </CardContent>
        </Card>
        </Link>
      ))}
    </div>
  );
}
