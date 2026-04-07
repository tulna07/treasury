"use client";

import { useState } from "react";
import { Header } from "@/components/header";
import { useTranslation } from "@/lib/i18n";
import { useAbility } from "@/hooks/use-ability";
import { useBondCatalog, useCreateBondCatalogItem, useUpdateBondCatalogItem } from "@/hooks/use-bonds";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogClose,
} from "@/components/ui/dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { DatePicker } from "@/components/ui/date-picker";
import { IconPlus, IconEdit, IconAlertCircle, IconBook } from "@tabler/icons-react";
import { toast } from "sonner";
import { extractErrorMessage, formatDate } from "@/lib/utils";
import type { BondCatalogItem } from "@/hooks/use-bonds";

interface CatalogFormState {
  bond_code: string;
  issuer: string;
  coupon_rate: string;
  issue_date: string;
  maturity_date: string;
  face_value: string;
}

const EMPTY_FORM: CatalogFormState = {
  bond_code: "",
  issuer: "",
  coupon_rate: "",
  issue_date: "",
  maturity_date: "",
  face_value: "",
};

export function BondCatalogManager() {
  const { t } = useTranslation();
  const ability = useAbility();
  const { data: catalog, isLoading, isError, error } = useBondCatalog();
  const createMutation = useCreateBondCatalogItem();
  const updateMutation = useUpdateBondCatalogItem();

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [form, setForm] = useState<CatalogFormState>(EMPTY_FORM);

  const canManage = ability.can("manage", "Settings");

  function openCreate() {
    setEditingId(null);
    setForm(EMPTY_FORM);
    setDialogOpen(true);
  }

  function openEdit(item: BondCatalogItem) {
    setEditingId(item.id);
    setForm({
      bond_code: item.bond_code,
      issuer: item.issuer,
      coupon_rate: item.coupon_rate,
      issue_date: item.issue_date,
      maturity_date: item.maturity_date,
      face_value: item.face_value,
    });
    setDialogOpen(true);
  }

  function handleSubmit() {
    const data = {
      bond_code: form.bond_code,
      issuer: form.issuer,
      coupon_rate: parseFloat(form.coupon_rate) || 0,
      issue_date: form.issue_date,
      maturity_date: form.maturity_date,
      face_value: parseFloat(form.face_value) || 0,
    };

    if (editingId) {
      updateMutation.mutate(
        { id: editingId, ...data },
        {
          onSuccess: () => {
            toast.success(t("bond.catalog.updateSuccess"));
            setDialogOpen(false);
          },
          onError: (err) => toast.error(extractErrorMessage(err)),
        }
      );
    } else {
      createMutation.mutate(data, {
        onSuccess: () => {
          toast.success(t("bond.catalog.createSuccess"));
          setDialogOpen(false);
        },
        onError: (err) => toast.error(extractErrorMessage(err)),
      });
    }
  }

  const isSubmitting = createMutation.isPending || updateMutation.isPending;

  return (
    <>
      <Header />
      <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
        <div className="flex flex-col gap-4 pt-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight">{t("bond.catalog.title")}</h1>
            <p className="text-muted-foreground">{t("bond.catalog.description")}</p>
          </div>
          {canManage && (
            <Button onClick={openCreate} className="shrink-0">
              <IconPlus className="mr-2 size-4" />
              {t("bond.catalog.add")}
            </Button>
          )}
        </div>

        {isLoading && (
          <div className="space-y-2">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} className="h-12 w-full" />
            ))}
          </div>
        )}

        {isError && (
          <Alert variant="destructive">
            <IconAlertCircle className="size-4" />
            <AlertDescription>{extractErrorMessage(error)}</AlertDescription>
          </Alert>
        )}

        {!isLoading && !isError && (
          <>
            {/* Desktop table */}
            <div className="hidden md:block">
              <Card>
                <CardContent className="p-0">
                  <div className="rounded-lg border overflow-x-auto">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>{t("bond.bondCode")}</TableHead>
                          <TableHead>{t("bond.issuer")}</TableHead>
                          <TableHead className="text-right">{t("bond.couponRate")}</TableHead>
                          <TableHead>{t("bond.issueDate")}</TableHead>
                          <TableHead>{t("bond.maturityDate")}</TableHead>
                          <TableHead className="text-right">{t("bond.faceValue")}</TableHead>
                          {canManage && <TableHead className="w-16" />}
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {(!catalog || catalog.length === 0) ? (
                          <TableRow>
                            <TableCell colSpan={canManage ? 7 : 6} className="h-32 text-center text-muted-foreground">
                              <IconBook className="mx-auto size-8 mb-2" />
                              {t("bond.catalog.empty")}
                            </TableCell>
                          </TableRow>
                        ) : (
                          catalog.map((item) => (
                            <TableRow key={item.id}>
                              <TableCell className="font-mono font-medium">{item.bond_code}</TableCell>
                              <TableCell>{item.issuer}</TableCell>
                              <TableCell className="text-right tabular-nums">{item.coupon_rate}%</TableCell>
                              <TableCell>{formatDate(item.issue_date)}</TableCell>
                              <TableCell>{formatDate(item.maturity_date)}</TableCell>
                              <TableCell className="text-right tabular-nums">
                                {Number(item.face_value).toLocaleString("en-US")}
                              </TableCell>
                              {canManage && (
                                <TableCell>
                                  <Button variant="ghost" size="icon" className="size-8" onClick={() => openEdit(item)}>
                                    <IconEdit className="size-4" />
                                  </Button>
                                </TableCell>
                              )}
                            </TableRow>
                          ))
                        )}
                      </TableBody>
                    </Table>
                  </div>
                </CardContent>
              </Card>
            </div>

            {/* Mobile cards */}
            <div className="md:hidden space-y-3">
              {(!catalog || catalog.length === 0) ? (
                <div className="text-center py-8 text-muted-foreground">
                  <IconBook className="mx-auto size-8 mb-2" />
                  {t("bond.catalog.empty")}
                </div>
              ) : (
                catalog.map((item) => (
                  <Card key={item.id}>
                    <CardHeader className="pb-2 flex-row items-center justify-between">
                      <CardTitle className="text-sm font-mono">{item.bond_code}</CardTitle>
                      {canManage && (
                        <Button variant="ghost" size="icon" className="size-8" onClick={() => openEdit(item)}>
                          <IconEdit className="size-4" />
                        </Button>
                      )}
                    </CardHeader>
                    <CardContent className="text-sm space-y-1">
                      <div className="text-muted-foreground">{item.issuer}</div>
                      <div className="flex justify-between">
                        <span className="text-muted-foreground">{t("bond.couponRate")}</span>
                        <span className="tabular-nums">{item.coupon_rate}%</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-muted-foreground">{t("bond.maturityDate")}</span>
                        <span>{formatDate(item.maturity_date)}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-muted-foreground">{t("bond.faceValue")}</span>
                        <span className="tabular-nums">{Number(item.face_value).toLocaleString("en-US")}</span>
                      </div>
                    </CardContent>
                  </Card>
                ))
              )}
            </div>
          </>
        )}

        {/* Create/Edit Dialog */}
        <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
          <DialogContent className="sm:max-w-md">
            <DialogHeader>
              <DialogTitle>
                {editingId ? t("bond.catalog.editTitle") : t("bond.catalog.addTitle")}
              </DialogTitle>
            </DialogHeader>
            <div className="grid gap-4">
              <div className="space-y-1.5">
                <Label>{t("bond.bondCode")}</Label>
                <Input
                  value={form.bond_code}
                  onChange={(e) => setForm({ ...form, bond_code: e.target.value })}
                  placeholder="TD2426001"
                />
              </div>
              <div className="space-y-1.5">
                <Label>{t("bond.issuer")}</Label>
                <Input
                  value={form.issuer}
                  onChange={(e) => setForm({ ...form, issuer: e.target.value })}
                  placeholder={t("bond.issuerPlaceholder")}
                />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1.5">
                  <Label>{t("bond.couponRate")}</Label>
                  <Input
                    type="number"
                    step="0.01"
                    value={form.coupon_rate}
                    onChange={(e) => setForm({ ...form, coupon_rate: e.target.value })}
                    placeholder="5.50"
                  />
                </div>
                <div className="space-y-1.5">
                  <Label>{t("bond.faceValue")}</Label>
                  <Input
                    type="number"
                    value={form.face_value}
                    onChange={(e) => setForm({ ...form, face_value: e.target.value })}
                    placeholder="100000"
                  />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1.5">
                  <Label>{t("bond.issueDate")}</Label>
                  <DatePicker
                    value={form.issue_date}
                    onChange={(val) => setForm({ ...form, issue_date: val })}
                    placeholder={t("bond.issueDate")}
                  />
                </div>
                <div className="space-y-1.5">
                  <Label>{t("bond.maturityDate")}</Label>
                  <DatePicker
                    value={form.maturity_date}
                    onChange={(val) => setForm({ ...form, maturity_date: val })}
                    placeholder={t("bond.maturityDate")}
                  />
                </div>
              </div>
            </div>
            <DialogFooter>
              <DialogClose render={<Button variant="outline" />}>
                {t("common.cancel")}
              </DialogClose>
              <Button
                onClick={handleSubmit}
                disabled={!form.bond_code || !form.issuer || isSubmitting}
              >
                {editingId ? t("bond.catalog.save") : t("bond.catalog.add")}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>
    </>
  );
}
