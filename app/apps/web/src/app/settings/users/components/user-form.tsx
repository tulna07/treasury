"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { IconAlertCircle } from "@tabler/icons-react";
import { useTranslation } from "@/lib/i18n";
import { useBranches } from "@/hooks/use-master-data";

interface UserFormProps {
  initialData?: {
    full_name: string;
    email: string;
    branch_id: string;
    department: string;
    position: string;
    username?: string;
  };
  onSubmit: (data: {
    username?: string;
    password?: string;
    full_name: string;
    email: string;
    branch_id: string;
    department: string;
    position: string;
  }) => void;
  onCancel: () => void;
  isSubmitting: boolean;
  error: string | null;
  submitLabel: string;
  isCreate?: boolean;
}

export function UserForm({
  initialData,
  onSubmit,
  onCancel,
  isSubmitting,
  error,
  submitLabel,
  isCreate = false,
}: UserFormProps) {
  const { t } = useTranslation();
  const { data: branches } = useBranches();

  const [form, setForm] = useState({
    username: initialData?.username ?? "",
    password: "",
    full_name: initialData?.full_name ?? "",
    email: initialData?.email ?? "",
    branch_id: initialData?.branch_id ?? "",
    department: initialData?.department ?? "",
    position: initialData?.position ?? "",
  });

  const [validationError, setValidationError] = useState<string | null>(null);

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setValidationError(null);

    if (isCreate && !form.username.trim()) {
      setValidationError(t("common.username") + " is required");
      return;
    }
    if (isCreate && !form.password.trim()) {
      setValidationError(t("common.password") + " is required");
      return;
    }
    if (!form.full_name.trim()) {
      setValidationError(t("common.name") + " is required");
      return;
    }
    if (!form.email.trim()) {
      setValidationError(t("common.email") + " is required");
      return;
    }

    onSubmit(form);
  }

  function update(field: string, value: string) {
    setForm((prev) => ({ ...prev, [field]: value }));
  }

  return (
    <form onSubmit={handleSubmit}>
      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("users.profile")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {(error || validationError) && (
            <Alert variant="destructive">
              <IconAlertCircle className="size-4" />
              <AlertDescription>{error || validationError}</AlertDescription>
            </Alert>
          )}

          {isCreate && (
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="username">{t("common.username")} *</Label>
                <Input
                  id="username"
                  value={form.username}
                  onChange={(e) => update("username", e.target.value)}
                  placeholder="e.g. dealer01"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="password">{t("common.password")} *</Label>
                <Input
                  id="password"
                  type="password"
                  value={form.password}
                  onChange={(e) => update("password", e.target.value)}
                  placeholder="********"
                />
              </div>
            </div>
          )}

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="full_name">{t("common.name")} *</Label>
              <Input
                id="full_name"
                value={form.full_name}
                onChange={(e) => update("full_name", e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="email">{t("common.email")} *</Label>
              <Input
                id="email"
                type="email"
                value={form.email}
                onChange={(e) => update("email", e.target.value)}
              />
            </div>
          </div>

          <div className="grid gap-4 sm:grid-cols-3">
            <div className="space-y-2">
              <Label htmlFor="branch_id">{t("common.branch")}</Label>
              <Select
                value={form.branch_id}
                onValueChange={(v) => update("branch_id", v ?? "")}
              >
                <SelectTrigger>
                  <SelectValue placeholder={t("common.branch")}>
                    {(value: string) => branches?.find((b) => b.id === value)?.name ?? value}
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  {branches?.map((b) => (
                    <SelectItem key={b.id} value={b.id} label={b.name}>
                      {b.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="department">{t("common.department")}</Label>
              <Input
                id="department"
                value={form.department}
                onChange={(e) => update("department", e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="position">{t("common.position")}</Label>
              <Input
                id="position"
                value={form.position}
                onChange={(e) => update("position", e.target.value)}
              />
            </div>
          </div>

          <div className="flex justify-end gap-2 pt-4">
            <Button
              type="button"
              variant="outline"
              onClick={onCancel}
              disabled={isSubmitting}
            >
              {t("common.cancel")}
            </Button>
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting ? t("common.loading") : submitLabel}
            </Button>
          </div>
        </CardContent>
      </Card>
    </form>
  );
}
