"use client";

import { useState } from "react";
import { Header } from "@/components/header";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import {
  IconAlertCircle,
  IconDownload,
  IconFileSpreadsheet,
} from "@tabler/icons-react";
import { useTranslation } from "@/lib/i18n";
import { formatDate, extractErrorMessage } from "@/lib/utils";
import { useExportHistory, downloadExport } from "@/hooks/use-exports";
import { PaginationBar } from "@/components/pagination";

const MODULES = ["FX", "GTCG", "MM", "SETTLEMENTS"] as const;

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export default function ExportsPage() {
  const { t } = useTranslation();
  const [page, setPage] = useState(1);
  const [module, setModule] = useState<string | undefined>();

  const { data, isLoading, isError, error } = useExportHistory(page, module);
  const exports = data?.data ?? [];
  const total = data?.total ?? 0;

  return (
    <>
      <Header />
      <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
        <div className="flex flex-col gap-4 pt-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight">
              {t("exports.title")}
            </h1>
            <p className="text-muted-foreground">{t("exports.description")}</p>
          </div>
        </div>

        <div className="flex items-center gap-3">
          <Select
            value={module || "ALL"}
            onValueChange={(v) => {
              setModule(!v || v === "ALL" ? undefined : v);
              setPage(1);
            }}
          >
            <SelectTrigger className="w-full sm:w-auto">
              <SelectValue placeholder={t("exports.filterByModule")}>{(v: string) => v === "ALL" ? t("common.all") : v}</SelectValue>
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="ALL" label={t("common.all")}>{t("common.all")}</SelectItem>
              {MODULES.map((m) => (
                <SelectItem key={m} value={m} label={m}>
                  {m}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {isError && (
          <Alert variant="destructive">
            <IconAlertCircle className="size-4" />
            <AlertDescription>{extractErrorMessage(error)}</AlertDescription>
          </Alert>
        )}

        {isLoading ? (
          <div className="space-y-3">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} className="h-12 w-full" />
            ))}
          </div>
        ) : exports.length === 0 ? (
          <Card>
            <CardContent className="py-12 text-center">
              <IconFileSpreadsheet className="mx-auto size-10 text-muted-foreground/50 mb-3" />
              <p className="text-muted-foreground">{t("exports.empty")}</p>
            </CardContent>
          </Card>
        ) : (
          <>
            {/* Desktop table */}
            <div className="hidden sm:block rounded-lg border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t("exports.code")}</TableHead>
                    <TableHead>{t("exports.module")}</TableHead>
                    <TableHead>{t("exports.createdAt")}</TableHead>
                    <TableHead className="text-right">{t("exports.records")}</TableHead>
                    <TableHead className="text-right">{t("exports.fileSize")}</TableHead>
                    <TableHead className="text-right">{t("exports.download")}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {exports.map((exp) => (
                    <TableRow key={exp.id}>
                      <TableCell className="font-mono text-sm">
                        {exp.code}
                      </TableCell>
                      <TableCell>{exp.module}</TableCell>
                      <TableCell>{formatDate(exp.created_at)}</TableCell>
                      <TableCell className="text-right tabular-nums">
                        {exp.record_count.toLocaleString("en-US")}
                      </TableCell>
                      <TableCell className="text-right tabular-nums">
                        {formatFileSize(exp.file_size)}
                      </TableCell>
                      <TableCell className="text-right">
                        <Button
                          variant="ghost"
                          size="icon"
                          className="size-8"
                          onClick={() => downloadExport(exp.code)}
                        >
                          <IconDownload className="size-4" />
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>

            {/* Mobile cards */}
            <div className="sm:hidden space-y-3">
              {exports.map((exp) => (
                <Card key={exp.id}>
                  <CardContent className="p-3 space-y-2">
                    <div className="flex items-center justify-between">
                      <span className="font-mono text-sm font-medium">
                        {exp.code}
                      </span>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="size-8"
                        onClick={() => downloadExport(exp.code)}
                      >
                        <IconDownload className="size-4" />
                      </Button>
                    </div>
                    <div className="text-sm text-muted-foreground">
                      {exp.module} · {exp.record_count.toLocaleString("en-US")}{" "}
                      {t("exports.records").toLowerCase()} ·{" "}
                      {formatFileSize(exp.file_size)}
                    </div>
                    <div className="text-xs text-muted-foreground">
                      {formatDate(exp.created_at)}
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          </>
        )}

        <PaginationBar
          page={page}
          total={total}
          pageSize={20}
          onPageChange={setPage}
        />
      </div>
    </>
  );
}
