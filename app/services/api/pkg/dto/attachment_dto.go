package dto

import (
	"time"

	"github.com/google/uuid"
)

// AttachmentResponse is the API response for a deal attachment.
type AttachmentResponse struct {
	ID          uuid.UUID `json:"id"`
	DealModule  string    `json:"deal_module"`
	DealID      uuid.UUID `json:"deal_id"`
	FileName    string    `json:"file_name"`
	FileSize    int64     `json:"file_size"`
	ContentType string    `json:"content_type"`
	UploadedBy  uuid.UUID `json:"uploaded_by"`
	CreatedAt   time.Time `json:"created_at"`
	DownloadURL string    `json:"download_url"`
}
