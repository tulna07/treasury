"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { IconAlertCircle } from "@tabler/icons-react";
import { useTranslation } from "@/lib/i18n";

interface CounterpartyFormData {
  code: string;
  full_name: string;
  short_name: string;
  swift_code: string;
  country_code: string;
  cif: string;
}

interface CounterpartyFormProps {
  initialData?: Partial<CounterpartyFormData>;
  onSubmit: (data: CounterpartyFormData) => void;
  onCancel: () => void;
  isSubmitting: boolean;
  error: string | null;
  submitLabel: string;
}

export function CounterpartyForm({
  initialData,
  onSubmit,
  onCancel,
  isSubmitting,
  error,
  submitLabel,
}: CounterpartyFormProps) {
  const { t } = useTranslation();

  const [form, setForm] = useState<CounterpartyFormData>({
    code: initialData?.code ?? "",
    full_name: initialData?.full_name ?? "",
    short_name: initialData?.short_name ?? "",
    swift_code: initialData?.swift_code ?? "",
    country_code: initialData?.country_code ?? "VN",
    cif: initialData?.cif ?? "",
  });

  const [validationError, setValidationError] = useState<string | null>(null);

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setValidationError(null);

    if (!form.code.trim() || !form.full_name.trim()) {
      setValidationError(t("counterparties.validation.required"));
      return;
    }

    onSubmit(form);
  }

  function update(field: keyof CounterpartyFormData, value: string) {
    setForm((prev) => ({ ...prev, [field]: value }));
  }

  return (
    <form onSubmit={handleSubmit}>
      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("common.details")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {(error || validationError) && (
            <Alert variant="destructive">
              <IconAlertCircle className="size-4" />
              <AlertDescription>{error || validationError}</AlertDescription>
            </Alert>
          )}

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="code">{t("counterparties.code")} *</Label>
              <Input
                id="code"
                value={form.code}
                onChange={(e) => update("code", e.target.value.toUpperCase())}
                placeholder="VD: VCB"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="full_name">{t("counterparties.name")} *</Label>
              <Input
                id="full_name"
                value={form.full_name}
                onChange={(e) => update("full_name", e.target.value)}
                placeholder="VD: Ngân hàng TMCP Ngoại thương Việt Nam"
              />
            </div>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="short_name">{t("counterparties.shortName")}</Label>
              <Input
                id="short_name"
                value={form.short_name}
                onChange={(e) => update("short_name", e.target.value)}
                placeholder="VD: Vietcombank"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="swift_code">{t("counterparties.swiftCode")}</Label>
              <Input
                id="swift_code"
                value={form.swift_code}
                onChange={(e) => update("swift_code", e.target.value.toUpperCase())}
                placeholder="VD: BFTVVNVX"
              />
            </div>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="country_code">{t("counterparties.country")}</Label>
              <Input
                id="country_code"
                value={form.country_code}
                onChange={(e) => update("country_code", e.target.value.toUpperCase())}
                placeholder="VD: VN"
                maxLength={2}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="cif">{t("counterparties.cif")}</Label>
              <Input
                id="cif"
                value={form.cif}
                onChange={(e) => update("cif", e.target.value)}
                placeholder="VD: CIF-VCB-001"
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
