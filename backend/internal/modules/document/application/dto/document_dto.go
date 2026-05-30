package dto

type DocumentResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	MimeType   string `json:"mime_type"`
	SizeBytes  int64  `json:"size_bytes"`
	Status     string `json:"status"`
	ChunkCount int    `json:"chunk_count"`
	CreatedAt  string `json:"created_at"`
}

type UploadResponse struct {
	DocumentResponse
	UploadURL string `json:"upload_url,omitempty"` // presigned URL for direct upload
}
