package dto

type UploadVoiceRequest struct {
	ConversationID string `form:"conversation_id" binding:"required"`
	Language       string `form:"language"`        // optional hint e.g. "en", "es"
}

type UploadVoiceResponse struct {
	SessionID string `json:"session_id"`
	Status    string `json:"status"`
}

type VoiceSessionResponse struct {
	ID             string  `json:"id"`
	ConversationID string  `json:"conversation_id"`
	Status         string  `json:"status"`
	Transcript     string  `json:"transcript,omitempty"`
	DurationSecs   float64 `json:"duration_seconds,omitempty"`
	LanguageCode   string  `json:"language_code,omitempty"`
	CreatedAt      string  `json:"created_at"`
}
