"use client";

import { useState, useMemo } from "react";
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
import { IconAlertCircle } from "@tabler/icons-react";
import { DatePicker } from "@/components/ui/date-picker";
import { useTranslation } from "@/lib/i18n";
import type {
  BondDeal,
  BondCategory,
  BondDirection,
  BondTransactionType,
  BondPortfolioType,
  BondConfirmationMethod,
  BondContractPreparedBy,
  CreateBondDealRequest,
} from "@/hooks/use-bonds";

interface Counterparty {
  id: string;
  code: string;
  name: string;
}

interface BondCatalogItem {
  id: string;
  bond_code: string;
  issuer: string;
  coupon_rate: string;
  issue_date: string;
  maturity_date: string;
  face_value: string;
}

interface BondDealFormProps {
  initialData?: BondDeal;
  counterparties: Counterparty[];
  bondCatalog?: BondCatalogItem[];
  onSubmit: (data: CreateBondDealRequest) => void;
  onCancel: () => void;
  isSubmitting: boolean;
  error?: string | null;
  submitLabel: string;
}

const BOND_CATEGORIES: BondCategory[] = ["GOVERNMENT", "FINANCIAL_INSTITUTION", "CERTIFICATE_OF_DEPOSIT"];
const DIRECTIONS: BondDirection[] = ["BUY", "SELL"];
const TRANSACTION_TYPES: BondTransactionType[] = ["REPO", "REVERSE_REPO", "OUTRIGHT", "OTHER"];
const PORTFOLIO_TYPES: BondPortfolioType[] = ["HTM", "AFS", "HFT"];
const CONFIRMATION_METHODS: BondConfirmationMethod[] = ["EMAIL", "REUTERS", "OTHER"];
const CONTRACT_PREPARED_BY: BondContractPreparedBy[] = ["INTERNAL", "COUNTERPARTY"];

interface FormState {
  bond_category: BondCategory;
  trade_date: string;
  order_date: string;
  value_date: string;
  direction: BondDirection;
  counterparty_id: string;
  transaction_type: BondTransactionType;
  transaction_type_other: string;
  bond_catalog_id: string;
  bond_code_manual: string;
  issuer: string;
  coupon_rate: number;
  issue_date: string;
  maturity_date: string;
  quantity: number;
  face_value: number;
  discount_rate: number;
  clean_price: number;
  settlement_price: number;
  total_value: number;
  portfolio_type: BondPortfolioType | "";
  payment_date: string;
  remaining_tenor_days: number;
  confirmation_method: BondConfirmationMethod;
  confirmation_other: string;
  contract_prepared_by: BondContractPreparedBy;
  note: string;
}

function dealToFormState(deal: BondDeal): FormState {
  return {
    bond_category: deal.bond_category,
    trade_date: deal.trade_date.slice(0, 10),
    order_date: deal.order_date?.slice(0, 10) || "",
    value_date: deal.value_date.slice(0, 10),
    direction: deal.direction,
    counterparty_id: deal.counterparty_id,
    transaction_type: deal.transaction_type,
    transaction_type_other: deal.transaction_type_other || "",
    bond_catalog_id: deal.bond_catalog_id || "",
    bond_code_manual: deal.bond_code_manual || "",
    issuer: deal.issuer,
    coupon_rate: Number(deal.coupon_rate),
    issue_date: deal.issue_date?.slice(0, 10) || "",
    maturity_date: deal.maturity_date.slice(0, 10),
    quantity: deal.quantity,
    face_value: Number(deal.face_value),
    discount_rate: Number(deal.discount_rate),
    clean_price: Number(deal.clean_price),
    settlement_price: Number(deal.settlement_price),
    total_value: Number(deal.total_value),
    portfolio_type: deal.portfolio_type || "",
    payment_date: deal.payment_date.slice(0, 10),
    remaining_tenor_days: deal.remaining_tenor_days,
    confirmation_method: deal.confirmation_method,
    confirmation_other: deal.confirmation_other || "",
    contract_prepared_by: deal.contract_prepared_by,
    note: deal.note || "",
  };
}

const defaultForm: FormState = {
  bond_category: "GOVERNMENT",
  trade_date: new Date().toISOString().slice(0, 10),
  order_date: "",
  value_date: "",
  direction: "BUY",
  counterparty_id: "",
  transaction_type: "OUTRIGHT",
  transaction_type_other: "",
  bond_catalog_id: "",
  bond_code_manual: "",
  issuer: "",
  coupon_rate: 0,
  issue_date: "",
  maturity_date: "",
  quantity: 0,
  face_value: 0,
  discount_rate: 0,
  clean_price: 0,
  settlement_price: 0,
  total_value: 0,
  portfolio_type: "",
  payment_date: "",
  remaining_tenor_days: 0,
  confirmation_method: "EMAIL",
  confirmation_other: "",
  contract_prepared_by: "INTERNAL",
  note: "",
};

export function BondDealForm({
  initialData,
  counterparties,
  bondCatalog = [],
  onSubmit,
  onCancel,
  isSubmitting,
  error,
  submitLabel,
}: BondDealFormProps) {
  const { t } = useTranslation();

  const [form, setForm] = useState<FormState>(
    initialData ? dealToFormState(initialData) : defaultForm
  );
  const [validationError, setValidationError] = useState<string | null>(null);

  const isGovernment = form.bond_category === "GOVERNMENT";
  const isBuy = form.direction === "BUY";
  const isTransactionOther = form.transaction_type === "OTHER";
  const isConfirmationOther = form.confirmation_method === "OTHER";

  // Auto-calculate total_value
  const calculatedTotal = useMemo(() => {
    if (form.quantity > 0 && form.settlement_price > 0) {
      return form.quantity * form.settlement_price;
    }
    return 0;
  }, [form.quantity, form.settlement_price]);

  // Auto-calculate remaining_tenor_days
  const calculatedTenor = useMemo(() => {
    if (form.maturity_date && form.payment_date) {
      const maturity = new Date(form.maturity_date);
      const payment = new Date(form.payment_date);
      const diff = Math.ceil((maturity.getTime() - payment.getTime()) / (1000 * 60 * 60 * 24));
      return diff > 0 ? diff : 0;
    }
    return 0;
  }, [form.maturity_date, form.payment_date]);

  function updateField<K extends keyof FormState>(key: K, value: FormState[K]) {
    setForm((prev) => {
      const next = { ...prev, [key]: value };
      // Auto-recalculate
      if (key === "quantity" || key === "settlement_price") {
        const qty = key === "quantity" ? (value as number) : prev.quantity;
        const price = key === "settlement_price" ? (value as number) : prev.settlement_price;
        next.total_value = qty > 0 && price > 0 ? qty * price : 0;
      }
      if (key === "maturity_date" || key === "payment_date") {
        const mat = key === "maturity_date" ? (value as string) : prev.maturity_date;
        const pay = key === "payment_date" ? (value as string) : prev.payment_date;
        if (mat && pay) {
          const diff = Math.ceil((new Date(mat).getTime() - new Date(pay).getTime()) / (1000 * 60 * 60 * 24));
          next.remaining_tenor_days = diff > 0 ? diff : 0;
        }
      }
      return next;
    });
  }

  function handleCatalogSelect(catalogId: string) {
    const item = bondCatalog.find((c) => c.id === catalogId);
    if (item) {
      setForm((prev) => ({
        ...prev,
        bond_catalog_id: catalogId,
        issuer: item.issuer,
        coupon_rate: Number(item.coupon_rate),
        issue_date: item.issue_date?.slice(0, 10) || "",
        maturity_date: item.maturity_date.slice(0, 10),
        face_value: Number(item.face_value),
      }));
    }
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setValidationError(null);

    if (!form.counterparty_id) {
      setValidationError(t("bond.validation.counterparty"));
      return;
    }
    if (!form.trade_date) {
      setValidationError(t("bond.validation.tradeDate"));
      return;
    }
    if (!form.value_date) {
      setValidationError(t("bond.validation.valueDate"));
      return;
    }
    if (!form.maturity_date) {
      setValidationError(t("bond.validation.maturityDate"));
      return;
    }
    if (!form.payment_date) {
      setValidationError(t("bond.validation.paymentDate"));
      return;
    }
    if (form.quantity <= 0) {
      setValidationError(t("bond.validation.quantity"));
      return;
    }
    if (form.settlement_price <= 0) {
      setValidationError(t("bond.validation.settlementPrice"));
      return;
    }
    if (!form.issuer) {
      setValidationError(t("bond.validation.issuer"));
      return;
    }
    if (isGovernment && !form.bond_catalog_id && !form.bond_code_manual) {
      setValidationError(t("bond.validation.bondCode"));
      return;
    }

    const data: CreateBondDealRequest = {
      bond_category: form.bond_category,
      trade_date: form.trade_date,
      order_date: form.order_date || undefined,
      value_date: form.value_date,
      direction: form.direction,
      counterparty_id: form.counterparty_id,
      transaction_type: form.transaction_type,
      transaction_type_other: isTransactionOther ? form.transaction_type_other || undefined : undefined,
      bond_catalog_id: isGovernment && form.bond_catalog_id ? form.bond_catalog_id : undefined,
      bond_code_manual: !isGovernment || !form.bond_catalog_id ? form.bond_code_manual || undefined : undefined,
      issuer: form.issuer,
      coupon_rate: form.coupon_rate,
      issue_date: form.issue_date || undefined,
      maturity_date: form.maturity_date,
      quantity: form.quantity,
      face_value: form.face_value,
      discount_rate: form.discount_rate || undefined,
      clean_price: form.clean_price,
      settlement_price: form.settlement_price,
      total_value: form.total_value || calculatedTotal,
      portfolio_type: isBuy && form.portfolio_type ? (form.portfolio_type as BondPortfolioType) : undefined,
      payment_date: form.payment_date,
      remaining_tenor_days: form.remaining_tenor_days || calculatedTenor,
      confirmation_method: form.confirmation_method,
      confirmation_other: isConfirmationOther ? form.confirmation_other || undefined : undefined,
      contract_prepared_by: form.contract_prepared_by,
      note: form.note || undefined,
    };

    onSubmit(data);
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {(error || validationError) && (
        <Alert variant="destructive">
          <IconAlertCircle className="size-4" />
          <AlertDescription>{error || validationError}</AlertDescription>
        </Alert>
      )}

      {/* Basic Info */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("bond.basicInfo")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            <div className="space-y-2">
              <Label htmlFor="bond_category">{t("bond.category")}</Label>
              <Select
                value={form.bond_category}
                onValueChange={(v) => v && updateField("bond_category", v as BondCategory)}
              >
                <SelectTrigger id="bond_category">
                  <SelectValue>{(v: string) => t(`bond.category.${v}`)}</SelectValue>
                </SelectTrigger>
                <SelectContent>
                  {BOND_CATEGORIES.map((c) => (
                    <SelectItem key={c} value={c} label={t(`bond.category.${c}`)}>
                      {t(`bond.category.${c}`)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="direction">{t("bond.direction")}</Label>
              <Select
                value={form.direction}
                onValueChange={(v) => v && updateField("direction", v as BondDirection)}
              >
                <SelectTrigger id="direction">
                  <SelectValue>{(v: string) => t(`bond.direction.${v}`)}</SelectValue>
                </SelectTrigger>
                <SelectContent>
                  {DIRECTIONS.map((d) => (
                    <SelectItem key={d} value={d} label={t(`bond.direction.${d}`)}>
                      {t(`bond.direction.${d}`)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="counterparty">{t("bond.counterparty")}</Label>
              <Select
                value={form.counterparty_id}
                onValueChange={(v) => updateField("counterparty_id", v ?? "")}
              >
                <SelectTrigger id="counterparty">
                  <SelectValue placeholder={t("bond.selectCounterparty")}>
                    {(value: string) => {
                      const cp = counterparties.find((c) => c.id === value);
                      return cp ? `${cp.code} — ${cp.name}` : value;
                    }}
                  </SelectValue>
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
              <Label htmlFor="transaction_type">{t("bond.transactionType")}</Label>
              <Select
                value={form.transaction_type}
                onValueChange={(v) => v && updateField("transaction_type", v as BondTransactionType)}
              >
                <SelectTrigger id="transaction_type">
                  <SelectValue>{(v: string) => t(`bond.txType.${v}`)}</SelectValue>
                </SelectTrigger>
                <SelectContent>
                  {TRANSACTION_TYPES.map((tt) => (
                    <SelectItem key={tt} value={tt} label={t(`bond.txType.${tt}`)}>
                      {t(`bond.txType.${tt}`)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {isTransactionOther && (
              <div className="space-y-2">
                <Label htmlFor="transaction_type_other">{t("bond.transactionTypeOther")}</Label>
                <Input
                  id="transaction_type_other"
                  value={form.transaction_type_other}
                  onChange={(e) => updateField("transaction_type_other", e.target.value)}
                  placeholder={t("bond.transactionTypeOtherPlaceholder")}
                />
              </div>
            )}

            {isBuy && (
              <div className="space-y-2">
                <Label htmlFor="portfolio_type">{t("bond.portfolioType")}</Label>
                <Select
                  value={form.portfolio_type || "NONE"}
                  onValueChange={(v) => updateField("portfolio_type", (!v || v === "NONE" ? "" : v) as BondPortfolioType | "")}
                >
                  <SelectTrigger id="portfolio_type">
                    <SelectValue placeholder={t("bond.selectPortfolioType")}>{(v: string) => v === "NONE" ? t("bond.selectPortfolioType") : v}</SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="NONE" label={t("bond.selectPortfolioType")}>{t("bond.selectPortfolioType")}</SelectItem>
                    {PORTFOLIO_TYPES.map((pt) => (
                      <SelectItem key={pt} value={pt} label={pt}>
                        {pt}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Bond Details */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("bond.bondInfo")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {isGovernment && bondCatalog.length > 0 ? (
              <div className="space-y-2">
                <Label htmlFor="bond_catalog">{t("bond.bondCode")}</Label>
                <Select
                  value={form.bond_catalog_id || "NONE"}
                  onValueChange={(v) => {
                    if (v && v !== "NONE") handleCatalogSelect(v);
                  }}
                >
                  <SelectTrigger id="bond_catalog">
                    <SelectValue placeholder={t("bond.selectBondCode")}>
                      {(value: string) => {
                        const item = bondCatalog.find((c) => c.id === value);
                        return item ? item.bond_code : value;
                      }}
                    </SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="NONE" label={t("bond.selectBondCode")}>{t("bond.selectBondCode")}</SelectItem>
                    {bondCatalog.map((item) => (
                      <SelectItem key={item.id} value={item.id} label={item.bond_code}>
                        {item.bond_code} — {item.issuer}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            ) : (
              <div className="space-y-2">
                <Label htmlFor="bond_code_manual">{t("bond.bondCode")}</Label>
                <Input
                  id="bond_code_manual"
                  value={form.bond_code_manual}
                  onChange={(e) => updateField("bond_code_manual", e.target.value)}
                  placeholder={t("bond.bondCodePlaceholder")}
                  maxLength={50}
                />
              </div>
            )}

            <div className="space-y-2">
              <Label htmlFor="issuer">{t("bond.issuer")}</Label>
              <Input
                id="issuer"
                value={form.issuer}
                onChange={(e) => updateField("issuer", e.target.value)}
                placeholder={t("bond.issuerPlaceholder")}
                disabled={isGovernment && !!form.bond_catalog_id}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="coupon_rate">{t("bond.couponRate")}</Label>
              <Input
                id="coupon_rate"
                type="number"
                step="0.01"
                min="0"
                value={form.coupon_rate || ""}
                onChange={(e) => updateField("coupon_rate", Number(e.target.value))}
                disabled={isGovernment && !!form.bond_catalog_id}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="face_value">{t("bond.faceValue")}</Label>
              <Input
                id="face_value"
                type="number"
                step="1"
                min="0"
                value={form.face_value || ""}
                onChange={(e) => updateField("face_value", Number(e.target.value))}
                disabled={isGovernment && !!form.bond_catalog_id}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="issue_date">{t("bond.issueDate")}</Label>
              <DatePicker
                id="issue_date"
                value={form.issue_date}
                onChange={(val) => updateField("issue_date", val)}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="maturity_date">{t("bond.maturityDate")}</Label>
              <DatePicker
                id="maturity_date"
                value={form.maturity_date}
                onChange={(val) => updateField("maturity_date", val)}
              />
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Pricing & Settlement */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("bond.pricingInfo")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            <div className="space-y-2">
              <Label htmlFor="quantity">{t("bond.quantity")}</Label>
              <Input
                id="quantity"
                type="number"
                step="1"
                min="1"
                value={form.quantity || ""}
                onChange={(e) => updateField("quantity", Number(e.target.value))}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="clean_price">{t("bond.cleanPrice")}</Label>
              <Input
                id="clean_price"
                type="number"
                step="0.01"
                min="0"
                value={form.clean_price || ""}
                onChange={(e) => updateField("clean_price", Number(e.target.value))}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="discount_rate">{t("bond.discountRate")}</Label>
              <Input
                id="discount_rate"
                type="number"
                step="0.01"
                min="0"
                value={form.discount_rate || ""}
                onChange={(e) => updateField("discount_rate", Number(e.target.value))}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="settlement_price">{t("bond.settlementPrice")}</Label>
              <Input
                id="settlement_price"
                type="number"
                step="0.01"
                min="0"
                value={form.settlement_price || ""}
                onChange={(e) => updateField("settlement_price", Number(e.target.value))}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="total_value">{t("bond.totalValue")}</Label>
              <Input
                id="total_value"
                type="number"
                step="1"
                min="0"
                value={form.total_value || calculatedTotal || ""}
                onChange={(e) => updateField("total_value", Number(e.target.value))}
              />
              {calculatedTotal > 0 && (
                <p className="text-xs text-muted-foreground">
                  {t("bond.autoCalculated")}: {calculatedTotal.toLocaleString("en-US")}
                </p>
              )}
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Dates & Settlement */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("bond.settlementInfo")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            <div className="space-y-2">
              <Label htmlFor="trade_date">{t("bond.tradeDate")}</Label>
              <DatePicker
                id="trade_date"
                value={form.trade_date}
                onChange={(val) => updateField("trade_date", val)}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="order_date">{t("bond.orderDate")}</Label>
              <DatePicker
                id="order_date"
                value={form.order_date}
                onChange={(val) => updateField("order_date", val)}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="value_date">{t("bond.valueDate")}</Label>
              <DatePicker
                id="value_date"
                value={form.value_date}
                onChange={(val) => updateField("value_date", val)}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="payment_date">{t("bond.paymentDate")}</Label>
              <DatePicker
                id="payment_date"
                value={form.payment_date}
                onChange={(val) => updateField("payment_date", val)}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="remaining_tenor_days">{t("bond.remainingTenor")}</Label>
              <Input
                id="remaining_tenor_days"
                type="number"
                min="0"
                value={form.remaining_tenor_days || calculatedTenor || ""}
                onChange={(e) => updateField("remaining_tenor_days", Number(e.target.value))}
              />
              {calculatedTenor > 0 && (
                <p className="text-xs text-muted-foreground">
                  {t("bond.autoCalculated")}: {calculatedTenor} {t("bond.days")}
                </p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="confirmation_method">{t("bond.confirmationMethod")}</Label>
              <Select
                value={form.confirmation_method}
                onValueChange={(v) => v && updateField("confirmation_method", v as BondConfirmationMethod)}
              >
                <SelectTrigger id="confirmation_method">
                  <SelectValue>{(v: string) => t(`bond.confirmation.${v}`)}</SelectValue>
                </SelectTrigger>
                <SelectContent>
                  {CONFIRMATION_METHODS.map((cm) => (
                    <SelectItem key={cm} value={cm} label={t(`bond.confirmation.${cm}`)}>
                      {t(`bond.confirmation.${cm}`)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {isConfirmationOther && (
              <div className="space-y-2">
                <Label htmlFor="confirmation_other">{t("bond.confirmationOther")}</Label>
                <Input
                  id="confirmation_other"
                  value={form.confirmation_other}
                  onChange={(e) => updateField("confirmation_other", e.target.value)}
                />
              </div>
            )}

            <div className="space-y-2">
              <Label htmlFor="contract_prepared_by">{t("bond.contractPreparedBy")}</Label>
              <Select
                value={form.contract_prepared_by}
                onValueChange={(v) => v && updateField("contract_prepared_by", v as BondContractPreparedBy)}
              >
                <SelectTrigger id="contract_prepared_by">
                  <SelectValue>{(v: string) => t(`bond.contractBy.${v}`)}</SelectValue>
                </SelectTrigger>
                <SelectContent>
                  {CONTRACT_PREPARED_BY.map((cp) => (
                    <SelectItem key={cp} value={cp} label={t(`bond.contractBy.${cp}`)}>
                      {t(`bond.contractBy.${cp}`)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="note">{t("bond.note")}</Label>
            <Textarea
              id="note"
              value={form.note}
              onChange={(e) => updateField("note", e.target.value)}
              rows={2}
              maxLength={2000}
            />
          </div>
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
