"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Separator } from "@/components/ui/separator";
import { Badge } from "@/components/ui/badge";
import { IconAlertCircle, IconPlus, IconTrash } from "@tabler/icons-react";
import { DatePicker } from "@/components/ui/date-picker";
import { useTranslation } from "@/lib/i18n";
import type {
  FxDeal,
  FxDealType,
  FxDirection,
  CreateFxDealRequest,
  CreateFxLeg,
} from "@/hooks/use-fx";

interface Counterparty {
  id: string;
  code: string;
  name: string;
}

interface FxDealFormProps {
  initialData?: FxDeal;
  counterparties: Counterparty[];
  onSubmit: (data: CreateFxDealRequest) => void;
  onCancel: () => void;
  isSubmitting: boolean;
  error?: string | null;
  submitLabel: string;
}

const DEAL_TYPES: FxDealType[] = ["SPOT", "FORWARD", "SWAP"];
const DIRECTIONS: FxDirection[] = ["BUY", "SELL", "BUY_SELL", "SELL_BUY"];
const CURRENCIES = ["USD", "EUR", "GBP", "JPY", "CHF", "AUD", "CAD", "SGD", "HKD", "CNY"];

function emptyLeg(legNumber: number): CreateFxLeg {
  return {
    leg_number: legNumber,
    value_date: "",
    exchange_rate: 0,
    buy_currency: "USD",
    sell_currency: "VND",
    buy_amount: 0,
    sell_amount: 0,
    pay_code_klb: "",
    pay_code_counterparty: "",
    execution_date: "",
  };
}

function dealToFormState(deal: FxDeal) {
  return {
    counterparty_id: deal.counterparty_id,
    deal_type: deal.deal_type,
    direction: deal.direction,
    notional_amount: Number(deal.notional_amount),
    currency_code: deal.currency_code,
    trade_date: deal.trade_date.slice(0, 10),
    execution_date: deal.execution_date?.slice(0, 10) || "",
    pay_code_klb: deal.pay_code_klb || "",
    pay_code_counterparty: deal.pay_code_counterparty || "",
    note: deal.note || "",
    legs: deal.legs.map((leg) => ({
      leg_number: leg.leg_number,
      value_date: leg.value_date.slice(0, 10),
      exchange_rate: Number(leg.exchange_rate),
      buy_currency: leg.buy_currency,
      sell_currency: leg.sell_currency,
      buy_amount: Number(leg.buy_amount),
      sell_amount: Number(leg.sell_amount),
      pay_code_klb: leg.pay_code_klb || "",
      pay_code_counterparty: leg.pay_code_counterparty || "",
      execution_date: leg.execution_date?.slice(0, 10) || "",
    })),
  };
}

export function FxDealForm({
  initialData,
  counterparties,
  onSubmit,
  onCancel,
  isSubmitting,
  error,
  submitLabel,
}: FxDealFormProps) {
  const { t } = useTranslation();

  const [form, setForm] = useState<{
    counterparty_id: string;
    deal_type: FxDealType;
    direction: FxDirection;
    notional_amount: number;
    currency_code: string;
    trade_date: string;
    execution_date: string;
    pay_code_klb: string;
    pay_code_counterparty: string;
    note: string;
    legs: CreateFxLeg[];
  }>(
    initialData
      ? dealToFormState(initialData)
      : {
          counterparty_id: "",
          deal_type: "SPOT",
          direction: "BUY",
          notional_amount: 0,
          currency_code: "USD",
          trade_date: new Date().toISOString().slice(0, 10),
          execution_date: "",
          pay_code_klb: "",
          pay_code_counterparty: "",
          note: "",
          legs: [emptyLeg(1)],
        }
  );

  const [validationError, setValidationError] = useState<string | null>(null);

  function updateField<K extends keyof typeof form>(key: K, value: (typeof form)[K]) {
    setForm((prev) => ({ ...prev, [key]: value }));
  }

  function updateLeg(index: number, field: keyof CreateFxLeg, value: string | number) {
    setForm((prev) => {
      const legs = [...prev.legs];
      legs[index] = { ...legs[index], [field]: value };
      return { ...prev, legs };
    });
  }

  function addLeg() {
    setForm((prev) => ({
      ...prev,
      legs: [...prev.legs, emptyLeg(prev.legs.length + 1)],
    }));
  }

  function removeLeg(index: number) {
    setForm((prev) => ({
      ...prev,
      legs: prev.legs
        .filter((_, i) => i !== index)
        .map((leg, i) => ({ ...leg, leg_number: i + 1 })),
    }));
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setValidationError(null);

    if (!form.counterparty_id) {
      setValidationError(t("fx.selectCounterparty"));
      return;
    }
    if (!form.trade_date) {
      setValidationError(t("fx.tradeDate"));
      return;
    }
    if (form.notional_amount <= 0) {
      setValidationError(t("fx.notionalAmount"));
      return;
    }
    if (form.legs.length === 0) {
      setValidationError(t("fx.addLeg"));
      return;
    }
    for (const leg of form.legs) {
      if (!leg.value_date || leg.exchange_rate <= 0) {
        setValidationError(`${t("fx.legNumber")} ${leg.leg_number}: ${t("fx.valueDate")} / ${t("fx.exchangeRate")}`);
        return;
      }
    }

    onSubmit({
      counterparty_id: form.counterparty_id,
      deal_type: form.deal_type,
      direction: form.direction,
      notional_amount: form.notional_amount,
      currency_code: form.currency_code,
      trade_date: form.trade_date,
      execution_date: form.execution_date || undefined,
      pay_code_klb: form.pay_code_klb || undefined,
      pay_code_counterparty: form.pay_code_counterparty || undefined,
      note: form.note || undefined,
      legs: form.legs,
    });
  }

  const isSwap = form.deal_type === "SWAP";

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {(error || validationError) && (
        <Alert variant="destructive">
          <IconAlertCircle className="size-4" />
          <AlertDescription>{error || validationError}</AlertDescription>
        </Alert>
      )}

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("fx.detail")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            <div className="space-y-2">
              <Label htmlFor="counterparty">{t("fx.counterparty")}</Label>
              <Select
                value={form.counterparty_id}
                onValueChange={(v) => updateField("counterparty_id", v ?? "")}
              >
                <SelectTrigger id="counterparty">
                  {form.counterparty_id
                    ? (() => { const cp = counterparties.find((c) => c.id === form.counterparty_id); return cp ? `${cp.code} — ${cp.name}` : t("fx.selectCounterparty"); })()
                    : <SelectValue placeholder={t("fx.selectCounterparty")} />}
                </SelectTrigger>
                <SelectContent>
                  {counterparties.map((cp) => (
                    <SelectItem key={cp.id} value={cp.id} label={`${cp.code} — ${cp.name}`}>
                      {cp.code} — {cp.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="deal_type">{t("fx.dealType")}</Label>
              <Select
                value={form.deal_type}
                onValueChange={(v) => {
                  if (!v) return;
                  updateField("deal_type", v as FxDealType);
                  if (v === "SWAP" && form.legs.length < 2) {
                    setForm((prev) => ({
                      ...prev,
                      deal_type: v as FxDealType,
                      legs: [...prev.legs, emptyLeg(prev.legs.length + 1)],
                    }));
                  }
                }}
              >
                <SelectTrigger id="deal_type">
                  {form.deal_type ? t(`fx.type.${form.deal_type}`) : <SelectValue placeholder={t("fx.dealType")} />}
                </SelectTrigger>
                <SelectContent>
                  {DEAL_TYPES.map((dt) => (
                    <SelectItem key={dt} value={dt} label={t(`fx.type.${dt}`)}>
                      {t(`fx.type.${dt}`)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="direction">{t("fx.direction")}</Label>
              <Select
                value={form.direction}
                onValueChange={(v) => v && updateField("direction", v as FxDirection)}
              >
                <SelectTrigger id="direction">
                  {form.direction ? t(`fx.direction.${form.direction}`) : <SelectValue placeholder={t("fx.direction")} />}
                </SelectTrigger>
                <SelectContent>
                  {DIRECTIONS.map((d) => (
                    <SelectItem key={d} value={d} label={t(`fx.direction.${d}`)}>
                      {t(`fx.direction.${d}`)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="notional_amount">{t("fx.notionalAmount")}</Label>
              <Input
                id="notional_amount"
                type="number"
                step="0.01"
                min="0"
                value={form.notional_amount || ""}
                onChange={(e) => updateField("notional_amount", Number(e.target.value))}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="currency_code">{t("fx.currency")}</Label>
              <Select
                value={form.currency_code}
                onValueChange={(v) => v && updateField("currency_code", v)}
              >
                <SelectTrigger id="currency_code">
                  <SelectValue>{(v: string) => v}</SelectValue>
                </SelectTrigger>
                <SelectContent>
                  {CURRENCIES.map((c) => (
                    <SelectItem key={c} value={c} label={c}>
                      {c}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="trade_date">{t("fx.tradeDate")}</Label>
              <DatePicker
                id="trade_date"
                value={form.trade_date}
                onChange={(val) => updateField("trade_date", val)}
              />
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="note">{t("fx.note")}</Label>
            <Textarea
              id="note"
              value={form.note}
              onChange={(e) => updateField("note", e.target.value)}
              rows={2}
            />
          </div>
        </CardContent>
      </Card>

      {!isSwap && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">{t("fx.paymentInfo")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
              <div className="space-y-2">
                <Label htmlFor="execution_date">{t("fx.executionDate")}</Label>
                <DatePicker
                  id="execution_date"
                  value={form.execution_date}
                  onChange={(val) => updateField("execution_date", val)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="pay_code_klb">{t("fx.payCodeKlb")}</Label>
                <Input
                  id="pay_code_klb"
                  placeholder="VD: IRVTUS3N-8901358797"
                  value={form.pay_code_klb}
                  onChange={(e) => updateField("pay_code_klb", e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="pay_code_counterparty">{t("fx.payCodeCounterparty")}</Label>
                <Input
                  id="pay_code_counterparty"
                  value={form.pay_code_counterparty}
                  onChange={(e) => updateField("pay_code_counterparty", e.target.value)}
                />
              </div>
            </div>
            <div className="flex items-center gap-2 text-sm">
              <span className="text-muted-foreground">Intl Settlement:</span>
              <Badge variant={form.pay_code_counterparty ? "default" : "secondary"}>
                {form.pay_code_counterparty ? t("common.yes") : t("common.no")}
              </Badge>
            </div>
            {initialData?.settlement_amount && (
              <div className="text-sm">
                <span className="text-muted-foreground">{t("fx.settlementAmount")}:</span>{" "}
                <span className="font-medium tabular-nums">
                  {Number(initialData.settlement_amount).toLocaleString("en-US", { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
                </span>{" "}
                <span className="font-mono">{initialData.settlement_currency}</span>
              </div>
            )}
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle className="text-base">{t("fx.legs")}</CardTitle>
          <Button type="button" variant="outline" size="sm" onClick={addLeg}>
            <IconPlus className="mr-1.5 size-4" />
            {t("fx.addLeg")}
          </Button>
        </CardHeader>
        <CardContent className="space-y-6">
          {form.legs.map((leg, index) => (
            <div key={index}>
              {index > 0 && <Separator className="mb-6" />}
              <div className="flex items-center justify-between mb-3">
                <h4 className="text-sm font-medium">
                  {t("fx.legNumber")} {leg.leg_number}
                </h4>
                {form.legs.length > 1 && !(isSwap && form.legs.length <= 2) && (
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => removeLeg(index)}
                    className="text-destructive hover:text-destructive"
                  >
                    <IconTrash className="mr-1 size-4" />
                    {t("fx.removeLeg")}
                  </Button>
                )}
              </div>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
                <div className="space-y-2">
                  <Label>{t("fx.valueDate")}</Label>
                  <DatePicker
                    value={leg.value_date}
                    onChange={(val) => updateLeg(index, "value_date", val)}
                  />
                </div>
                <div className="space-y-2">
                  <Label>{t("fx.exchangeRate")}</Label>
                  <Input
                    type="number"
                    step="0.01"
                    min="0"
                    value={leg.exchange_rate || ""}
                    onChange={(e) =>
                      updateLeg(index, "exchange_rate", Number(e.target.value))
                    }
                  />
                </div>
                <div className="space-y-2">
                  <Label>{t("fx.buyCurrency")}</Label>
                  <Select
                    value={leg.buy_currency}
                    onValueChange={(v) => v && updateLeg(index, "buy_currency", v)}
                  >
                    <SelectTrigger>
                      <SelectValue>{(v: string) => v}</SelectValue>
                    </SelectTrigger>
                    <SelectContent>
                      {[...CURRENCIES, "VND"].map((c) => (
                        <SelectItem key={c} value={c} label={c}>
                          {c}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label>{t("fx.sellCurrency")}</Label>
                  <Select
                    value={leg.sell_currency}
                    onValueChange={(v) => v && updateLeg(index, "sell_currency", v)}
                  >
                    <SelectTrigger>
                      <SelectValue>{(v: string) => v}</SelectValue>
                    </SelectTrigger>
                    <SelectContent>
                      {[...CURRENCIES, "VND"].map((c) => (
                        <SelectItem key={c} value={c} label={c}>
                          {c}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label>{t("fx.buyAmount")}</Label>
                  <Input
                    type="number"
                    step="0.01"
                    min="0"
                    value={leg.buy_amount || ""}
                    onChange={(e) =>
                      updateLeg(index, "buy_amount", Number(e.target.value))
                    }
                  />
                </div>
                <div className="space-y-2">
                  <Label>{t("fx.sellAmount")}</Label>
                  <Input
                    type="number"
                    step="0.01"
                    min="0"
                    value={leg.sell_amount || ""}
                    onChange={(e) =>
                      updateLeg(index, "sell_amount", Number(e.target.value))
                    }
                  />
                </div>
              </div>
              {isSwap && (
                <>
                  <Separator className="my-4" />
                  <h5 className="text-sm font-medium text-muted-foreground mb-3">
                    {t("fx.paymentInfo")}
                  </h5>
                  <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
                    <div className="space-y-2">
                      <Label>{t("fx.executionDate")}</Label>
                      <DatePicker
                        value={leg.execution_date || ""}
                        onChange={(val) => updateLeg(index, "execution_date", val)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label>{t("fx.payCodeKlb")}</Label>
                      <Input
                        placeholder="VD: IRVTUS3N-8901358797"
                        value={leg.pay_code_klb || ""}
                        onChange={(e) => updateLeg(index, "pay_code_klb", e.target.value)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label>{t("fx.payCodeCounterparty")}</Label>
                      <Input
                        value={leg.pay_code_counterparty || ""}
                        onChange={(e) => updateLeg(index, "pay_code_counterparty", e.target.value)}
                      />
                    </div>
                  </div>
                </>
              )}
            </div>
          ))}
        </CardContent>
      </Card>

      <div className="flex items-center justify-end gap-3">
        <Button type="button" variant="outline" onClick={onCancel} disabled={isSubmitting}>
          {t("common.cancel")}
        </Button>
        <Button type="submit" disabled={isSubmitting}>
          {isSubmitting ? t("common.loading") : submitLabel}
        </Button>
      </div>
    </form>
  );
}
