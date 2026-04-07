"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  IconCheck,
  IconX,
  IconArrowBack,
  IconBan,
  IconCopy,
  IconEdit,
  IconTrash,
} from "@tabler/icons-react";
import { useTranslation } from "@/lib/i18n";
import { useAbility } from "@/hooks/use-ability";
import type { BondDeal, BondStatus } from "@/hooks/use-bonds";

interface BondActionButtonsProps {
  deal: BondDeal;
  onApprove?: (reason?: string) => void;
  onReject?: (reason: string) => void;
  onRecall?: (reason: string) => void;
  onCancel?: (reason: string) => void;
  onCancelApprove?: (reason?: string) => void;
  onCancelReject?: (reason: string) => void;
  onClone?: () => void;
  onEdit?: () => void;
  onDelete?: () => void;
  isLoading?: boolean;
}

const APPROVABLE_STATUSES: BondStatus[] = ["PENDING_L2_APPROVAL", "PENDING_BOOKING", "PENDING_CHIEF_ACCOUNTANT"];
const RECALLABLE_STATUSES: BondStatus[] = ["PENDING_L2_APPROVAL", "PENDING_BOOKING", "PENDING_CHIEF_ACCOUNTANT"];
const CANCELLABLE_STATUSES: BondStatus[] = ["COMPLETED"];
const CANCEL_APPROVABLE_STATUSES: BondStatus[] = ["PENDING_CANCEL_L1", "PENDING_CANCEL_L2"];
const EDITABLE_STATUSES: BondStatus[] = ["OPEN"];
const DELETABLE_STATUSES: BondStatus[] = ["OPEN"];

type ReasonAction = "reject" | "recall" | "cancel" | "delete" | "cancelReject" | null;

export function BondActionButtons({
  deal,
  onApprove,
  onReject,
  onRecall,
  onCancel,
  onCancelApprove,
  onCancelReject,
  onClone,
  onEdit,
  onDelete,
  isLoading,
}: BondActionButtonsProps) {
  const { t } = useTranslation();
  const ability = useAbility();
  const [activeAction, setActiveAction] = useState<ReasonAction>(null);
  const [reason, setReason] = useState("");

  const canApprove = ability.can("approve", "GTCGTransaction");
  const canUpdate = ability.can("update", "GTCGTransaction");
  const canCancel = ability.can("cancel", "GTCGTransaction");
  const canRecall = ability.can("recall", "GTCGTransaction");
  const canDeletePerm = ability.can("delete", "GTCGTransaction") || ability.can("update", "GTCGTransaction");

  const isApprovable = APPROVABLE_STATUSES.includes(deal.status);
  const isRecallable = RECALLABLE_STATUSES.includes(deal.status);
  const isCancellable = CANCELLABLE_STATUSES.includes(deal.status);
  const isCancelApprovable = CANCEL_APPROVABLE_STATUSES.includes(deal.status);
  const isEditable = EDITABLE_STATUSES.includes(deal.status);
  const isDeletable = DELETABLE_STATUSES.includes(deal.status);

  function handleConfirm() {
    const trimmedReason = reason.trim();
    switch (activeAction) {
      case "reject":
        if (trimmedReason) onReject?.(trimmedReason);
        break;
      case "recall":
        if (trimmedReason) onRecall?.(trimmedReason);
        break;
      case "cancel":
        if (trimmedReason) onCancel?.(trimmedReason);
        break;
      case "cancelReject":
        if (trimmedReason) onCancelReject?.(trimmedReason);
        break;
      case "delete":
        onDelete?.();
        break;
    }
    setActiveAction(null);
    setReason("");
  }

  function cancelAction() {
    setActiveAction(null);
    setReason("");
  }

  const needsReason = activeAction === "reject" || activeAction === "recall" || activeAction === "cancel" || activeAction === "cancelReject";

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap gap-2">
        {canApprove && isApprovable && onApprove && (
          <Button size="sm" onClick={() => onApprove()} disabled={isLoading}>
            <IconCheck className="mr-1.5 size-4" />
            {t("bond.action.approve")}
          </Button>
        )}

        {canApprove && isApprovable && onReject && (
          <Button
            size="sm"
            variant="destructive"
            disabled={isLoading}
            onClick={() => setActiveAction("reject")}
          >
            <IconX className="mr-1.5 size-4" />
            {t("bond.action.reject")}
          </Button>
        )}

        {isCancelApprovable && onCancelApprove && (
          <Button size="sm" onClick={() => onCancelApprove()} disabled={isLoading}>
            <IconCheck className="mr-1.5 size-4" />
            {t("bond.action.cancelApprove")}
          </Button>
        )}

        {isCancelApprovable && onCancelReject && (
          <Button
            size="sm"
            variant="destructive"
            disabled={isLoading}
            onClick={() => setActiveAction("cancelReject")}
          >
            <IconX className="mr-1.5 size-4" />
            {t("bond.action.cancelReject")}
          </Button>
        )}

        {canRecall && isRecallable && onRecall && (
          <Button
            size="sm"
            variant="outline"
            disabled={isLoading}
            onClick={() => setActiveAction("recall")}
          >
            <IconArrowBack className="mr-1.5 size-4" />
            {t("bond.action.recall")}
          </Button>
        )}

        {canCancel && isCancellable && onCancel && (
          <Button
            size="sm"
            variant="outline"
            disabled={isLoading}
            onClick={() => setActiveAction("cancel")}
          >
            <IconBan className="mr-1.5 size-4" />
            {t("bond.action.cancel")}
          </Button>
        )}

        {canUpdate && isEditable && onEdit && (
          <Button size="sm" variant="outline" onClick={onEdit} disabled={isLoading}>
            <IconEdit className="mr-1.5 size-4" />
            {t("bond.action.edit")}
          </Button>
        )}

        {onClone && (
          <Button size="sm" variant="outline" onClick={onClone} disabled={isLoading}>
            <IconCopy className="mr-1.5 size-4" />
            {t("bond.action.clone")}
          </Button>
        )}

        {canDeletePerm && isDeletable && onDelete && (
          <Button
            size="sm"
            variant="destructive"
            disabled={isLoading}
            onClick={() => setActiveAction("delete")}
          >
            <IconTrash className="mr-1.5 size-4" />
            {t("bond.action.delete")}
          </Button>
        )}
      </div>

      {activeAction && (
        <Card className="border-destructive/50">
          <CardHeader className="pb-3">
            <CardTitle className="text-sm">
              {activeAction === "delete"
                ? t("bond.confirmDelete")
                : activeAction === "cancel"
                  ? t("bond.confirmCancel")
                  : t("bond.reasonRequired")}
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {needsReason && (
              <div>
                <Label htmlFor="action-reason">{t("bond.reason")}</Label>
                <Textarea
                  id="action-reason"
                  value={reason}
                  onChange={(e) => setReason(e.target.value)}
                  placeholder={t("bond.reasonRequired")}
                  className="mt-1.5"
                  rows={2}
                />
              </div>
            )}
            <div className="flex gap-2 justify-end">
              <Button variant="outline" size="sm" onClick={cancelAction}>
                {t("common.cancel")}
              </Button>
              <Button
                size="sm"
                variant="destructive"
                onClick={handleConfirm}
                disabled={(needsReason && !reason.trim()) || isLoading}
              >
                {t("common.confirm")}
              </Button>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
