package event

import (
	"time"

	"github.com/google/uuid"
)

const (
	EvtDocumentUploaded = "document.uploaded"
	EvtDocumentIndexed  = "document.indexed"
	EvtDocumentDeleted  = "document.deleted"
)

type DocumentUploaded struct {
	DocumentID uuid.UUID `json:"document_id"`
	UserID     uuid.UUID `json:"user_id"`
	OccuredAt  time.Time `json:"occured_at"`
}

func (e DocumentUploaded) EventName() string { return EvtDocumentUploaded }

type DocumentIndexed struct {
	DocumentID uuid.UUID `json:"document_id"`
	ChunkCount int       `json:"chunk_count"`
	OccuredAt  time.Time `json:"occured_at"`
}

func (e DocumentIndexed) EventName() string { return EvtDocumentIndexed }

type DocumentDeleted struct {
	DocumentID uuid.UUID `json:"document_id"`
	UserID     uuid.UUID `json:"user_id"`
	OccuredAt  time.Time `json:"occured_at"`
}

func (e DocumentDeleted) EventName() string { return EvtDocumentDeleted }
