package memory

import "time"

type Document struct {
	ID         int64
	Agent      string
	SourceType string
	SourceRef  string
	Content    string
	Importance float64
	TTL        *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type Chunk struct {
	ID         int64
	DocumentID int64
	Text       string
	Tokens     int
	Embedding  []float64
	Hash       string
	CreatedAt  time.Time
}

type ChunkInput struct {
	Text      string
	Tokens    int
	Embedding []float64
	Hash      string
}

type SearchResult struct {
	ChunkID       int64
	DocumentID    int64
	Text          string
	Score         float64
	SemanticScore float64
	LexicalScore  float64
	SourceRef     string
	Agent         string
}

type Stats struct {
	Documents int
	Chunks    int
}

type IngestInput struct {
	Agent    string
	Task     string
	Output   string
	Source   string
	MaxChars int
	TTLHours int
}

type RetrieverConfig struct {
	TopK     int
	MinScore float64
	MaxChars int
}

type Embedder interface {
	Embed(text string) ([]float64, error)
}
