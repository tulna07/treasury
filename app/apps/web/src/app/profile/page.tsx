"use client";

import { useState } from "react";
import { Header } from "@/components/header";
import { useTranslation } from "@/lib/i18n";
import { useAuthStore } from "@/lib/auth-store";
import { api } from "@/lib/api";
import { toast } from "sonner";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Separator } from "@/components/ui/separator";
import {
  IconUser,
  IconLock,
  IconMail,
  IconBuilding,
  IconShieldCheck,
} from "@tabler/icons-react";

function getInitials(name: string): string {
  return name
    .split(" ")
    .map((w) => w[0])
    .filter(Boolean)
    .slice(0, 2)
    .join("")
    .toUpperCase();
}

export default function ProfilePage() {
  const { t } = useTranslation();
  const user = useAuthStore((s) => s.user);

  // Tab 1: editable fields
  const [name, setName] = useState(user?.name ?? "");
  const [email, setEmail] = useState(user?.email ?? "");

  // Tab 2: change password
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [passwordError, setPasswordError] = useState("");
  const [isChangingPassword, setIsChangingPassword] = useState(false);

  const handleSaveInfo = () => {
    toast.info(t("profile.comingSoon"));
  };

  const handleChangePassword = async () => {
    setPasswordError("");

    if (!currentPassword) {
      setPasswordError(t("profile.currentPassword"));
      return;
    }
    if (newPassword.length < 8) {
      setPasswordError(t("profile.passwordMinLength"));
      return;
    }
    if (newPassword !== confirmPassword) {
      setPasswordError(t("profile.passwordMismatch"));
      return;
    }

    setIsChangingPassword(true);
    try {
      await api.post("/auth/password", {
        current_password: currentPassword,
        new_password: newPassword,
      });
      toast.success(t("profile.passwordChanged"));
      setCurrentPassword("");
      setNewPassword("");
      setConfirmPassword("");
    } catch (err: unknown) {
      const message =
        (err as { error?: string })?.error || "Error";
      setPasswordError(message);
    } finally {
      setIsChangingPassword(false);
    }
  };

  if (!user) return null;

  return (
    <>
      <Header />
      <div className="flex-1 space-y-6 p-6">
        {/* Page header */}
        <div>
          <h1 className="text-2xl font-bold tracking-tight">
            {t("profile.title")}
          </h1>
          <p className="text-muted-foreground">{t("profile.description")}</p>
        </div>

        <Tabs defaultValue="info" className="space-y-6">
          <TabsList>
            <TabsTrigger value="info" className="gap-2">
              <IconUser className="h-4 w-4" />
              {t("profile.tab.info")}
            </TabsTrigger>
            <TabsTrigger value="password" className="gap-2">
              <IconLock className="h-4 w-4" />
              {t("profile.tab.password")}
            </TabsTrigger>
          </TabsList>

          {/* ─── Tab 1: Personal Info ─── */}
          <TabsContent value="info">
            <div className="grid gap-6 md:grid-cols-3">
              {/* Profile card */}
              <Card className="md:col-span-1">
                <CardContent className="flex flex-col items-center gap-4 pt-6">
                  <Avatar className="h-24 w-24 text-2xl">
                    <AvatarFallback className="bg-primary text-primary-foreground">
                      {getInitials(user.name)}
                    </AvatarFallback>
                  </Avatar>
                  <div className="text-center">
                    <p className="text-lg font-semibold">{user.name}</p>
                    <p className="text-sm text-muted-foreground">
                      {user.roleLabel}
                    </p>
                  </div>
                  <Separator />
                  <div className="w-full space-y-2 text-sm">
                    <div className="flex items-center gap-2 text-muted-foreground">
                      <IconMail className="h-4 w-4" />
                      {user.email}
                    </div>
                    <div className="flex items-center gap-2 text-muted-foreground">
                      <IconBuilding className="h-4 w-4" />
                      {user.branchName}
                    </div>
                    <div className="flex items-center gap-2 text-muted-foreground">
                      <IconShieldCheck className="h-4 w-4" />
                      <Badge variant="secondary">{user.roleLabel}</Badge>
                    </div>
                  </div>
                </CardContent>
              </Card>

              {/* Editable form */}
              <Card className="md:col-span-2">
                <CardHeader>
                  <CardTitle>{t("profile.tab.info")}</CardTitle>
                  <CardDescription>{t("profile.description")}</CardDescription>
                </CardHeader>
                <CardContent className="space-y-6">
                  <div className="grid gap-4 sm:grid-cols-2">
                    {/* Editable: Name */}
                    <div className="space-y-2">
                      <Label htmlFor="name">{t("profile.name")}</Label>
                      <Input
                        id="name"
                        value={name}
                        onChange={(e) => setName(e.target.value)}
                      />
                    </div>

                    {/* Editable: Email */}
                    <div className="space-y-2">
                      <Label htmlFor="email">{t("profile.email")}</Label>
                      <Input
                        id="email"
                        type="email"
                        value={email}
                        onChange={(e) => setEmail(e.target.value)}
                      />
                    </div>

                    {/* Read-only: Username */}
                    <div className="space-y-2">
                      <Label>{t("profile.username")}</Label>
                      <Input value={user.username} disabled />
                    </div>

                    {/* Read-only: Department */}
                    <div className="space-y-2">
                      <Label>{t("profile.department")}</Label>
                      <Input value={user.department} disabled />
                    </div>

                    {/* Read-only: Branch */}
                    <div className="space-y-2">
                      <Label>{t("profile.branch")}</Label>
                      <Input value={user.branchName} disabled />
                    </div>

                    {/* Read-only: Role */}
                    <div className="space-y-2">
                      <Label>{t("profile.role")}</Label>
                      <Input value={user.roleLabel} disabled />
                    </div>

                    {/* Read-only: Permissions count */}
                    <div className="space-y-2">
                      <Label>{t("profile.permissions")}</Label>
                      <Input
                        value={`${user.permissions?.length ?? 0} ${t("profile.permissionCount")}`}
                        disabled
                      />
                    </div>
                  </div>

                  <div className="flex justify-end">
                    <Button onClick={handleSaveInfo}>
                      {t("profile.save")}
                    </Button>
                  </div>
                </CardContent>
              </Card>
            </div>
          </TabsContent>

          {/* ─── Tab 2: Change Password ─── */}
          <TabsContent value="password">
            <Card className="max-w-lg">
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <IconLock className="h-5 w-5" />
                  {t("profile.changePassword")}
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="current-password">
                    {t("profile.currentPassword")}
                  </Label>
                  <Input
                    id="current-password"
                    type="password"
                    value={currentPassword}
                    onChange={(e) => setCurrentPassword(e.target.value)}
                  />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="new-password">
                    {t("profile.newPassword")}
                  </Label>
                  <Input
                    id="new-password"
                    type="password"
                    value={newPassword}
                    onChange={(e) => setNewPassword(e.target.value)}
                  />
                  {newPassword.length > 0 && newPassword.length < 8 && (
                    <p className="text-sm text-destructive">
                      {t("profile.passwordMinLength")}
                    </p>
                  )}
                </div>

                <div className="space-y-2">
                  <Label htmlFor="confirm-password">
                    {t("profile.confirmPassword")}
                  </Label>
                  <Input
                    id="confirm-password"
                    type="password"
                    value={confirmPassword}
                    onChange={(e) => setConfirmPassword(e.target.value)}
                  />
                  {confirmPassword.length > 0 &&
                    confirmPassword !== newPassword && (
                      <p className="text-sm text-destructive">
                        {t("profile.passwordMismatch")}
                      </p>
                    )}
                </div>

                {passwordError && (
                  <p className="text-sm text-destructive">{passwordError}</p>
                )}

                <div className="flex justify-end pt-2">
                  <Button
                    onClick={handleChangePassword}
                    disabled={isChangingPassword}
                  >
                    {isChangingPassword
                      ? t("common.saving")
                      : t("profile.changePassword")}
                  </Button>
                </div>
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </div>
    </>
  );
}
