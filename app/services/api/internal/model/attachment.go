package model

import (
	"time"

	"github.com/google/uuid"
)

// DealAttachment represents a file attached to a deal.
type DealAttachment struct {
	ID          uuid.UUID
	DealModule  string
	DealID      uuid.UUID
	FileName    string
	FileSize    int64
	ContentType string
	MinioBucket string
	MinioKey    string
	UploadedBy  uuid.UUID
	CreatedAt   time.Time
}
