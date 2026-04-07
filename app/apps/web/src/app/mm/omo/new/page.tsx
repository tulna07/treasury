"use client";

import { useState, useMemo } from "react";
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
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { IconArrowLeft, IconSelector } from "@tabler/icons-react";
import { toast } from "sonner";
import { extractErrorMessage } from "@/lib/utils";
import { useCreateMMOMO } from "@/hooks/use-mm";
import { useCounterparties } from "@/hooks/use-counterparties";
import { useBondCatalog } from "@/hooks/use-bonds";

const today = () => new Date().toISOString().slice(0, 10);

export default function MMOMONewPage() {
  const { t } = useTranslation();
  const router = useRouter();
  const counterparties = useCounterparties();
  const { data: bondCatalog = [] } = useBondCatalog();
  const createMutation = useCreateMMOMO();
  const [apiError, setApiError] = useState<string | null>(null);
  const [bondOpen, setBondOpen] = useState(false);

  // Find SBV counterparty or fall back to first
  const sbvCounterparty = useMemo(() => {
    const sbv = counterparties.find(
      (cp) =>
        cp.name?.toLowerCase().includes("nhnn") ||
        cp.name?.toLowerCase().includes("ngân hàng nhà nước") ||
        cp.code?.toUpperCase() === "SBV" ||
        cp.code?.toUpperCase() === "NHNN"
    );
    return sbv || counterparties[0];
  }, [counterparties]);

  const [form, setForm] = useState({
    session_name: "",
    counterparty_id: sbvCounterparty?.id ?? "",
    bond_catalog_id: "",
    bond_code: "",
    bond_issuer: "",
    bond_coupon_rate: "",
    bond_maturity_date: "",
    notional_amount: "",
    winning_rate: "",
    tenor_days: "",
    trade_date: today(),
    settlement_date_1: "",
    settlement_date_2: "",
    haircut_pct: "",
    note: "",
  });

  // Update counterparty_id when counterparties load
  const counterpartyId = sbvCounterparty?.id ?? "";
  if (form.counterparty_id === "" && counterpartyId) {
    setForm((prev) => ({ ...prev, counterparty_id: counterpartyId }));
  }

  const set = (field: string, value: string) =>
    setForm((prev) => ({ ...prev, [field]: value }));

  const handleBondSelect = (catalogId: string) => {
    const bond = bondCatalog.find((b) => b.id === catalogId);
    if (bond) {
      setForm((prev) => ({
        ...prev,
        bond_catalog_id: bond.id,
        bond_code: bond.bond_code,
        bond_issuer: bond.issuer,
        bond_coupon_rate: bond.coupon_rate,
        bond_maturity_date: bond.maturity_date ?? "",
      }));
    }
    setBondOpen(false);
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setApiError(null);

    const payload: Record<string, unknown> = {
      deal_subtype: "OMO",
      session_name: form.session_name,
      counterparty_id: form.counterparty_id,
      bond_catalog_id: form.bond_catalog_id,
      bond_code: form.bond_code,
      bond_issuer: form.bond_issuer,
      bond_coupon_rate: form.bond_coupon_rate,
      notional_amount: form.notional_amount,
      winning_rate: form.winning_rate,
      tenor_days: parseInt(form.tenor_days, 10),
      trade_date: form.trade_date,
      settlement_date_1: form.settlement_date_1,
      settlement_date_2: form.settlement_date_2,
      haircut_pct: form.haircut_pct,
    };

    if (form.note) payload.note = form.note;

    createMutation.mutate(payload, {
      onSuccess: (res) => {
        toast.success(t("mm.omo.create.success"));
        router.push(`/mm/omo/${res.id}`);
      },
      onError: (err) => {
        setApiError(extractErrorMessage(err));
      },
    });
  };

  const selectedBondLabel = form.bond_catalog_id
    ? `${form.bond_code} — ${form.bond_issuer}`
    : t("mm.omo.create.selectBond");

  return (
    <>
      <Header />
      <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
        <div className="flex items-center gap-4 pt-4">
          <Button variant="ghost" size="icon" onClick={() => router.push("/mm")}>
            <IconArrowLeft className="size-4" />
          </Button>
          <div>
            <h1 className="text-2xl font-bold tracking-tight">{t("mm.omo.create.title")}</h1>
            <p className="text-muted-foreground">{t("mm.omo.create.description")}</p>
          </div>
        </div>

        <form onSubmit={handleSubmit} className="space-y-6">
          {apiError && (
            <div className="rounded-md border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive">
              {apiError}
            </div>
          )}

          {/* Session & Counterparty */}
          <Card>
            <CardHeader>
              <CardTitle>{t("mm.omo.create.dealInfo")}</CardTitle>
            </CardHeader>
            <CardContent className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <Label>{t("mm.omo.create.sessionName")} *</Label>
                <Input
                  value={form.session_name}
                  onChange={(e) => set("session_name", e.target.value)}
                  placeholder={t("mm.omo.create.sessionNamePlaceholder")}
                  required
                />
              </div>

              <div className="space-y-2">
                <Label>{t("mm.omo.create.counterparty")}</Label>
                <Input
                  value="Sở giao dịch NHNN"
                  readOnly
                  disabled
                  className="bg-muted"
                />
              </div>
            </CardContent>
          </Card>

          {/* Bond Information */}
          <Card>
            <CardHeader>
              <CardTitle>{t("mm.omo.create.bondInfo")}</CardTitle>
            </CardHeader>
            <CardContent className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <div className="space-y-2 md:col-span-2">
                <Label>{t("mm.omo.create.selectBond")} *</Label>
                <Popover open={bondOpen} onOpenChange={setBondOpen}>
                  <PopoverTrigger
                    className="flex h-9 w-full items-center justify-between rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-xs ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    <span className="truncate">{selectedBondLabel}</span>
                    <IconSelector className="ml-2 size-4 shrink-0 opacity-50" />
                  </PopoverTrigger>
                  <PopoverContent className="w-[var(--anchor-width)] p-0" align="start">
                    <Command>
                      <CommandInput placeholder={t("mm.omo.create.selectBond")} />
                      <CommandList>
                        <CommandEmpty>Không tìm thấy trái phiếu</CommandEmpty>
                        <CommandGroup>
                          {bondCatalog.map((bond) => (
                            <CommandItem
                              key={bond.id}
                              value={`${bond.bond_code} ${bond.issuer}`}
                              data-checked={form.bond_catalog_id === bond.id}
                              onSelect={() => handleBondSelect(bond.id)}
                            >
                              {bond.bond_code} — {bond.issuer}
                            </CommandItem>
                          ))}
                        </CommandGroup>
                      </CommandList>
                    </Command>
                  </PopoverContent>
                </Popover>
              </div>

              <div className="space-y-2">
                <Label>{t("mm.omo.create.bondIssuer")}</Label>
                <Input value={form.bond_issuer} readOnly disabled className="bg-muted" />
              </div>

              <div className="space-y-2">
                <Label>{t("mm.omo.create.couponRate")}</Label>
                <Input value={form.bond_coupon_rate} readOnly disabled className="bg-muted" />
              </div>

              {form.bond_maturity_date && (
                <div className="space-y-2">
                  <Label>{t("mm.omo.detail.bondMaturityDate")}</Label>
                  <Input value={form.bond_maturity_date} readOnly disabled className="bg-muted" />
                </div>
              )}
            </CardContent>
          </Card>

          {/* Financial Details */}
          <Card>
            <CardHeader>
              <CardTitle>{t("mm.omo.create.financials")}</CardTitle>
            </CardHeader>
            <CardContent className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <Label>{t("mm.omo.create.notionalAmount")} *</Label>
                <Input
                  type="number"
                  min="0"
                  step="any"
                  value={form.notional_amount}
                  onChange={(e) => set("notional_amount", e.target.value)}
                  placeholder="0.00"
                  required
                />
              </div>

              <div className="space-y-2">
                <Label>{t("mm.omo.create.winningRate")} *</Label>
                <Input
                  type="number"
                  min="0"
                  step="any"
                  value={form.winning_rate}
                  onChange={(e) => set("winning_rate", e.target.value)}
                  placeholder="0.0000"
                  required
                />
              </div>

              <div className="space-y-2">
                <Label>{t("mm.omo.create.haircutPct")} *</Label>
                <Input
                  type="number"
                  min="0"
                  step="any"
                  value={form.haircut_pct}
                  onChange={(e) => set("haircut_pct", e.target.value)}
                  placeholder="0.00"
                  required
                />
              </div>
            </CardContent>
          </Card>

          {/* Tenor & Settlement Dates */}
          <Card>
            <CardHeader>
              <CardTitle>{t("mm.omo.create.dates")}</CardTitle>
            </CardHeader>
            <CardContent className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
              <div className="space-y-2">
                <Label>{t("mm.omo.detail.tradeDate")} *</Label>
                <DatePicker
                  value={form.trade_date}
                  onChange={(v) => set("trade_date", v)}
                />
              </div>
              <div className="space-y-2">
                <Label>{t("mm.omo.create.tenorDays")} *</Label>
                <Input
                  type="number"
                  min="1"
                  step="1"
                  value={form.tenor_days}
                  onChange={(e) => set("tenor_days", e.target.value)}
                  placeholder="7"
                  required
                />
              </div>
              <div className="space-y-2">
                <Label>{t("mm.omo.create.settlementDate1")} *</Label>
                <DatePicker
                  value={form.settlement_date_1}
                  onChange={(v) => set("settlement_date_1", v)}
                />
              </div>
              <div className="space-y-2">
                <Label>{t("mm.omo.create.settlementDate2")} *</Label>
                <DatePicker
                  value={form.settlement_date_2}
                  onChange={(v) => set("settlement_date_2", v)}
                />
              </div>
            </CardContent>
          </Card>

          {/* Note */}
          <Card>
            <CardHeader>
              <CardTitle>{t("mm.omo.create.note")}</CardTitle>
            </CardHeader>
            <CardContent>
              <Textarea
                value={form.note}
                onChange={(e) => set("note", e.target.value)}
                placeholder={t("mm.omo.create.notePlaceholder")}
                rows={3}
              />
            </CardContent>
          </Card>

          {/* Actions */}
          <div className="flex justify-end gap-3">
            <Button type="button" variant="outline" onClick={() => router.push("/mm")}>
              {t("common.cancel")}
            </Button>
            <Button type="submit" disabled={createMutation.isPending}>
              {createMutation.isPending ? t("common.saving") : t("mm.omo.create.submit")}
            </Button>
          </div>
        </form>
      </div>
    </>
  );
}
