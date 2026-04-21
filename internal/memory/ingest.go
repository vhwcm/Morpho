package memory

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func IngestRun(ctx context.Context, provider EmbeddingProvider, input IngestInput) error {
	if strings.TrimSpace(input.Agent) == "" {
		return fmt.Errorf("agent é obrigatório")
	}
	text := ExtractKnowledge(input.Task, input.Output)
	if strings.TrimSpace(text) == "" {
		return nil
	}

	var ttl *time.Time
	if input.TTLHours > 0 {
		expiresAt := time.Now().UTC().Add(time.Duration(input.TTLHours) * time.Hour)
		ttl = &expiresAt
	}

	docID, err := UpsertDocument(input.Agent, "output", input.Source, text, 0.7, ttl)
	if err != nil {
		return err
	}

	maxChunk := 700
	if input.MaxChars > 200 && input.MaxChars < 3000 {
		maxChunk = input.MaxChars
	}
	chunks := SplitIntoChunks(text, maxChunk, maxChunk/6)
	prepared := make([]ChunkInput, 0, len(chunks))
	for _, ch := range chunks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		emb := []float64(nil)
		if provider != nil {
			v, err := provider.Embed(ctx, ch)
			if err == nil {
				emb = v
			}
		}
		prepared = append(prepared, ChunkInput{
			Text:      ch,
			Tokens:    estimateTokens(ch),
			Embedding: emb,
			Hash:      HashChunk(ch),
		})
	}

	return InsertChunks(input.Agent, docID, prepared)
}
