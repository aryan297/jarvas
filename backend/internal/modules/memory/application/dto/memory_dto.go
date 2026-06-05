package dto

type CreateMemoryRequest struct {
	Type       string  `json:"type"       binding:"required,oneof=FACT PREFERENCE EVENT SKILL RELATIONSHIP"`
	Content    string  `json:"content"    binding:"required,min=1,max=2000"`
	Importance float64 `json:"importance" binding:"min=0,max=1"`
}

type SearchMemoryRequest struct {
	Query  string  `json:"query"   binding:"required,min=1"`
	TopK   int     `json:"top_k"   binding:"min=1,max=20"`
	MinScore float32 `json:"min_score"`
}

type MemoryResponse struct {
	ID          string  `json:"id"`
	Type        string  `json:"type"`
	Content     string  `json:"content"`
	Importance  float64 `json:"importance"`
	AccessCount int     `json:"access_count"`
	CreatedAt   string  `json:"created_at"`
}

type MemorySearchResult struct {
	ID         string  `json:"id"`
	Type       string  `json:"type"`
	Content    string  `json:"content"`
	Importance float64 `json:"importance"`
	Score      float32 `json:"score"`
}
