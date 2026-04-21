package memory

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/vhwcm/Morpho/internal/config"
)

type EmbeddingProvider interface {
	Embed(ctx context.Context, text string) ([]float64, error)
}

const (
	semanticWeight = 0.75
	lexicalWeight  = 0.25
)

func Search(ctx context.Context, agent, query string, topK int, minScore float64, provider EmbeddingProvider) ([]SearchResult, error) {
	return SearchWithPolicy(ctx, agent, query, topK, minScore, provider, config.MemoryReadPolicySelf)
}

func SearchWithPolicy(ctx context.Context, currentAgent, query string, topK int, minScore float64, provider EmbeddingProvider, readPolicy string) ([]SearchResult, error) {
	if topK <= 0 {
		topK = 5
	}
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}

	policy, err := config.NormalizeMemoryReadPolicy(readPolicy)
	if err != nil {
		policy = config.MemoryReadPolicySelf
	}

	agents := []string{currentAgent}
	if policy == config.MemoryReadPolicyShared {
		all, err := ListMemoryAgents()
		if err == nil {
			agents = all
		}
	}

	if len(agents) == 0 {
		return nil, nil
	}

	results := make([]SearchResult, 0, topK*2)
	for _, agent := range agents {
		_ = expireSilently(agent)
		single, err := searchSingleAgent(ctx, agent, query, topK*2, minScore, provider)
		if err != nil {
			continue
		}
		results = append(results, single...)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	if len(results) > topK {
		results = results[:topK]
	}
	for _, r := range results {
		_ = LogRetrieval(r.Agent, query, r)
	}
	return results, nil
}

func searchSingleAgent(ctx context.Context, agent, query string, topK int, minScore float64, provider EmbeddingProvider) ([]SearchResult, error) {
	if provider == nil {
		return lexicalOnly(agent, query, topK), nil
	}

	queryEmb, embErr := provider.Embed(ctx, query)
	if embErr != nil || len(queryEmb) == 0 {
		return lexicalOnly(agent, query, topK), nil
	}

	chunks, err := ListChunks(agent)
	if err != nil {
		return nil, err
	}
	if len(chunks) == 0 {
		return nil, nil
	}

	results := make([]SearchResult, 0, len(chunks))
	for _, c := range chunks {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		lex := lexicalSimilarity(query, c.Text)
		semantic := 0.0
		if len(c.Embedding) > 0 {
			if sim, err := cosine(queryEmb, c.Embedding); err == nil {
				semantic = normalizeCosine(sim)
			}
		}
		score := semanticWeight*semantic + lexicalWeight*lex
		if score < minScore {
			continue
		}
		results = append(results, SearchResult{
			ChunkID:       c.ID,
			DocumentID:    c.DocumentID,
			Text:          c.Text,
			Score:         score,
			SemanticScore: semantic,
			LexicalScore:  lex,
			Agent:         agent,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	if len(results) > topK {
		results = results[:topK]
	}
	return results, nil
}

func lexicalOnly(agent, query string, topK int) []SearchResult {
	matches, err := LexicalSearch(agent, query, topK)
	if err != nil {
		return nil
	}
	for i := range matches {
		matches[i].LexicalScore = lexicalSimilarity(query, matches[i].Text)
		matches[i].SemanticScore = 0
		matches[i].Score = matches[i].LexicalScore
	}
	return matches
}

func lexicalSimilarity(query, text string) float64 {
	q := normalizeTokens(query)
	t := normalizeTokens(text)
	if len(q) == 0 || len(t) == 0 {
		return 0
	}
	tSet := map[string]struct{}{}
	for _, token := range t {
		tSet[token] = struct{}{}
	}
	matches := 0
	for _, token := range q {
		if _, ok := tSet[token]; ok {
			matches++
		}
	}
	return float64(matches) / float64(len(q))
}

func normalizeTokens(text string) []string {
	clean := strings.ToLower(strings.TrimSpace(text))
	clean = strings.NewReplacer(",", " ", ".", " ", ";", " ", ":", " ", "\n", " ", "\t", " ", "(", " ", ")", " ", "[", " ", "]", " ", "{", " ", "}", " ").Replace(clean)
	parts := strings.Fields(clean)
	seen := map[string]struct{}{}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if len(p) <= 1 {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

func normalizeCosine(v float64) float64 {
	x := (v + 1) / 2
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

func expireSilently(agent string) error {
	_, err := ExpireDocuments(agent)
	return err
}

func cosine(a, b []float64) (float64, error) {
	if len(a) == 0 || len(b) == 0 {
		return 0, errors.New("vetor vazio")
	}
	if len(a) != len(b) {
		return 0, errors.New("dimensões incompatíveis")
	}
	var dot float64
	var na float64
	var nb float64
	for i := range a {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	if na == 0 || nb == 0 {
		return 0, errors.New("norma zero")
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb)), nil
}

func BuildRAGContext(results []SearchResult, maxChars int) string {
	if maxChars <= 0 {
		maxChars = 3000
	}
	if len(results) == 0 {
		return ""
	}
	var b strings.Builder
	for i, r := range results {
		piece := strings.TrimSpace(r.Text)
		header := fmt.Sprintf("\n### Memória %d (score=%.3f", i+1, r.Score)
		if r.Agent != "" {
			header += fmt.Sprintf(" | agent=%s", r.Agent)
		}
		header += ")\n"
		chunk := header + piece + "\n"
		if b.Len()+len(chunk) > maxChars {
			remaining := maxChars - b.Len()
			if remaining <= 0 {
				break
			}
			chunk = chunk[:remaining]
		}
		b.WriteString(chunk)
		if b.Len() >= maxChars {
			break
		}
	}
	return strings.TrimSpace(b.String())
}
