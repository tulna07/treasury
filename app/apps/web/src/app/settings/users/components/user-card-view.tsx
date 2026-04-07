"use client";

import { useTranslation } from "@/lib/i18n";
import { formatDate } from "@/lib/utils";
import { UserStatusBadge } from "./user-status-badge";
import type { AdminUser } from "@/hooks/use-admin-users";

interface UserCardViewProps {
  users: AdminUser[];
  onView: (user: AdminUser) => void;
}

export function UserCardView({ users, onView }: UserCardViewProps) {
  const { t } = useTranslation();

  return (
    <div className="space-y-3">
      {users.map((user) => (
        <div
          key={user.id}
          className="rounded-lg border p-4 space-y-2 cursor-pointer hover:bg-accent/50 transition-colors"
          onClick={() => onView(user)}
        >
          <div className="flex items-center justify-between">
            <div>
              <span className="font-medium">{user.full_name}</span>
              <span className="text-sm text-muted-foreground ml-2">
                @{user.username}
              </span>
            </div>
            <UserStatusBadge isActive={user.is_active} />
          </div>
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm">
            <span className="text-muted-foreground">{user.department}</span>
            <span className="text-muted-foreground">·</span>
            <span>{user.role_names?.join(", ") || "-"}</span>
          </div>
        </div>
      ))}
    </div>
  );
}
