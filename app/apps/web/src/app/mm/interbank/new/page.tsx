"use client";

import { useState, useMemo, useEffect } from "react";
import { useRouter } from "next/navigation";
import { Header } from "@/components/header";
import { useTranslation } from "@/lib/i18n";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { DatePicker } from "@/components/ui/date-picker";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Checkbox } from "@/components/ui/checkbox";
import { Badge } from "@/components/ui/badge";
import { IconArrowLeft, IconCalculator } from "@tabler/icons-react";
import { toast } from "sonner";
import { extractErrorMessage } from "@/lib/utils";
import { useCreateMMInterbank } from "@/hooks/use-mm";
import { useCounterparties } from "@/hooks/use-counterparties";

const CURRENCIES = ["VND", "USD", "EUR", "GBP", "AUD", "JPY", "CHF", "KRW"];
const DIRECTIONS = ["PLACE", "TAKE", "LEND", "BORROW"] as const;
const DAY_COUNT_CONVENTIONS = ["ACT_365", "ACT_360", "ACT_ACT"] as const;

// Swift code lookup by counterparty code
const SWIFT_CODES: Record<string, string> = {
  MSB: "MCOBVNVX", ACB: "ASCBVNVX", VCB: "BFTVVNVX", TCB: "VTCBVNVX",
  VPB: "VPBKVNVX", MBB: "MSCBVNVX", BID: "BIDVVNVX", CTG: "ICBVVNVX",
  STB: "SGTTVNVX", SHB: "SHBAVNVX",
};

// SSI KLB options
const SSI_KLB_OPTIONS = [
  { value: "SGD_NHNN_CITAD", label: "SGD NHNN – Citad" },
  { value: "VCB", label: "VCB" },
  { value: "HABIB", label: "Habib" },
] as const;

function getDayCountBase(convention: string): number {
  if (convention === "ACT_360") return 360;
  return 365; // ACT_365 and ACT_ACT (simplified)
}

function formatAmount(value: number, currency: string): string {
  if (currency === "VND") {
    return new Intl.NumberFormat("vi-VN", { maximumFractionDigits: 0 }).format(value) + " VND";
  }
  const decimals = 2;
  return new Intl.NumberFormat("en-US", { minimumFractionDigits: decimals, maximumFractionDigits: decimals }).format(value) + ` ${currency || ""}`;
}

function addDays(dateStr: string, days: number): string {
  const d = new Date(dateStr + "T00:00:00");
  d.setDate(d.getDate() + days);
  return d.toISOString().split("T")[0];
}

function formatAmountInput(value: number, currency: string): string {
  if (currency === "VND") {
    return new Intl.NumberFormat("vi-VN", { maximumFractionDigits: 0 }).format(value);
  }
  return new Intl.NumberFormat("en-US", { minimumFractionDigits: 2, maximumFractionDigits: 2 }).format(value);
}

export default function MMInterbankNewPage() {
  const { t } = useTranslation();
  const router = useRouter();
  const counterparties = useCounterparties();
  const createMutation = useCreateMMInterbank();
  const [apiError, setApiError] = useState<string | null>(null);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [amountDisplay, setAmountDisplay] = useState("");
  const [amountFocused, setAmountFocused] = useState(false);

  const [form, setForm] = useState({
    counterparty_id: "",
    currency_code: "",
    direction: "",
    principal_amount: "",
    interest_rate: "",
    day_count_convention: "",
    trade_date: "",
    effective_date: "",
    tenor_days: "",
    has_collateral: false,
    collateral_currency: "",
    collateral_description: "",
    requires_international_settlement: false,
    ticket_number: "",
    note: "",
    internal_ssi_text: "",
    counterparty_ssi_text: "",
  });

  // Track whether user has manually edited the counterparty SSI
  const [ssiManuallyEdited, setSsiManuallyEdited] = useState(false);

  const set = (field: string, value: string | boolean) =>
    setForm((prev) => ({ ...prev, [field]: value }));

  // Derived: selected counterparty & swift code
  const selectedCp = counterparties.find((c) => c.id === form.counterparty_id);
  const swiftCode = selectedCp ? SWIFT_CODES[selectedCp.code] || "" : "";

  // Auto-compute international settlement flag based on SSI KLB selection
  useEffect(() => {
    const isIntlSettlement = form.internal_ssi_text === "HABIB";
    if (form.requires_international_settlement !== isIntlSettlement) {
      setForm((prev) => ({ ...prev, requires_international_settlement: isIntlSettlement }));
    }
  }, [form.internal_ssi_text, form.requires_international_settlement]);

  // Auto-generate counterparty SSI text when counterparty or currency changes
  useEffect(() => {
    if (ssiManuallyEdited) return;
    if (!selectedCp || !form.currency_code) {
      setForm((prev) => ({ ...prev, counterparty_ssi_text: "" }));
      return;
    }
    const sw = SWIFT_CODES[selectedCp.code] || "";
    const text = `Tài khoản: ${selectedCp.code}-${form.currency_code} / SWIFT: ${sw}`;
    setForm((prev) => ({ ...prev, counterparty_ssi_text: text }));
  }, [form.counterparty_id, form.currency_code, selectedCp, ssiManuallyEdited]);

  // Auto-calc maturity_date
  const maturityDate = useMemo(() => {
    if (!form.effective_date || !form.tenor_days) return "";
    const days = parseInt(form.tenor_days, 10);
    if (isNaN(days) || days <= 0) return "";
    return addDays(form.effective_date, days);
  }, [form.effective_date, form.tenor_days]);

  // Auto-calc interest & maturity amounts
  const { interestAmount, maturityAmount } = useMemo(() => {
    const principal = parseFloat(form.principal_amount);
    const rate = parseFloat(form.interest_rate);
    const tenor = parseInt(form.tenor_days, 10);
    if (!principal || !rate || !tenor || !form.day_count_convention) {
      return { interestAmount: null, maturityAmount: null };
    }
    const base = getDayCountBase(form.day_count_convention);
    const interest = principal * (rate / 100) * tenor / base;
    return { interestAmount: interest, maturityAmount: principal + interest };
  }, [form.principal_amount, form.interest_rate, form.tenor_days, form.day_count_convention]);

  const validate = (): Record<string, string> => {
    const errs: Record<string, string> = {};
    if (!form.counterparty_id) errs.counterparty_id = t("mm.create.validation.counterpartyRequired");
    if (!form.direction) errs.direction = t("mm.create.validation.directionRequired");
    if (!form.currency_code) errs.currency_code = t("mm.create.validation.currencyRequired");
    const principal = parseFloat(form.principal_amount);
    if (!principal || principal <= 0) errs.principal_amount = t("mm.create.validation.principalPositive");
    const rate = parseFloat(form.interest_rate);
    if (isNaN(rate) || rate < 0) errs.interest_rate = t("mm.create.validation.rateNonNegative");
    if (!form.trade_date) errs.trade_date = t("mm.create.validation.tradeDateRequired");
    if (!form.effective_date) errs.effective_date = t("mm.create.validation.effectiveDateRequired");
    const tenor = parseInt(form.tenor_days, 10);
    if (!tenor || tenor <= 0) errs.tenor_days = t("mm.create.validation.tenorPositive");
    if (!form.day_count_convention) errs.day_count_convention = t("mm.create.validation.dayCountRequired");
    return errs;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setApiError(null);
    const validationErrors = validate();
    setErrors(validationErrors);
    if (Object.keys(validationErrors).length > 0) return;

    const payload: Record<string, unknown> = {
      counterparty_id: form.counterparty_id,
      currency_code: form.currency_code,
      direction: form.direction,
      principal_amount: form.principal_amount,
      interest_rate: form.interest_rate,
      day_count_convention: form.day_count_convention,
      trade_date: form.trade_date,
      effective_date: form.effective_date,
      maturity_date: maturityDate,
      has_collateral: form.has_collateral,
      requires_international_settlement: form.requires_international_settlement,
    };

    if (form.has_collateral) {
      if (form.collateral_currency) payload.collateral_currency = form.collateral_currency;
      if (form.collateral_description) payload.collateral_description = form.collateral_description;
    }
    if (form.ticket_number) payload.ticket_number = form.ticket_number;
    if (form.note) payload.note = form.note;

    createMutation.mutate(payload, {
      onSuccess: (res) => {
        toast.success(t("mm.create.success"));
        router.push(`/mm/interbank/${res.id}`);
      },
      onError: (err) => {
        setApiError(extractErrorMessage(err));
      },
    });
  };

  return (
    <>
      <Header />
      <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
        <div className="flex items-center gap-4 pt-4">
          <Button variant="ghost" size="icon" onClick={() => router.push("/mm")}>
            <IconArrowLeft className="size-4" />
          </Button>
          <div>
            <h1 className="text-2xl font-bold tracking-tight">{t("mm.create.title")}</h1>
            <p className="text-muted-foreground">{t("mm.create.description")}</p>
          </div>
        </div>

        <form onSubmit={handleSubmit} className="space-y-6">
          {apiError && (
            <div className="rounded-md border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive">
              {apiError}
            </div>
          )}

          {/* Deal Information */}
          <Card>
            <CardHeader><CardTitle>{t("mm.create.dealInfo")}</CardTitle></CardHeader>
            <CardContent className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <Label>{t("mm.create.counterparty")} *</Label>
                <Select value={form.counterparty_id} onValueChange={(v) => { v && set("counterparty_id", v); setErrors((prev) => { const { counterparty_id: _, ...rest } = prev; return rest; }); }}>
                  <SelectTrigger>
                    {form.counterparty_id
                      ? (() => { const cp = counterparties.find(c => c.id === form.counterparty_id); return cp ? `${cp.code} — ${cp.name}` : t("mm.create.selectCounterparty"); })()
                      : <SelectValue placeholder={t("mm.create.selectCounterparty")} />}
                  </SelectTrigger>
                  <SelectContent>
                    {counterparties.map((cp) => (
                      <SelectItem key={cp.id} value={cp.id} label={`${cp.code} — ${cp.name}`}>{cp.code} — {cp.name}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {errors.counterparty_id && <p className="text-sm text-destructive">{errors.counterparty_id}</p>}
              </div>
              <div className="space-y-2">
                <Label>{t("mm.create.direction")} *</Label>
                <Select value={form.direction} onValueChange={(v) => { v && set("direction", v); setErrors((prev) => { const { direction: _, ...rest } = prev; return rest; }); }}>
                  <SelectTrigger>
                    {form.direction ? t(`mm.direction.${form.direction}`) : <SelectValue placeholder={t("mm.create.selectDirection")} />}
                  </SelectTrigger>
                  <SelectContent>
                    {DIRECTIONS.map((d) => (
                      <SelectItem key={d} value={d} label={t(`mm.direction.${d}`)}>{t(`mm.direction.${d}`)}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {errors.direction && <p className="text-sm text-destructive">{errors.direction}</p>}
              </div>
              <div className="space-y-2">
                <Label>{t("mm.create.currency")} *</Label>
                <Select value={form.currency_code} onValueChange={(v) => { v && set("currency_code", v); setErrors((prev) => { const { currency_code: _, ...rest } = prev; return rest; }); }}>
                  <SelectTrigger>{form.currency_code || <SelectValue placeholder={t("mm.create.selectCurrency")} />}</SelectTrigger>
                  <SelectContent>
                    {CURRENCIES.map((c) => (<SelectItem key={c} value={c} label={c}>{c}</SelectItem>))}
                  </SelectContent>
                </Select>
                {errors.currency_code && <p className="text-sm text-destructive">{errors.currency_code}</p>}
              </div>
              <div className="space-y-2">
                <Label>{t("mm.create.principalAmount")} *</Label>
                <Input
                  type="text"
                  inputMode="numeric"
                  value={amountFocused ? form.principal_amount : (amountDisplay || form.principal_amount)}
                  onChange={(e) => {
                    const raw = e.target.value.replace(/[^0-9.]/g, "");
                    set("principal_amount", raw);
                    setErrors((prev) => { const { principal_amount: _, ...rest } = prev; return rest; });
                  }}
                  onFocus={() => { setAmountFocused(true); setAmountDisplay(""); }}
                  onBlur={() => {
                    setAmountFocused(false);
                    const num = parseFloat(form.principal_amount);
                    if (!isNaN(num) && num > 0) {
                      setAmountDisplay(formatAmountInput(num, form.currency_code || "VND"));
                    } else {
                      setAmountDisplay("");
                    }
                  }}
                  placeholder="0"
                />
                {errors.principal_amount && <p className="text-sm text-destructive">{errors.principal_amount}</p>}
              </div>
              <div className="space-y-2">
                <Label>{t("mm.create.tradeDate")} *</Label>
                <DatePicker value={form.trade_date} onChange={(v) => { set("trade_date", v); setErrors((prev) => { const { trade_date: _, ...rest } = prev; return rest; }); }} />
                {errors.trade_date && <p className="text-sm text-destructive">{errors.trade_date}</p>}
              </div>
              <div className="space-y-2">
                <Label>{t("mm.create.ticketNumber")}</Label>
                <Input value={form.ticket_number} onChange={(e) => set("ticket_number", e.target.value)} placeholder={t("mm.create.ticketNumberPlaceholder")} />
              </div>
            </CardContent>
          </Card>

          {/* Financial Details */}
          <Card>
            <CardHeader><CardTitle>{t("mm.create.financials")}</CardTitle></CardHeader>
            <CardContent className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <Label>{t("mm.create.interestRate")} *</Label>
                <Input type="number" min="0" step="any" value={form.interest_rate} onChange={(e) => { set("interest_rate", e.target.value); setErrors((prev) => { const { interest_rate: _, ...rest } = prev; return rest; }); }} placeholder="0.00" />
                {errors.interest_rate && <p className="text-sm text-destructive">{errors.interest_rate}</p>}
              </div>
              <div className="space-y-2">
                <Label>{t("mm.create.dayCountConvention")} *</Label>
                <Select value={form.day_count_convention} onValueChange={(v) => { v && set("day_count_convention", v); setErrors((prev) => { const { day_count_convention: _, ...rest } = prev; return rest; }); }}>
                  <SelectTrigger>{form.day_count_convention ? form.day_count_convention.replace(/_/g, "/") : <SelectValue placeholder={t("mm.create.selectDayCount")} />}</SelectTrigger>
                  <SelectContent>
                    {DAY_COUNT_CONVENTIONS.map((d) => (
                      <SelectItem key={d} value={d} label={d.replace(/_/g, "/")}>{d.replace(/_/g, "/")}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {errors.day_count_convention && <p className="text-sm text-destructive">{errors.day_count_convention}</p>}
              </div>
            </CardContent>
          </Card>

          {/* Dates */}
          <Card>
            <CardHeader><CardTitle>{t("mm.create.dates")}</CardTitle></CardHeader>
            <CardContent className="grid grid-cols-1 gap-4 md:grid-cols-3">
              <div className="space-y-2">
                <Label>{t("mm.create.effectiveDate")} *</Label>
                <DatePicker value={form.effective_date} onChange={(v) => { set("effective_date", v); setErrors((prev) => { const { effective_date: _, ...rest } = prev; return rest; }); }} />
                {errors.effective_date && <p className="text-sm text-destructive">{errors.effective_date}</p>}
              </div>
              <div className="space-y-2">
                <Label>{t("mm.create.tenorDays")} *</Label>
                <Input type="number" min="1" step="1" value={form.tenor_days} onChange={(e) => { set("tenor_days", e.target.value); setErrors((prev) => { const { tenor_days: _, ...rest } = prev; return rest; }); }} placeholder="30" />
                {errors.tenor_days && <p className="text-sm text-destructive">{errors.tenor_days}</p>}
              </div>
              <div className="space-y-2">
                <Label>{t("mm.create.maturityDate")} ({t("mm.create.autoCalculated")})</Label>
                <DatePicker value={maturityDate} disabled placeholder={t("mm.create.autoCalculated")} />
              </div>
            </CardContent>
          </Card>

          {/* Calculation Result */}
          {interestAmount !== null && maturityAmount !== null && (
            <Card className="border-primary/30 bg-primary/5">
              <CardHeader className="pb-3">
                <CardTitle className="flex items-center gap-2 text-base">
                  <IconCalculator className="size-4" />
                  {t("mm.create.calculationResult")}
                </CardTitle>
              </CardHeader>
              <CardContent className="grid grid-cols-1 gap-4 md:grid-cols-2">
                <div className="space-y-1">
                  <p className="text-sm text-muted-foreground">{t("mm.create.interestAmount")}</p>
                  <p className="text-lg font-semibold">{formatAmount(interestAmount, form.currency_code)}</p>
                </div>
                <div className="space-y-1">
                  <p className="text-sm text-muted-foreground">{t("mm.create.maturityAmount")}</p>
                  <p className="text-lg font-semibold">{formatAmount(maturityAmount, form.currency_code)}</p>
                </div>
              </CardContent>
            </Card>
          )}

          {/* Settlement & Collateral */}
          <Card>
            <CardHeader><CardTitle>{t("mm.create.settlement")}</CardTitle></CardHeader>
            <CardContent className="space-y-4">
              {/* Swift Code (read-only) */}
              <div className="space-y-2">
                <Label>{t("mm.create.swiftCode")}</Label>
                <Input value={swiftCode} disabled placeholder="—" />
              </div>

              {/* SSI KLB */}
              <div className="space-y-2">
                <Label>{t("mm.create.ssiKlb")}</Label>
                <Select value={form.internal_ssi_text} onValueChange={(v) => v && set("internal_ssi_text", v)}>
                  <SelectTrigger>{form.internal_ssi_text ? (SSI_KLB_OPTIONS.find(o => o.value === form.internal_ssi_text)?.label ?? form.internal_ssi_text) : <SelectValue placeholder={t("mm.create.selectSsi")} />}</SelectTrigger>
                  <SelectContent>
                    {SSI_KLB_OPTIONS.map((opt) => (
                      <SelectItem key={opt.value} value={opt.value} label={opt.label}>{opt.label}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              {/* SSI Counterparty (editable) */}
              <div className="space-y-2">
                <Label>{t("mm.create.ssiCounterparty")}</Label>
                <Textarea
                  value={form.counterparty_ssi_text}
                  onChange={(e) => {
                    setSsiManuallyEdited(true);
                    set("counterparty_ssi_text", e.target.value);
                  }}
                  rows={2}
                  placeholder="—"
                />
              </div>

              {/* International settlement auto flag */}
              <div className="flex items-center gap-3">
                <Label>{t("mm.create.intlSettlement")}</Label>
                {form.requires_international_settlement ? (
                  <Badge variant="default" className="bg-green-600 hover:bg-green-700">{t("mm.create.intlYes")}</Badge>
                ) : (
                  <Badge variant="secondary">{t("mm.create.intlNo")}</Badge>
                )}
              </div>

              {/* Has Collateral toggle */}
              <div className="flex items-center gap-3">
                <Checkbox checked={form.has_collateral} onCheckedChange={(v) => set("has_collateral", !!v)} />
                <Label>{t("mm.create.hasCollateral")}</Label>
              </div>
              {form.has_collateral && (
                <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                  <div className="space-y-2">
                    <Label>{t("mm.create.collateralCurrency")}</Label>
                    <Select value={form.collateral_currency} onValueChange={(v) => v && set("collateral_currency", v)}>
                      <SelectTrigger>{form.collateral_currency || <SelectValue placeholder={t("mm.create.selectCurrency")} />}</SelectTrigger>
                      <SelectContent>
                        {CURRENCIES.map((c) => (<SelectItem key={c} value={c} label={c}>{c}</SelectItem>))}
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label>{t("mm.create.collateralDescription")}</Label>
                    <Input value={form.collateral_description} onChange={(e) => set("collateral_description", e.target.value)} placeholder={t("mm.create.collateralDescPlaceholder")} />
                  </div>
                </div>
              )}
            </CardContent>
          </Card>

          {/* Note */}
          <Card>
            <CardHeader><CardTitle>{t("mm.create.note")}</CardTitle></CardHeader>
            <CardContent>
              <Textarea value={form.note} onChange={(e) => set("note", e.target.value)} placeholder={t("mm.create.notePlaceholder")} rows={3} />
            </CardContent>
          </Card>

          {/* Actions */}
          <div className="flex justify-end gap-3">
            <Button type="button" variant="outline" onClick={() => router.push("/mm")}>
              {t("common.cancel")}
            </Button>
            <Button type="submit" disabled={createMutation.isPending}>
              {createMutation.isPending ? t("common.saving") : t("mm.create.submit")}
            </Button>
          </div>
        </form>
      </div>
    </>
  );
}
