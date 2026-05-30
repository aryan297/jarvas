package entity

import (
	"time"

	"github.com/google/uuid"
)

type DocumentStatus string
type DocumentType   string

const (
	StatusUploaded   DocumentStatus = "UPLOADED"
	StatusProcessing DocumentStatus = "PROCESSING"
	StatusIndexed    DocumentStatus = "INDEXED"
	StatusFailed     DocumentStatus = "FAILED"

	TypePDF  DocumentType = "PDF"
	TypeDOCX DocumentType = "DOCX"
	TypeTXT  DocumentType = "TXT"
	TypeMD   DocumentType = "MD"
	TypeHTML DocumentType = "HTML"
	TypeCSV  DocumentType = "CSV"
	TypeOther DocumentType = "OTHER"
)

// Document is the aggregate root for the document bounded context.
type Document struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	Name       string
	Type       DocumentType
	MimeType   string
	SizeBytes  int64
	StorageKey string    // MinIO object key
	Status     DocumentStatus
	ChunkCount int
	ErrorMsg   string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// DocumentChunk represents a single text segment of a document.
type DocumentChunk struct {
	ID         uuid.UUID
	DocumentID uuid.UUID
	UserID     uuid.UUID
	Content    string
	ChunkIndex int
	QdrantID   *uuid.UUID
	TokenCount int
	CreatedAt  time.Time
}

func NewDocument(userID uuid.UUID, name, mimeType, storageKey string, sizeBytes int64) *Document {
	now := time.Now().UTC()
	return &Document{
		ID:         uuid.New(),
		UserID:     userID,
		Name:       name,
		MimeType:   mimeType,
		SizeBytes:  sizeBytes,
		StorageKey: storageKey,
		Status:     StatusUploaded,
		Type:       inferType(mimeType),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func (d *Document) MarkProcessing() {
	d.Status = StatusProcessing
	d.UpdatedAt = time.Now().UTC()
}

func (d *Document) MarkIndexed(chunkCount int) {
	d.Status = StatusIndexed
	d.ChunkCount = chunkCount
	d.UpdatedAt = time.Now().UTC()
}

func (d *Document) MarkFailed(msg string) {
	d.Status = StatusFailed
	d.ErrorMsg = msg
	d.UpdatedAt = time.Now().UTC()
}

func inferType(mime string) DocumentType {
	switch mime {
	case "application/pdf":
		return TypePDF
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return TypeDOCX
	case "text/plain":
		return TypeTXT
	case "text/markdown":
		return TypeMD
	case "text/html":
		return TypeHTML
	case "text/csv":
		return TypeCSV
	default:
		return TypeOther
	}
}
