package event

import "github.com/google/uuid"

const EvtChatCompleted = "chat.completed"

// ChatCompleted fires after every successful assistant reply (streaming or not).
// It carries enough context for the memory extractor to do its work without
// needing to re-query the database.
type ChatCompleted struct {
	UserID    uuid.UUID
	ConvID    uuid.UUID
	UserMsg   string
	AssistMsg string
}

func (e ChatCompleted) EventName() string { return EvtChatCompleted }
