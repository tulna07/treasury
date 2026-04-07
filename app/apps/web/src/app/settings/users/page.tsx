"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { Header } from "@/components/header";
import { useTranslation } from "@/lib/i18n";
import { useAbility } from "@/hooks/use-ability";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { PaginationBar } from "@/components/pagination";
import {
  IconPlus,
  IconAlertCircle,
  IconInbox,
  IconSearch,
} from "@tabler/icons-react";
import { formatDate, extractErrorMessage } from "@/lib/utils";
import { useIsMobile } from "@/hooks/use-mobile";
import { useAdminUsers, type AdminUser, type AdminUserFilters } from "@/hooks/use-admin-users";
import { UserStatusBadge } from "./components/user-status-badge";
import { UserCardView } from "./components/user-card-view";

export default function UsersPage() {
  const { t } = useTranslation();
  const ability = useAbility();
  const router = useRouter();
  const isMobile = useIsMobile();
  const [filters, setFilters] = useState<AdminUserFilters>({
    page: 1,
    page_size: 20,
  });
  const [search, setSearch] = useState("");

  const { data, isLoading, isError, error, refetch } = useAdminUsers(filters);
  const users = data?.data ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.ceil(total / (filters.page_size || 20));

  function handleSearch() {
    setFilters((f) => ({ ...f, search: search || undefined, page: 1 }));
  }

  function handleView(user: AdminUser) {
    router.push(`/settings/users/${user.id}`);
  }

  return (
    <>
      <Header />
      <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
        <div className="flex flex-col gap-4 pt-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight">{t("users.title")}</h1>
            <p className="text-muted-foreground">{t("users.description")}</p>
          </div>
          {ability.can("manage", "Settings") && (
            <Button onClick={() => router.push("/settings/users/new")} className="shrink-0">
              <IconPlus className="mr-2 size-4" />
              {t("users.new")}
            </Button>
          )}
        </div>

        {/* Filters */}
        <div className="flex flex-col gap-2 sm:flex-row">
          <div className="relative flex-1">
            <IconSearch className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder={t("users.search")}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && handleSearch()}
              className="pl-9"
            />
          </div>
          <Select
            value={filters.department || "all"}
            onValueChange={(v) => {
              const val = !v || v === "all" ? undefined : v;
              setFilters((f) => ({ ...f, department: val, page: 1 }));
            }}
          >
            <SelectTrigger className="w-full sm:w-auto">
              <SelectValue placeholder={t("users.filterByDepartment")}>{(v: string) => v === "all" ? t("users.allDepartments") : v}</SelectValue>
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all" label={t("users.allDepartments")}>{t("users.allDepartments")}</SelectItem>
              <SelectItem value="FX Trading" label="FX Trading">FX Trading</SelectItem>
              <SelectItem value="Treasury" label="Treasury">Treasury</SelectItem>
              <SelectItem value="Risk" label="Risk">Risk</SelectItem>
              <SelectItem value="Accounting" label="Accounting">Accounting</SelectItem>
              <SelectItem value="Operations" label="Operations">Operations</SelectItem>
              <SelectItem value="IT" label="IT">IT</SelectItem>
            </SelectContent>
          </Select>
          <Select
            value={filters.is_active || "all"}
            onValueChange={(v) => {
              const val = !v || v === "all" ? undefined : v;
              setFilters((f) => ({ ...f, is_active: val, page: 1 }));
            }}
          >
            <SelectTrigger className="w-full sm:w-auto">
              <SelectValue placeholder={t("users.filterByStatus")}>{(v: string) => v === "all" ? t("users.allStatuses") : v === "true" ? t("users.activeOnly") : t("users.lockedOnly")}</SelectValue>
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all" label={t("users.allStatuses")}>{t("users.allStatuses")}</SelectItem>
              <SelectItem value="true" label={t("users.activeOnly")}>{t("users.activeOnly")}</SelectItem>
              <SelectItem value="false" label={t("users.lockedOnly")}>{t("users.lockedOnly")}</SelectItem>
            </SelectContent>
          </Select>
        </div>

        {/* Content */}
        {isError ? (
          <Alert variant="destructive">
            <IconAlertCircle className="size-4" />
            <AlertDescription className="flex items-center justify-between">
              <span>{t("users.loadError")}: {extractErrorMessage(error)}</span>
              <button onClick={() => refetch()} className="underline font-medium ml-2">
                {t("common.retry")}
              </button>
            </AlertDescription>
          </Alert>
        ) : isLoading ? (
          <div className="space-y-3">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} className="h-14 w-full rounded-lg" />
            ))}
          </div>
        ) : users.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 text-center">
            <IconInbox className="size-12 text-muted-foreground/50 mb-4" />
            <h3 className="text-lg font-medium">{t("users.noUsers")}</h3>
            <p className="text-sm text-muted-foreground mt-1">{t("users.noUsersDescription")}</p>
          </div>
        ) : (
          <div className="space-y-4">
            {isMobile ? (
              <UserCardView users={users} onView={handleView} />
            ) : (
              <div className="rounded-lg border overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{t("common.name")}</TableHead>
                      <TableHead>{t("common.username")}</TableHead>
                      <TableHead>{t("common.email")}</TableHead>
                      <TableHead>{t("common.department")}</TableHead>
                      <TableHead>{t("common.roles")}</TableHead>
                      <TableHead>{t("common.status")}</TableHead>
                      <TableHead>{t("common.lastLogin")}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {users.map((user) => (
                      <TableRow
                        key={user.id}
                        className="cursor-pointer"
                        onClick={() => handleView(user)}
                      >
                        <TableCell className="font-medium">
                          <Link href={`/settings/users/${user.id}`} className="hover:underline text-primary">
                            {user.full_name}
                          </Link>
                        </TableCell>
                        <TableCell className="font-mono text-sm">{user.username}</TableCell>
                        <TableCell className="text-sm">{user.email}</TableCell>
                        <TableCell>{user.department}</TableCell>
                        <TableCell>
                          <div className="flex flex-wrap gap-1">
                            {user.role_names?.map((role) => (
                              <Badge key={role} variant="secondary" className="text-xs">
                                {role}
                              </Badge>
                            ))}
                          </div>
                        </TableCell>
                        <TableCell>
                          <UserStatusBadge isActive={user.is_active} />
                        </TableCell>
                        <TableCell className="text-sm text-muted-foreground">
                          {user.last_login_at
                            ? formatDate(user.last_login_at)
                            : t("common.never")}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}

            {totalPages > 1 && (
              <PaginationBar
                page={filters.page || 1}
                total={total}
                pageSize={filters.page_size || 20}
                onPageChange={(p) => setFilters((f) => ({ ...f, page: p }))}
              />
            )}
          </div>
        )}
      </div>
    </>
  );
}
