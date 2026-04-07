"use client";

import { Button } from "@/components/ui/button";
import { useTranslation } from "@/lib/i18n";
import {
  useAttachments,
  useDeleteAttachment,
  downloadAttachment,
  type Attachment,
} from "@/hooks/use-attachments";
import {
  IconDownload,
  IconTrash,
  IconFileTypePdf,
  IconPhoto,
  IconFileSpreadsheet,
  IconFileText,
  IconFile,
  IconExternalLink,
} from "@tabler/icons-react";
import { toast } from "sonner";
import { extractErrorMessage } from "@/lib/utils";

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function FileIcon({ contentType }: { contentType: string }) {
  if (contentType === "application/pdf")
    return <IconFileTypePdf className="size-5 text-red-500" />;
  if (contentType.startsWith("image/"))
    return <IconPhoto className="size-5 text-blue-500" />;
  if (
    contentType.includes("spreadsheet") ||
    contentType.includes("excel") ||
    contentType.includes("ms-excel")
  )
    return <IconFileSpreadsheet className="size-5 text-green-600" />;
  if (
    contentType.includes("word") ||
    contentType.includes("document") ||
    contentType === "text/plain"
  )
    return <IconFileText className="size-5 text-blue-600" />;
  return <IconFile className="size-5 text-muted-foreground" />;
}

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "/api/v1";

function PreviewLink({ attachment }: { attachment: Attachment }) {
  const { t } = useTranslation();
  const isImage = attachment.content_type.startsWith("image/");
  const isPdf = attachment.content_type === "application/pdf";

  if (!isImage && !isPdf) return null;

  const url = `${API_BASE_URL}/attachments/${attachment.id}/download`;

  return (
    <a
      href={url}
      target="_blank"
      rel="noopener noreferrer"
      className="text-xs text-primary hover:underline inline-flex items-center gap-1"
    >
      <IconExternalLink className="size-3" />
      {t("attachment.preview")}
    </a>
  );
}

interface FxAttachmentListProps {
  dealId: string;
  dealModule?: string;
  canDelete?: boolean;
  currentUserId?: string;
}

export function FxAttachmentList({
  dealId,
  dealModule = "FX",
  canDelete = false,
  currentUserId,
}: FxAttachmentListProps) {
  const { t } = useTranslation();
  const { data: attachments, isLoading } = useAttachments(dealModule, dealId);
  const deleteMutation = useDeleteAttachment();

  if (isLoading) {
    return (
      <p className="text-sm text-muted-foreground py-2">
        {t("common.loading")}...
      </p>
    );
  }

  if (!attachments || attachments.length === 0) {
    return (
      <p className="text-sm text-muted-foreground py-2">
        {t("attachment.noFiles")}
      </p>
    );
  }

  const handleDelete = (att: Attachment) => {
    if (!confirm(t("attachment.confirmDelete"))) return;
    deleteMutation.mutate(
      { id: att.id, dealModule, dealId },
      {
        onSuccess: () => toast.success(t("attachment.deleteSuccess")),
        onError: (err) => toast.error(extractErrorMessage(err)),
      }
    );
  };

  return (
    <div className="space-y-2">
      {attachments.map((att) => (
        <div
          key={att.id}
          className="flex items-center gap-3 rounded-md border px-3 py-2"
        >
          <FileIcon contentType={att.content_type} />
          <div className="min-w-0 flex-1">
            <p className="truncate text-sm font-medium">{att.file_name}</p>
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <span>{formatFileSize(att.file_size)}</span>
              <PreviewLink attachment={att} />
            </div>
          </div>
          <div className="flex shrink-0 items-center gap-1">
            <Button
              variant="ghost"
              size="icon"
              className="size-8"
              onClick={() => downloadAttachment(att.id)}
              title={t("attachment.download")}
            >
              <IconDownload className="size-4" />
            </Button>
            {canDelete &&
              currentUserId &&
              att.uploaded_by === currentUserId && (
                <Button
                  variant="ghost"
                  size="icon"
                  className="size-8 text-destructive hover:text-destructive"
                  onClick={() => handleDelete(att)}
                  disabled={deleteMutation.isPending}
                  title={t("attachment.delete")}
                >
                  <IconTrash className="size-4" />
                </Button>
              )}
          </div>
        </div>
      ))}
    </div>
  );
}
