"use client";

import { useState, useMemo, useCallback, useEffect } from "react";
import { Header } from "@/components/header";
import { useTranslation } from "@/lib/i18n";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import {
  IconAlertCircle,
  IconChevronDown,
  IconInbox,
  IconLoader2,
} from "@tabler/icons-react";
import { extractErrorMessage } from "@/lib/utils";
import { useIsMobile } from "@/hooks/use-mobile";
import {
  useAdminRoles,
  useRolePermissions,
  useAllPermissions,
  useUpdateRolePermissions,
  type AdminRole,
} from "@/hooks/use-admin-roles";
import { toast } from "sonner";

// ─── Permission Editor ─────────────────────────────────────────

function RolePermissionsPanel({ roleCode }: { roleCode: string }) {
  const { t } = useTranslation();
  const { data: currentPerms, isLoading: permsLoading } =
    useRolePermissions(roleCode);
  const { data: allPermissions, isLoading: allPermsLoading } =
    useAllPermissions();
  const updateMutation = useUpdateRolePermissions();

  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [initialized, setInitialized] = useState(false);

  // Sync selected state whenever currentPerms changes (initial load + after save refetch)
  const currentPermsKey = currentPerms?.join(",") ?? "";
  useEffect(() => {
    if (currentPerms) {
      setSelected(new Set(currentPerms));
      setInitialized(true);
    }
  }, [currentPermsKey]); // eslint-disable-line react-hooks/exhaustive-deps

  // Group permissions by module prefix (before first ".")
  const grouped = useMemo(() => {
    if (!allPermissions) return {};
    const groups: Record<string, typeof allPermissions> = {};
    for (const perm of allPermissions) {
      const dotIdx = perm.code.indexOf(".");
      const module = dotIdx > 0 ? perm.code.substring(0, dotIdx) : perm.code;
      if (!groups[module]) groups[module] = [];
      groups[module].push(perm);
    }
    return groups;
  }, [allPermissions]);

  const moduleNames = useMemo(() => Object.keys(grouped).sort(), [grouped]);

  // Dirty check
  const isDirty = useMemo(() => {
    if (!currentPerms || !initialized) return false;
    const currentSet = new Set(currentPerms);
    if (currentSet.size !== selected.size) return true;
    for (const code of selected) {
      if (!currentSet.has(code)) return true;
    }
    return false;
  }, [currentPerms, selected, initialized]);

  const togglePermission = useCallback((code: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(code)) {
        next.delete(code);
      } else {
        next.add(code);
      }
      return next;
    });
  }, []);

  const selectAllModule = useCallback(
    (module: string) => {
      const perms = grouped[module];
      if (!perms) return;
      setSelected((prev) => {
        const next = new Set(prev);
        for (const p of perms) next.add(p.code);
        return next;
      });
    },
    [grouped]
  );

  const deselectAllModule = useCallback(
    (module: string) => {
      const perms = grouped[module];
      if (!perms) return;
      setSelected((prev) => {
        const next = new Set(prev);
        for (const p of perms) next.delete(p.code);
        return next;
      });
    },
    [grouped]
  );

  const handleSave = useCallback(() => {
    updateMutation.mutate(
      { roleCode, permissions: Array.from(selected) },
      {
        onSuccess: () => {
          toast.success(t("roles.saveSuccess"));
          // Query invalidation in mutation hook triggers refetch → useEffect syncs selected
        },
        onError: (err) => {
          toast.error(
            t("roles.saveError") + ": " + extractErrorMessage(err)
          );
        },
      }
    );
  }, [roleCode, selected, updateMutation, t]);

  if (permsLoading || allPermsLoading) {
    return (
      <div className="space-y-2">
        <Skeleton className="h-8 w-full" />
        <Skeleton className="h-8 w-3/4" />
      </div>
    );
  }

  if (!allPermissions?.length) {
    return (
      <span className="text-sm text-muted-foreground">
        {t("roles.noPermissions")}
      </span>
    );
  }

  return (
    <div className="space-y-4">
      {moduleNames.map((module) => {
        const perms = grouped[module];
        const allChecked = perms.every((p) => selected.has(p.code));
        const someChecked =
          !allChecked && perms.some((p) => selected.has(p.code));
        const label = module.replace(/_/g, " ");

        return (
          <Collapsible key={module} defaultOpen>
            <div className="space-y-2">
              <div className="flex items-center gap-2">
                <CollapsibleTrigger className="flex items-center gap-1 text-sm font-medium hover:underline">
                  <IconChevronDown className="size-4 text-muted-foreground" />
                  {label}
                </CollapsibleTrigger>
                <Badge variant="secondary" className="text-xs">
                  {perms.filter((p) => selected.has(p.code)).length}/
                  {perms.length}
                </Badge>
                <button
                  type="button"
                  className="text-xs text-primary hover:underline ml-auto"
                  onClick={() =>
                    allChecked
                      ? deselectAllModule(module)
                      : selectAllModule(module)
                  }
                >
                  {allChecked
                    ? t("roles.deselectAll")
                    : t("roles.selectAll")}
                </button>
              </div>
              <CollapsibleContent>
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2 pl-5">
                  {perms.map((perm) => (
                    <div
                      key={perm.code}
                      className="flex items-center gap-2 text-sm cursor-pointer rounded-md px-2 py-1.5 hover:bg-muted/50"
                      onClick={(e) => {
                        // Avoid double-toggle when clicking the checkbox itself
                        if ((e.target as HTMLElement).closest('[data-slot="checkbox"]')) return;
                        togglePermission(perm.code);
                      }}
                    >
                      <Checkbox
                        checked={selected.has(perm.code)}
                        onCheckedChange={() => togglePermission(perm.code)}
                      />
                      <span className="font-mono text-xs">{perm.code}</span>
                    </div>
                  ))}
                </div>
              </CollapsibleContent>
            </div>
          </Collapsible>
        );
      })}

      <div className="flex justify-end pt-2">
        <Button
          onClick={handleSave}
          disabled={!isDirty || updateMutation.isPending}
          size="sm"
        >
          {updateMutation.isPending && (
            <IconLoader2 className="size-4 mr-1 animate-spin" />
          )}
          {t("roles.savePermissions")}
        </Button>
      </div>
    </div>
  );
}

// ─── Role Card ──────────────────────────────────────────────────

function RoleCard({ role }: { role: AdminRole }) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);

  return (
    <Collapsible open={open} onOpenChange={setOpen}>
      <Card>
        <CollapsibleTrigger className="w-full">
          <CardHeader className="flex flex-row items-center gap-2 cursor-pointer">
            <div className="flex-1 text-left">
              <CardTitle className="text-base">{role.name}</CardTitle>
              <p className="text-sm text-muted-foreground">
                {role.code} — {role.description}
              </p>
            </div>
            <Badge variant="secondary">{role.scope}</Badge>
            <IconChevronDown
              className={`size-4 text-muted-foreground transition-transform ${open ? "rotate-180" : ""}`}
            />
          </CardHeader>
        </CollapsibleTrigger>
        <CollapsibleContent>
          <CardContent className="pt-0">
            <p className="text-sm text-muted-foreground mb-3">
              {t("common.permissions")}:
            </p>
            <RolePermissionsPanel roleCode={role.code} />
          </CardContent>
        </CollapsibleContent>
      </Card>
    </Collapsible>
  );
}

// ─── Page ──────────────────────────────────────────────────────

export default function RolesPage() {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const {
    data: roles,
    isLoading,
    isError,
    error,
    refetch,
  } = useAdminRoles();

  return (
    <>
      <Header />
      <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
        <div className="pt-4">
          <h1 className="text-2xl font-bold tracking-tight">
            {t("roles.title")}
          </h1>
          <p className="text-muted-foreground">{t("roles.description")}</p>
        </div>

        {isError ? (
          <Alert variant="destructive">
            <IconAlertCircle className="size-4" />
            <AlertDescription className="flex items-center justify-between">
              <span>
                {t("roles.loadError")}: {extractErrorMessage(error)}
              </span>
              <button
                onClick={() => refetch()}
                className="underline font-medium ml-2"
              >
                {t("common.retry")}
              </button>
            </AlertDescription>
          </Alert>
        ) : isLoading ? (
          <div className="space-y-3">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} className="h-16 w-full rounded-lg" />
            ))}
          </div>
        ) : !roles?.length ? (
          <div className="flex flex-col items-center justify-center py-16 text-center">
            <IconInbox className="size-12 text-muted-foreground/50 mb-4" />
            <h3 className="text-lg font-medium">{t("roles.noRoles")}</h3>
          </div>
        ) : (
          <div className="space-y-3">
            {roles.map((role) => (
              <RoleCard key={role.code} role={role} />
            ))}
          </div>
        )}
      </div>
    </>
  );
}
