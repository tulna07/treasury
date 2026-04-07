"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "/api/v1";

export interface Attachment {
  id: string;
  deal_module: string;
  deal_id: string;
  file_name: string;
  file_size: number;
  content_type: string;
  uploaded_by: string;
  created_at: string;
  download_url: string;
}

interface ApiResponse<T> {
  success: boolean;
  data: T;
}

// ─── Query keys ───────────────────────────────────────────────

export const attachmentKeys = {
  all: ["attachments"] as const,
  byDeal: (module: string, dealId: string) =>
    [...attachmentKeys.all, module, dealId] as const,
};

// ─── Hooks ────────────────────────────────────────────────────

export function useAttachments(module: string, dealId: string) {
  return useQuery({
    queryKey: attachmentKeys.byDeal(module, dealId),
    queryFn: async () => {
      const res = await fetch(
        `${API_BASE_URL}/attachments/deal/${module}/${dealId}`,
        { credentials: "include" }
      );
      if (!res.ok) throw new Error(`Failed to load attachments: ${res.status}`);
      const json: ApiResponse<Attachment[]> = await res.json();
      return json.data;
    },
    enabled: !!dealId,
  });
}

export function useUploadAttachments() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (params: {
      dealModule: string;
      dealId: string;
      files: File[];
    }) => {
      const formData = new FormData();
      formData.append("deal_module", params.dealModule);
      formData.append("deal_id", params.dealId);
      params.files.forEach((f) => formData.append("files", f));

      const res = await fetch(`${API_BASE_URL}/attachments/upload`, {
        method: "POST",
        body: formData,
        credentials: "include",
      });
      if (!res.ok) {
        const body = await res.json().catch(() => null);
        throw new Error(
          body?.error?.message || `Upload failed: ${res.status}`
        );
      }
      const json: ApiResponse<Attachment[]> = await res.json();
      return json.data;
    },
    onSuccess: (_data, vars) => {
      queryClient.invalidateQueries({
        queryKey: attachmentKeys.byDeal(vars.dealModule, vars.dealId),
      });
    },
  });
}

export function useDeleteAttachment() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (params: {
      id: string;
      dealModule: string;
      dealId: string;
    }) => {
      const res = await fetch(`${API_BASE_URL}/attachments/${params.id}`, {
        method: "DELETE",
        credentials: "include",
      });
      if (!res.ok && res.status !== 204) {
        const body = await res.json().catch(() => null);
        throw new Error(
          body?.error?.message || `Delete failed: ${res.status}`
        );
      }
    },
    onSuccess: (_data, vars) => {
      queryClient.invalidateQueries({
        queryKey: attachmentKeys.byDeal(vars.dealModule, vars.dealId),
      });
    },
  });
}

export function downloadAttachment(id: string) {
  const url = `${API_BASE_URL}/attachments/${id}/download`;
  const a = document.createElement("a");
  a.href = url;
  a.download = "";
  a.click();
}
