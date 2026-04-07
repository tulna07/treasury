"use client";

import * as React from "react";
import { IconTrendingUp, IconTrendingDown } from "@tabler/icons-react";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";

export interface StatsCardProps {
  /** Label hiển thị phía trên số */
  label: string;
  /** Giá trị chính (số hoặc text) */
  value: string | number;
  /** Icon hiển thị bên phải */
  icon?: React.ElementType;
  /** % thay đổi so với kỳ trước (dương = tăng, âm = giảm) */
  change?: number;
  /** Mô tả ngắn dưới footer */
  description?: string;
  /** Footer text in đậm */
  footerText?: string;
  /** Color variant */
  variant?: "default" | "success" | "danger" | "warning" | "info";
  /** Loading state */
  isLoading?: boolean;
}

const variantStyles = {
  default: {
    gradient: "from-primary/5 to-card dark:from-primary/10 dark:to-card",
    icon: "bg-primary/10 text-primary",
  },
  success: {
    gradient: "from-emerald-500/8 to-card dark:from-emerald-500/15 dark:to-card",
    icon: "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400",
  },
  danger: {
    gradient: "from-red-500/8 to-card dark:from-red-500/15 dark:to-card",
    icon: "bg-red-500/10 text-red-600 dark:text-red-400",
  },
  warning: {
    gradient: "from-amber-500/8 to-card dark:from-amber-500/15 dark:to-card",
    icon: "bg-amber-500/10 text-amber-600 dark:text-amber-400",
  },
  info: {
    gradient: "from-blue-500/8 to-card dark:from-blue-500/15 dark:to-card",
    icon: "bg-blue-500/10 text-blue-600 dark:text-blue-400",
  },
};

export function StatsCard({
  label,
  value,
  icon: Icon,
  change,
  description,
  footerText,
  variant = "default",
  isLoading,
}: StatsCardProps) {
  const styles = variantStyles[variant];

  if (isLoading) {
    return (
      <Card className="@container/card">
        <CardHeader>
          <div className="flex items-center justify-between">
            <div className="space-y-2 flex-1">
              <Skeleton className="h-4 w-20" />
              <Skeleton className="h-7 w-16" />
            </div>
            <Skeleton className="h-10 w-10 rounded-xl" />
          </div>
        </CardHeader>
        {(footerText || description) && (
          <CardFooter>
            <Skeleton className="h-3 w-28" />
          </CardFooter>
        )}
      </Card>
    );
  }

  const isPositive = change !== undefined && change >= 0;
  const TrendIcon = isPositive ? IconTrendingUp : IconTrendingDown;

  return (
    <Card className={cn("@container/card bg-gradient-to-t shadow-xs", styles.gradient)}>
      <CardHeader>
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <CardDescription>{label}</CardDescription>
            <CardTitle className="text-2xl font-semibold tabular-nums @[250px]/card:text-3xl mt-1">
              {value}
            </CardTitle>
          </div>
          <div className="flex items-center gap-2 shrink-0">
            {change !== undefined && (
              <Badge variant="outline" className="gap-1 hidden @[200px]/card:inline-flex">
                <TrendIcon className="size-3" />
                {isPositive ? "+" : ""}
                {change}%
              </Badge>
            )}
            {Icon && (
              <div className={cn("flex size-8 @[200px]/card:size-10 items-center justify-center rounded-lg @[200px]/card:rounded-xl", styles.icon)}>
                <Icon className="size-4 @[200px]/card:size-5" />
              </div>
            )}
          </div>
        </div>
      </CardHeader>
      {(footerText || description) && (
        <CardFooter className="flex-col items-start gap-1 text-sm">
          {footerText && (
            <div className="line-clamp-1 flex gap-2 font-medium text-xs">
              {footerText}
              {change !== undefined && <TrendIcon className="size-3.5" />}
            </div>
          )}
          {description && (
            <div className="text-muted-foreground text-xs">{description}</div>
          )}
        </CardFooter>
      )}
    </Card>
  );
}

export interface StatsGridProps {
  children: React.ReactNode;
  columns?: 2 | 3 | 4;
}

export function StatsGrid({ children, columns = 4 }: StatsGridProps) {
  const colClass =
    columns === 2
      ? "md:grid-cols-2"
      : columns === 3
        ? "md:grid-cols-3"
        : "md:grid-cols-4";

  return (
    <div
      className={cn(
        "grid grid-cols-2 gap-3",
        colClass
      )}
    >
      {children}
    </div>
  );
}
