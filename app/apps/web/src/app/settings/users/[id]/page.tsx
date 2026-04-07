"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { Header } from "@/components/header";
import { useTranslation } from "@/lib/i18n";
import { useAbility } from "@/hooks/use-ability";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Separator } from "@/components/ui/separator";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  IconArrowLeft,
  IconAlertCircle,
  IconLock,
  IconLockOpen,
  IconKey,
  IconEdit,
  IconCopy,
  IconCheck,
  IconShieldPlus,
  IconTrash,
} from "@tabler/icons-react";
import { formatDate, extractErrorMessage } from "@/lib/utils";
import { toast } from "sonner";
import {
  useAdminUser,
  useLockUser,
  useUnlockUser,
  useResetPassword,
  useAssignRole,
  useRevokeRole,
} from "@/hooks/use-admin-users";
import { useAdminRoles } from "@/hooks/use-admin-roles";
import { UserStatusBadge } from "../components/user-status-badge";

export default function UserDetailPage() {
  const { t } = useTranslation();
  const router = useRouter();
  const ability = useAbility();
  const params = useParams();
  const id = params.id as string;

  const { data: user, isLoading, isError, error } = useAdminUser(id);
  const { data: allRoles } = useAdminRoles();

  const lockMutation = useLockUser(id);
  const unlockMutation = useUnlockUser(id);
  const resetMutation = useResetPassword(id);
  const assignRoleMutation = useAssignRole(id);
  const revokeRoleMutation = useRevokeRole(id);

  // Inline sections
  const [lockOpen, setLockOpen] = useState(false);
  const [lockReason, setLockReason] = useState("");
  const [unlockOpen, setUnlockOpen] = useState(false);
  const [unlockReason, setUnlockReason] = useState("");
  const [resetOpen, setResetOpen] = useState(false);
  const [resetReason, setResetReason] = useState("");
  const [tempPassword, setTempPassword] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  // Role assignment
  const [assignOpen, setAssignOpen] = useState(false);
  const [selectedRole, setSelectedRole] = useState("");
  const [assignReason, setAssignReason] = useState("");
  const [revokeReason, setRevokeReason] = useState("");
  const [revokingRole, setRevokingRole] = useState<string | null>(null);

  if (isLoading) {
    return (
      <>
        <Header />
        <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
          <div className="pt-4 space-y-4">
            <Skeleton className="h-8 w-64" />
            <Skeleton className="h-48 w-full" />
            <Skeleton className="h-32 w-full" />
          </div>
        </div>
      </>
    );
  }

  if (isError || !user) {
    return (
      <>
        <Header />
        <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
          <div className="pt-4">
            <Button variant="ghost" onClick={() => router.push("/settings/users")}>
              <IconArrowLeft className="mr-2 size-4" />
              {t("users.backToList")}
            </Button>
          </div>
          <Alert variant="destructive">
            <IconAlertCircle className="size-4" />
            <AlertDescription>
              {t("users.loadError")}: {extractErrorMessage(error)}
            </AlertDescription>
          </Alert>
        </div>
      </>
    );
  }

  function handleCopy() {
    if (tempPassword) {
      navigator.clipboard.writeText(tempPassword);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  }

  const availableRoles = allRoles?.filter(
    (r) => !user.roles.includes(r.code)
  );

  return (
    <>
      <Header />
      <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
        {/* Header */}
        <div className="flex flex-col gap-4 pt-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex items-center gap-4">
            <Button variant="ghost" size="icon" onClick={() => router.push("/settings/users")}>
              <IconArrowLeft className="size-4" />
            </Button>
            <div>
              <div className="flex items-center gap-3">
                <h1 className="text-2xl font-bold tracking-tight">
                  {user.full_name}
                </h1>
                <UserStatusBadge isActive={user.is_active} />
              </div>
              <p className="text-muted-foreground">@{user.username}</p>
            </div>
          </div>

          {ability.can("manage", "Settings") && (
            <div className="flex flex-wrap gap-2">
              <Button
                variant="outline"
                onClick={() => router.push(`/settings/users/${id}/edit`)}
              >
                <IconEdit className="mr-2 size-4" />
                {t("common.edit")}
              </Button>
            </div>
          )}
        </div>

        {/* Profile card */}
        <div className="grid gap-4 md:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">{t("users.profile")}</CardTitle>
            </CardHeader>
            <CardContent>
              <dl className="grid grid-cols-2 gap-x-4 gap-y-3 text-sm">
                <div>
                  <dt className="text-muted-foreground">{t("common.name")}</dt>
                  <dd className="font-medium">{user.full_name}</dd>
                </div>
                <div>
                  <dt className="text-muted-foreground">{t("common.username")}</dt>
                  <dd className="font-mono font-medium">{user.username}</dd>
                </div>
                <div>
                  <dt className="text-muted-foreground">{t("common.email")}</dt>
                  <dd className="font-medium break-all">{user.email}</dd>
                </div>
                <div>
                  <dt className="text-muted-foreground">{t("common.department")}</dt>
                  <dd className="font-medium">{user.department || "-"}</dd>
                </div>
                <div>
                  <dt className="text-muted-foreground">{t("common.position")}</dt>
                  <dd className="font-medium">{user.position || "-"}</dd>
                </div>
                <div>
                  <dt className="text-muted-foreground">{t("common.branch")}</dt>
                  <dd className="font-medium">{user.branch_name || "-"}</dd>
                </div>
                <div>
                  <dt className="text-muted-foreground">{t("common.lastLogin")}</dt>
                  <dd className="font-medium">
                    {user.last_login_at ? formatDate(user.last_login_at) : t("common.never")}
                  </dd>
                </div>
                <div>
                  <dt className="text-muted-foreground">{t("common.createdAt")}</dt>
                  <dd className="font-medium">{formatDate(user.created_at)}</dd>
                </div>
              </dl>
            </CardContent>
          </Card>

          {/* Roles card */}
          <Card>
            <CardHeader className="flex flex-row items-center justify-between">
              <CardTitle className="text-base">{t("common.roles")}</CardTitle>
              {ability.can("manage", "Settings") && availableRoles && availableRoles.length > 0 && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setAssignOpen(!assignOpen)}
                >
                  <IconShieldPlus className="mr-2 size-4" />
                  {t("users.assignRole")}
                </Button>
              )}
            </CardHeader>
            <CardContent className="space-y-4">
              {/* Assigned roles */}
              <div className="flex flex-wrap gap-2">
                {user.roles.length === 0 ? (
                  <span className="text-sm text-muted-foreground">-</span>
                ) : (
                  user.role_names?.map((name, i) => (
                    <div key={user.roles[i]} className="flex items-center gap-1">
                      <Badge
                        variant="secondary"
                        className="bg-blue-500/10 text-blue-700 dark:text-blue-400 border-blue-500/30"
                      >
                        {name}
                      </Badge>
                      {ability.can("manage", "Settings") && (
                        revokingRole === user.roles[i] ? (
                          <div className="flex items-center gap-1 ml-1">
                            <Input
                              placeholder={t("users.revokeReason")}
                              value={revokeReason}
                              onChange={(e) => setRevokeReason(e.target.value)}
                              className="h-7 w-40 text-xs"
                            />
                            <Button
                              size="sm"
                              variant="destructive"
                              className="h-7 px-2 text-xs"
                              disabled={!revokeReason.trim() || revokeRoleMutation.isPending}
                              onClick={() => {
                                revokeRoleMutation.mutate(
                                  { roleCode: user.roles[i], reason: revokeReason },
                                  {
                                    onSuccess: () => {
                                      toast.success(t("users.revokeSuccess"));
                                      setRevokingRole(null);
                                      setRevokeReason("");
                                    },
                                    onError: (err) =>
                                      toast.error(extractErrorMessage(err)),
                                  }
                                );
                              }}
                            >
                              {t("common.confirm")}
                            </Button>
                            <Button
                              size="sm"
                              variant="ghost"
                              className="h-7 px-2 text-xs"
                              onClick={() => { setRevokingRole(null); setRevokeReason(""); }}
                            >
                              {t("common.cancel")}
                            </Button>
                          </div>
                        ) : (
                          <button
                            className="text-muted-foreground hover:text-destructive transition-colors"
                            onClick={() => setRevokingRole(user.roles[i])}
                            title={t("users.revokeRole")}
                          >
                            <IconTrash className="size-3.5" />
                          </button>
                        )
                      )}
                    </div>
                  ))
                )}
              </div>

              {/* Assign role inline */}
              {assignOpen && (
                <>
                  <Separator />
                  <div className="space-y-3">
                    <Label>{t("users.assignRole")}</Label>
                    <Select value={selectedRole} onValueChange={(v) => setSelectedRole(v ?? "")}>
                      <SelectTrigger>
                        <SelectValue placeholder={t("users.selectRole")}>
                          {(value: string) => {
                            const r = availableRoles?.find((role) => role.code === value);
                            return r ? `${r.name} (${r.code})` : value;
                          }}
                        </SelectValue>
                      </SelectTrigger>
                      <SelectContent>
                        {availableRoles?.map((r) => (
                          <SelectItem key={r.code} value={r.code} label={`${r.name} (${r.code})`}>
                            {r.name} ({r.code})
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <Input
                      placeholder={t("users.assignReason")}
                      value={assignReason}
                      onChange={(e) => setAssignReason(e.target.value)}
                    />
                    <div className="flex gap-2">
                      <Button
                        size="sm"
                        disabled={!selectedRole || !assignReason.trim() || assignRoleMutation.isPending}
                        onClick={() => {
                          assignRoleMutation.mutate(
                            { role_code: selectedRole, reason: assignReason },
                            {
                              onSuccess: () => {
                                toast.success(t("users.assignSuccess"));
                                setAssignOpen(false);
                                setSelectedRole("");
                                setAssignReason("");
                              },
                              onError: (err) =>
                                toast.error(extractErrorMessage(err)),
                            }
                          );
                        }}
                      >
                        {t("common.confirm")}
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => {
                          setAssignOpen(false);
                          setSelectedRole("");
                          setAssignReason("");
                        }}
                      >
                        {t("common.cancel")}
                      </Button>
                    </div>
                  </div>
                </>
              )}
            </CardContent>
          </Card>
        </div>

        {/* Quick Actions — Dialog-based */}
        {ability.can("manage", "Settings") && (
          <Card>
            <CardHeader>
              <CardTitle className="text-base">{t("common.actions")}</CardTitle>
            </CardHeader>
            <CardContent className="flex flex-wrap gap-2">
              {/* Lock / Unlock */}
              {user.is_active ? (
                <Dialog open={lockOpen} onOpenChange={(open) => { setLockOpen(open); if (!open) setLockReason(""); }}>
                  <DialogTrigger render={<Button variant="destructive" size="sm" />}>
                    <IconLock className="size-4 mr-1.5" />
                    {t("users.lock")}
                  </DialogTrigger>
                  <DialogContent className="sm:max-w-md">
                    <DialogHeader>
                      <DialogTitle>{t("users.lock")}</DialogTitle>
                      <DialogDescription>{t("users.lockDesc") || `Khóa tài khoản "${user.full_name}". Người dùng sẽ không thể đăng nhập.`}</DialogDescription>
                    </DialogHeader>
                    <div className="space-y-2">
                      <Label>{t("users.lockReason")} *</Label>
                      <Input
                        value={lockReason}
                        onChange={(e) => setLockReason(e.target.value)}
                        placeholder={t("users.lockReason")}
                        autoFocus
                      />
                    </div>
                    <DialogFooter>
                      <Button variant="outline" onClick={() => setLockOpen(false)}>
                        {t("common.cancel")}
                      </Button>
                      <Button
                        variant="destructive"
                        disabled={!lockReason.trim() || lockMutation.isPending}
                        onClick={() => {
                          lockMutation.mutate(
                            { reason: lockReason },
                            {
                              onSuccess: () => {
                                toast.success(t("users.lockSuccess"));
                                setLockOpen(false);
                                setLockReason("");
                              },
                              onError: (err) =>
                                toast.error(extractErrorMessage(err)),
                            }
                          );
                        }}
                      >
                        {lockMutation.isPending ? t("common.loading") : t("common.confirm")}
                      </Button>
                    </DialogFooter>
                  </DialogContent>
                </Dialog>
              ) : (
                <Dialog open={unlockOpen} onOpenChange={(open) => { setUnlockOpen(open); if (!open) setUnlockReason(""); }}>
                  <DialogTrigger render={<Button variant="outline" size="sm" />}>
                    <IconLockOpen className="size-4 mr-1.5" />
                    {t("users.unlock")}
                  </DialogTrigger>
                  <DialogContent className="sm:max-w-md">
                    <DialogHeader>
                      <DialogTitle>{t("users.unlock")}</DialogTitle>
                      <DialogDescription>{t("users.unlockDesc") || `Mở khóa tài khoản "${user.full_name}".`}</DialogDescription>
                    </DialogHeader>
                    <div className="space-y-2">
                      <Label>{t("users.unlockReason")} *</Label>
                      <Input
                        value={unlockReason}
                        onChange={(e) => setUnlockReason(e.target.value)}
                        placeholder={t("users.unlockReason")}
                        autoFocus
                      />
                    </div>
                    <DialogFooter>
                      <Button variant="outline" onClick={() => setUnlockOpen(false)}>
                        {t("common.cancel")}
                      </Button>
                      <Button
                        disabled={!unlockReason.trim() || unlockMutation.isPending}
                        onClick={() => {
                          unlockMutation.mutate(
                            { reason: unlockReason },
                            {
                              onSuccess: () => {
                                toast.success(t("users.unlockSuccess"));
                                setUnlockOpen(false);
                                setUnlockReason("");
                              },
                              onError: (err) =>
                                toast.error(extractErrorMessage(err)),
                            }
                          );
                        }}
                      >
                        {unlockMutation.isPending ? t("common.loading") : t("common.confirm")}
                      </Button>
                    </DialogFooter>
                  </DialogContent>
                </Dialog>
              )}

              {/* Reset Password */}
              <Dialog open={resetOpen} onOpenChange={(open) => { setResetOpen(open); if (!open) { setResetReason(""); setTempPassword(""); } }}>
                <DialogTrigger render={<Button variant="outline" size="sm" />}>
                  <IconKey className="size-4 mr-1.5" />
                  {t("users.resetPassword")}
                </DialogTrigger>
                <DialogContent className="sm:max-w-md">
                  <DialogHeader>
                    <DialogTitle>{t("users.resetPassword")}</DialogTitle>
                    <DialogDescription>{t("users.resetDesc") || `Đặt lại mật khẩu cho "${user.full_name}". Mật khẩu tạm sẽ được tạo tự động.`}</DialogDescription>
                  </DialogHeader>
                  {tempPassword ? (
                    <Alert className="border-amber-500/50 bg-amber-500/10">
                      <IconKey className="size-4 text-amber-600" />
                      <AlertTitle className="text-amber-700 dark:text-amber-400">
                        {t("users.tempPassword")}
                      </AlertTitle>
                      <AlertDescription className="space-y-2">
                        <div className="flex items-center gap-2">
                          <code className="rounded bg-muted px-3 py-1.5 font-mono text-lg font-bold tracking-wider">
                            {tempPassword}
                          </code>
                          <Button variant="outline" size="sm" onClick={handleCopy}>
                            {copied ? <IconCheck className="size-4 text-green-500" /> : <IconCopy className="size-4" />}
                            {copied ? t("common.copied") : t("common.copy")}
                          </Button>
                        </div>
                        <p className="text-sm text-amber-700/80 dark:text-amber-400/80">
                          {t("users.tempPasswordNote")}
                        </p>
                      </AlertDescription>
                    </Alert>
                  ) : (
                    <div className="space-y-2">
                      <Label>{t("users.resetReason")} *</Label>
                      <Input
                        value={resetReason}
                        onChange={(e) => setResetReason(e.target.value)}
                        placeholder={t("users.resetReason")}
                        autoFocus
                      />
                    </div>
                  )}
                  <DialogFooter>
                    {tempPassword ? (
                      <Button onClick={() => setResetOpen(false)}>{t("common.close")}</Button>
                    ) : (
                      <>
                        <Button variant="outline" onClick={() => setResetOpen(false)}>
                          {t("common.cancel")}
                        </Button>
                        <Button
                          disabled={!resetReason.trim() || resetMutation.isPending}
                          onClick={() => {
                            resetMutation.mutate(
                              { reason: resetReason },
                              {
                                onSuccess: (res) => {
                                  toast.success(t("users.resetSuccess"));
                                  setTempPassword(res.temp_password);
                                  setResetReason("");
                                },
                                onError: (err) =>
                                  toast.error(extractErrorMessage(err)),
                              }
                            );
                          }}
                        >
                          {resetMutation.isPending ? t("common.loading") : t("common.confirm")}
                        </Button>
                      </>
                    )}
                  </DialogFooter>
                </DialogContent>
              </Dialog>
            </CardContent>
          </Card>
        )}
      </div>
    </>
  );
}
