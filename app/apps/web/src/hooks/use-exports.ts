"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";

export interface ExportRecord {
  id: string;
  code: string;
  module: string;
  created_at: string;
  record_count: number;
  file_size: number;
  created_by: string;
}

interface ExportListResponse {
  data: ExportRecord[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

interface ApiResponse<T> {
  success: boolean;
  data: T;
}

export function useExportHistory(page: number, module?: string) {
  const params: Record<string, string> = {
    page: String(page),
    page_size: "20",
  };
  if (module) params.module = module;

  return useQuery({
    queryKey: ["exports", page, module],
    queryFn: async () => {
      const res = await api.get<ApiResponse<ExportListResponse>>("/exports", {
        params,
      });
      return res.data;
    },
  });
}

export function downloadExport(code: string) {
  const url = `${process.env.NEXT_PUBLIC_API_URL || "/api/v1"}/exports/${code}/download`;
  const a = document.createElement("a");
  a.href = url;
  a.download = "";
  a.click();
}
