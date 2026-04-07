"use client";

import { useCallback, useState, useRef } from "react";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { useTranslation } from "@/lib/i18n";
import { useUploadAttachments } from "@/hooks/use-attachments";
import {
  IconUpload,
  IconX,
  IconFile,
  IconAlertCircle,
  IconLoader2,
} from "@tabler/icons-react";
import { toast } from "sonner";
import { extractErrorMessage } from "@/lib/utils";

const MAX_FILES = 5;
const MAX_SIZE = 10 * 1024 * 1024; // 10 MB
const ALLOWED_EXTENSIONS = ".pdf,.jpg,.jpeg,.png,.gif,.txt,.doc,.docx,.xls,.xlsx";

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

interface BondAttachmentUploadProps {
  dealId: string;
  onUploaded?: () => void;
}

export function BondAttachmentUpload({
  dealId,
  onUploaded,
}: BondAttachmentUploadProps) {
  const { t } = useTranslation();
  const uploadMutation = useUploadAttachments();
  const [selectedFiles, setSelectedFiles] = useState<File[]>([]);
  const [dragActive, setDragActive] = useState(false);
  const [validationError, setValidationError] = useState<string | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const validateFiles = useCallback(
    (files: File[]): string | null => {
      if (selectedFiles.length + files.length > MAX_FILES) {
        return t("attachment.maxFiles");
      }
      for (const file of files) {
        if (file.size > MAX_SIZE) {
          return `${file.name}: ${t("attachment.tooLarge")}`;
        }
        const ext = file.name.split(".").pop()?.toLowerCase();
        const validExt = ["pdf", "jpg", "jpeg", "png", "gif", "txt", "doc", "docx", "xls", "xlsx"];
        if (!validExt.includes(ext || "")) {
          return `${file.name}: ${t("attachment.invalidType")}`;
        }
      }
      return null;
    },
    [selectedFiles.length, t]
  );

  const addFiles = useCallback(
    (newFiles: FileList | File[]) => {
      const arr = Array.from(newFiles);
      const err = validateFiles(arr);
      if (err) {
        setValidationError(err);
        return;
      }
      setValidationError(null);
      setSelectedFiles((prev) => [...prev, ...arr]);
    },
    [validateFiles]
  );

  const removeFile = useCallback((index: number) => {
    setSelectedFiles((prev) => prev.filter((_, i) => i !== index));
    setValidationError(null);
  }, []);

  const handleUpload = useCallback(() => {
    if (selectedFiles.length === 0) return;
    uploadMutation.mutate(
      { dealModule: "BOND", dealId, files: selectedFiles },
      {
        onSuccess: () => {
          toast.success(t("attachment.uploadSuccess"));
          setSelectedFiles([]);
          onUploaded?.();
        },
        onError: (err) => {
          toast.error(extractErrorMessage(err));
        },
      }
    );
  }, [selectedFiles, dealId, uploadMutation, t, onUploaded]);

  const handleDrag = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.type === "dragenter" || e.type === "dragover") {
      setDragActive(true);
    } else if (e.type === "dragleave") {
      setDragActive(false);
    }
  }, []);

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
      setDragActive(false);
      if (e.dataTransfer.files && e.dataTransfer.files.length > 0) {
        addFiles(e.dataTransfer.files);
      }
    },
    [addFiles]
  );

  return (
    <div className="space-y-3">
      <div
        className={`relative rounded-lg border-2 border-dashed p-6 text-center transition-colors ${
          dragActive
            ? "border-primary bg-primary/5"
            : "border-muted-foreground/25 hover:border-muted-foreground/50"
        }`}
        onDragEnter={handleDrag}
        onDragOver={handleDrag}
        onDragLeave={handleDrag}
        onDrop={handleDrop}
        onClick={() => inputRef.current?.click()}
      >
        <input
          ref={inputRef}
          type="file"
          multiple
          accept={ALLOWED_EXTENSIONS}
          className="hidden"
          onChange={(e) => e.target.files && addFiles(e.target.files)}
        />
        <IconUpload className="mx-auto size-8 text-muted-foreground" />
        <p className="mt-2 text-sm font-medium">{t("attachment.dragDrop")}</p>
        <p className="mt-1 text-xs text-muted-foreground">
          {t("attachment.maxSize")} · {t("attachment.allowedTypes")}
        </p>
        <p className="mt-1 text-xs text-muted-foreground">
          {selectedFiles.length}/{MAX_FILES} {t("attachment.files")}
        </p>
      </div>

      {validationError && (
        <Alert variant="destructive">
          <IconAlertCircle className="size-4" />
          <AlertDescription>{validationError}</AlertDescription>
        </Alert>
      )}

      {selectedFiles.length > 0 && (
        <div className="space-y-2">
          {selectedFiles.map((file, idx) => (
            <div
              key={`${file.name}-${idx}`}
              className="flex items-center gap-2 rounded-md border px-3 py-2 text-sm"
            >
              <IconFile className="size-4 shrink-0 text-muted-foreground" />
              <span className="truncate flex-1">{file.name}</span>
              <span className="shrink-0 text-xs text-muted-foreground">
                {formatFileSize(file.size)}
              </span>
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation();
                  removeFile(idx);
                }}
                className="shrink-0 rounded p-0.5 hover:bg-muted"
              >
                <IconX className="size-3.5" />
              </button>
            </div>
          ))}

          <div className="flex justify-end">
            <Button
              onClick={handleUpload}
              disabled={uploadMutation.isPending}
              size="sm"
              className="w-full sm:w-auto"
            >
              {uploadMutation.isPending ? (
                <>
                  <IconLoader2 className="mr-2 size-4 animate-spin" />
                  {t("attachment.uploading")}
                </>
              ) : (
                <>
                  <IconUpload className="mr-2 size-4" />
                  {t("attachment.upload")} ({selectedFiles.length})
                </>
              )}
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
