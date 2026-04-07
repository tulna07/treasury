"use client";

import { useState } from "react";
import { useTranslation } from "@/lib/i18n";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
  DialogClose,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";

interface BondApprovalDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description?: string;
  requireReason?: boolean;
  onConfirm: (reason?: string) => void;
  isLoading?: boolean;
  variant?: "default" | "destructive";
}

export function BondApprovalDialog({
  open,
  onOpenChange,
  title,
  description,
  requireReason = true,
  onConfirm,
  isLoading,
  variant = "destructive",
}: BondApprovalDialogProps) {
  const { t } = useTranslation();
  const [reason, setReason] = useState("");

  function handleConfirm() {
    if (requireReason && !reason.trim()) return;
    onConfirm(reason.trim() || undefined);
    setReason("");
    onOpenChange(false);
  }

  function handleCancel() {
    setReason("");
    onOpenChange(false);
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          {description && (
            <DialogDescription>{description}</DialogDescription>
          )}
        </DialogHeader>

        {requireReason && (
          <div className="space-y-2">
            <Label htmlFor="approval-reason">{t("bond.reason")}</Label>
            <Textarea
              id="approval-reason"
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder={t("bond.reasonRequired")}
              rows={3}
            />
          </div>
        )}

        <DialogFooter>
          <DialogClose render={<Button variant="outline" onClick={handleCancel} />}>
            {t("common.cancel")}
          </DialogClose>
          <Button
            variant={variant}
            onClick={handleConfirm}
            disabled={(requireReason && !reason.trim()) || isLoading}
          >
            {t("common.confirm")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
