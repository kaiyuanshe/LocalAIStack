package rag

type Chunk struct {
	ID        string         `json:"id"`
	Text      string         `json:"text"`
	SourceURI string         `json:"source_uri"`
	Title     string         `json:"title,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type Index struct {
	Version string  `json:"version"`
	Chunks  []Chunk `json:"chunks"`
}

type Citation struct {
	ChunkID   string  `json:"chunk_id"`
	SourceURI string  `json:"source_uri"`
	Title     string  `json:"title,omitempty"`
	Score     float64 `json:"score"`
	Snippet   string  `json:"snippet"`
}

type QueryResponse struct {
	Answer    string     `json:"answer"`
	Citations []Citation `json:"citations"`
	Found     bool       `json:"found"`
}

type QueryOptions struct {
	TopK    int
	BaseURL string
	Model   string
}
