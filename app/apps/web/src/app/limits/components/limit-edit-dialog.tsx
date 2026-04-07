"use client";

import { useState, useEffect } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogClose,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import { Textarea } from "@/components/ui/textarea";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { IconAlertCircle } from "@tabler/icons-react";
import { useTranslation } from "@/lib/i18n";
import { useUpdateCreditLimit } from "@/hooks/use-limits";
import type { CreditLimit } from "@/hooks/use-limits";
import { toast } from "sonner";
import { extractErrorMessage } from "@/lib/utils";

interface LimitEditDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  limit: CreditLimit | null;
}

export function LimitEditDialog({
  open,
  onOpenChange,
  limit,
}: LimitEditDialogProps) {
  const { t } = useTranslation();
  const [uncollateralizedUnlimited, setUncollateralizedUnlimited] =
    useState(false);
  const [collateralizedUnlimited, setCollateralizedUnlimited] =
    useState(false);
  const [uncollateralizedAmount, setUncollateralizedAmount] = useState("");
  const [collateralizedAmount, setCollateralizedAmount] = useState("");
  const [approvalReference, setApprovalReference] = useState("");
  const [formError, setFormError] = useState("");

  const mutation = useUpdateCreditLimit(limit?.counterparty_id ?? "");

  useEffect(() => {
    if (limit) {
      setUncollateralizedUnlimited(limit.uncollateralized_unlimited);
      setCollateralizedUnlimited(limit.collateralized_unlimited);
      setUncollateralizedAmount(
        limit.uncollateralized_limit?.toString() ?? ""
      );
      setCollateralizedAmount(
        limit.collateralized_limit?.toString() ?? ""
      );
      setApprovalReference(limit.approval_reference ?? "");
      setFormError("");
    }
  }, [limit]);

  function handleSave() {
    setFormError("");

    if (
      !uncollateralizedUnlimited &&
      !uncollateralizedAmount.trim()
    ) {
      setFormError(t("limit.errorAmountRequired"));
      return;
    }
    if (!collateralizedUnlimited && !collateralizedAmount.trim()) {
      setFormError(t("limit.errorAmountRequired"));
      return;
    }

    mutation.mutate(
      {
        uncollateralized_unlimited: uncollateralizedUnlimited,
        collateralized_unlimited: collateralizedUnlimited,
        uncollateralized_limit: uncollateralizedUnlimited
          ? null
          : Number(uncollateralizedAmount),
        collateralized_limit: collateralizedUnlimited
          ? null
          : Number(collateralizedAmount),
        approval_reference: approvalReference || undefined,
        version: limit?.version ?? 0,
      },
      {
        onSuccess: () => {
          toast.success(t("limit.updateSuccess"));
          onOpenChange(false);
        },
        onError: (err) => {
          setFormError(extractErrorMessage(err));
        },
      }
    );
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>
            {t("limit.editTitle")} — {limit?.counterparty_name}
          </DialogTitle>
        </DialogHeader>

        {formError && (
          <Alert variant="destructive">
            <IconAlertCircle className="size-4" />
            <AlertDescription>{formError}</AlertDescription>
          </Alert>
        )}

        <div className="space-y-6">
          {/* Uncollateralized */}
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <Label className="text-sm font-medium">
                {t("limit.uncollateralized")}
              </Label>
              <div className="flex items-center gap-2">
                <Checkbox
                  id="uncollateralized-unlimited"
                  checked={uncollateralizedUnlimited}
                  onCheckedChange={(checked) =>
                    setUncollateralizedUnlimited(checked as boolean)
                  }
                />
                <span className="text-xs text-muted-foreground">
                  {t("limit.unlimited")}
                </span>
              </div>
            </div>
            {!uncollateralizedUnlimited && (
              <Input
                type="number"
                placeholder={t("limit.amountPlaceholder")}
                value={uncollateralizedAmount}
                onChange={(e) =>
                  setUncollateralizedAmount(e.target.value)
                }
              />
            )}
          </div>

          {/* Collateralized */}
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <Label className="text-sm font-medium">
                {t("limit.collateralized")}
              </Label>
              <div className="flex items-center gap-2">
                <Checkbox
                  id="collateralized-unlimited"
                  checked={collateralizedUnlimited}
                  onCheckedChange={(checked) =>
                    setCollateralizedUnlimited(checked as boolean)
                  }
                />
                <span className="text-xs text-muted-foreground">
                  {t("limit.unlimited")}
                </span>
              </div>
            </div>
            {!collateralizedUnlimited && (
              <Input
                type="number"
                placeholder={t("limit.amountPlaceholder")}
                value={collateralizedAmount}
                onChange={(e) =>
                  setCollateralizedAmount(e.target.value)
                }
              />
            )}
          </div>

          {/* Approval reference */}
          <div className="space-y-2">
            <Label>{t("limit.approvalReference")}</Label>
            <Textarea
              placeholder={t("limit.approvalReferencePlaceholder")}
              value={approvalReference}
              onChange={(e) => setApprovalReference(e.target.value)}
              rows={2}
            />
          </div>
        </div>

        <DialogFooter>
          <DialogClose render={<Button variant="outline" />}>
            {t("common.cancel")}
          </DialogClose>
          <Button onClick={handleSave} disabled={mutation.isPending}>
            {mutation.isPending ? t("common.saving") : t("common.save")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
