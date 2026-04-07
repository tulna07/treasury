"use client";

import { useState, useMemo } from "react";
import { useTranslation } from "@/lib/i18n";
import { useExportMMInterbankDeals } from "@/hooks/use-mm";
import { toast } from "sonner";
import { extractErrorMessage } from "@/lib/utils";
import {
  Dialog,
  DialogTrigger,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
  DialogClose,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { DatePicker } from "@/components/ui/date-picker";
import {
  IconFileExport,
  IconLock,
  IconEye,
  IconEyeOff,
  IconCheck,
  IconX,
  IconDownload,
} from "@tabler/icons-react";

function getPasswordStrength(password: string): number {
  if (password.length === 0) return 0;
  if (password.length < 8) return 1;

  const hasUpper = /[A-Z]/.test(password);
  const hasLower = /[a-z]/.test(password);
  const hasDigit = /\d/.test(password);
  const hasSpecial = /[^A-Za-z0-9]/.test(password);

  const met = [hasUpper, hasLower, hasDigit].filter(Boolean).length;

  if (met < 3) return 2;
  if (hasSpecial) return 4;
  return 3;
}

const strengthColors = ["", "bg-red-500", "bg-orange-500", "bg-yellow-500", "bg-green-500"];

export function MMExportDialog() {
  const { t } = useTranslation();
  const exportMutation = useExportMMInterbankDeals();

  const [open, setOpen] = useState(false);
  const currentYear = new Date().getFullYear();
  const [fromDate, setFromDate] = useState(`${currentYear}-01-01`);
  const [toDate, setToDate] = useState(`${currentYear}-12-31`);
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);

  const strength = useMemo(() => getPasswordStrength(password), [password]);
  const hasUpper = /[A-Z]/.test(password);
  const hasLower = /[a-z]/.test(password);
  const hasDigit = /\d/.test(password);
  const hasMinLength = password.length >= 8;
  const passwordsMatch = password.length > 0 && password === confirmPassword;
  const dateValid = fromDate && toDate && fromDate < toDate;
  const passwordValid = hasMinLength && hasUpper && hasLower && hasDigit && passwordsMatch;
  const canSubmit = dateValid && passwordValid && !exportMutation.isPending;

  const strengthLabel =
    strength <= 1
      ? t("bond.export.strength.weak")
      : strength === 2
        ? t("bond.export.strength.fair")
        : strength === 3
          ? t("bond.export.strength.good")
          : t("bond.export.strength.strong");

  function handleReset() {
    setFromDate(`${currentYear}-01-01`);
    setToDate(`${currentYear}-12-31`);
    setPassword("");
    setConfirmPassword("");
    setShowPassword(false);
    setShowConfirm(false);
  }

  function handleSubmit() {
    if (!canSubmit) return;
    exportMutation.mutate(
      { from: fromDate, to: toDate, password },
      {
        onSuccess: () => {
          toast.success(t("mm.export.success"));
          setOpen(false);
          handleReset();
        },
        onError: (err) => {
          toast.error(extractErrorMessage(err));
        },
      }
    );
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(isOpen) => {
        setOpen(isOpen);
        if (!isOpen) handleReset();
      }}
    >
      <DialogTrigger
        render={
          <Button variant="outline" className="shrink-0">
            <IconFileExport className="mr-2 size-4" />
            {t("mm.export")}
          </Button>
        }
      />
      <DialogContent className="sm:max-w-md">
        <DialogHeader className="-mx-4 -mt-4 rounded-t-xl bg-gradient-to-r from-slate-800 to-slate-900 p-4 text-white dark:from-slate-900 dark:to-slate-950">
          <div className="flex items-center gap-3">
            <div className="flex size-10 items-center justify-center rounded-lg bg-white/10">
              <IconFileExport className="size-5" />
            </div>
            <div>
              <DialogTitle className="text-white">
                {t("mm.export.title")}
              </DialogTitle>
              <DialogDescription className="text-slate-300">
                {t("mm.export.subtitle")}
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <div className="space-y-3">
          <Label className="text-sm font-medium">{t("bond.export.dateRange")}</Label>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1">
              <Label className="text-xs text-muted-foreground">{t("bond.fromDate")}</Label>
              <DatePicker
                value={fromDate}
                onChange={(val) => setFromDate(val)}
                placeholder={t("bond.fromDate")}
              />
            </div>
            <div className="space-y-1">
              <Label className="text-xs text-muted-foreground">{t("bond.toDate")}</Label>
              <DatePicker
                value={toDate}
                onChange={(val) => setToDate(val)}
                placeholder={t("bond.toDate")}
              />
            </div>
          </div>
        </div>

        <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 dark:border-amber-900 dark:bg-amber-950/30">
          <div className="mb-3 flex items-center gap-2">
            <IconLock className="size-4 text-amber-600 dark:text-amber-400" />
            <span className="text-sm font-medium text-amber-800 dark:text-amber-300">
              {t("bond.export.password")}
            </span>
          </div>
          <p className="mb-3 text-xs text-amber-700 dark:text-amber-400">
            {t("bond.export.passwordHint")}
          </p>

          <div className="space-y-3">
            <div className="relative">
              <Input
                type={showPassword ? "text" : "password"}
                placeholder={t("bond.export.passwordPlaceholder")}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="pr-10 bg-white dark:bg-slate-900"
              />
              <button
                type="button"
                onClick={() => setShowPassword(!showPassword)}
                className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
              >
                {showPassword ? (
                  <IconEyeOff className="size-4" />
                ) : (
                  <IconEye className="size-4" />
                )}
              </button>
            </div>

            {password.length > 0 && (
              <div className="space-y-1">
                <div className="flex gap-1">
                  {[1, 2, 3, 4].map((level) => (
                    <div
                      key={level}
                      className={`h-1.5 flex-1 rounded-full transition-colors ${
                        level <= strength
                          ? strengthColors[strength]
                          : "bg-slate-200 dark:bg-slate-700"
                      }`}
                    />
                  ))}
                </div>
                <p className="text-xs text-muted-foreground">{strengthLabel}</p>
              </div>
            )}

            <div className="relative">
              <Input
                type={showConfirm ? "text" : "password"}
                placeholder={t("bond.export.confirmPlaceholder")}
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                className="pr-10 bg-white dark:bg-slate-900"
              />
              <button
                type="button"
                onClick={() => setShowConfirm(!showConfirm)}
                className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
              >
                {showConfirm ? (
                  <IconEyeOff className="size-4" />
                ) : (
                  <IconEye className="size-4" />
                )}
              </button>
            </div>

            {confirmPassword.length > 0 && (
              <div className="flex items-center gap-1.5">
                {passwordsMatch ? (
                  <>
                    <IconCheck className="size-3.5 text-green-600" />
                    <span className="text-xs text-green-600">{t("bond.export.passwordMatch")}</span>
                  </>
                ) : (
                  <>
                    <IconX className="size-3.5 text-red-500" />
                    <span className="text-xs text-red-500">{t("bond.export.passwordMismatch")}</span>
                  </>
                )}
              </div>
            )}

            <div className="space-y-1">
              <p className="text-xs font-medium text-muted-foreground">
                {t("bond.export.requirements")}
              </p>
              <ul className="space-y-0.5">
                {[
                  { met: hasMinLength, label: t("bond.export.req.minLength") },
                  { met: hasUpper && hasLower, label: t("bond.export.req.cases") },
                  { met: hasDigit, label: t("bond.export.req.digit") },
                ].map((req) => (
                  <li key={req.label} className="flex items-center gap-1.5">
                    {req.met ? (
                      <IconCheck className="size-3 text-green-600" />
                    ) : (
                      <IconX className="size-3 text-muted-foreground" />
                    )}
                    <span
                      className={`text-xs ${
                        req.met ? "text-green-600" : "text-muted-foreground"
                      }`}
                    >
                      {req.label}
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          </div>
        </div>

        <DialogFooter>
          <p className="mr-auto text-xs text-muted-foreground">
            {t("bond.export.auditNote")}
          </p>
          <DialogClose render={<Button variant="outline" />}>
            {t("common.cancel")}
          </DialogClose>
          <Button onClick={handleSubmit} disabled={!canSubmit}>
            {exportMutation.isPending ? (
              t("bond.export.exporting")
            ) : (
              <>
                <IconDownload className="mr-2 size-4" />
                {t("bond.export.submit")}
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
