package memory

import (
	"context"
	"os"
	"testing"
)

type testEmbedder struct{}

func (t testEmbedder) Embed(_ context.Context, text string) ([]float64, error) {
	if text == "" {
		return []float64{0, 0, 0}, nil
	}
	return []float64{1, 0.5, 0.25}, nil
}

func TestRetrieverSearch(t *testing.T) {
	old, _ := os.Getwd()
	tmp := t.TempDir()
	_ = os.Chdir(tmp)
	t.Cleanup(func() { _ = os.Chdir(old) })

	docID, err := UpsertDocument("qa-tester", "output", "run-2", "erro 500 no endpoint", 0.7, nil)
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	err = InsertChunks("qa-tester", docID, []ChunkInput{{
		Text:      "erro 500 no endpoint de login",
		Tokens:    6,
		Embedding: []float64{1, 0.5, 0.25},
		Hash:      HashChunk("erro 500 no endpoint de login"),
	}})
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	results, err := Search(context.Background(), "qa-tester", "erro 500 login", 3, 0.1, testEmbedder{})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) == 0 {
		t.Fatalf("esperava resultados")
	}
	if results[0].Score <= 0 {
		t.Fatalf("score inválido: %f", results[0].Score)
	}
	if results[0].SemanticScore <= 0 {
		t.Fatalf("semantic score inválido: %f", results[0].SemanticScore)
	}
	if results[0].LexicalScore <= 0 {
		t.Fatalf("lexical score inválido: %f", results[0].LexicalScore)
	}
}

func TestRetrieverSharedPolicy(t *testing.T) {
	old, _ := os.Getwd()
	tmp := t.TempDir()
	_ = os.Chdir(tmp)
	t.Cleanup(func() { _ = os.Chdir(old) })

	docA, err := UpsertDocument("backend-go", "output", "run-a", "timeouts em autenticação", 0.7, nil)
	if err != nil {
		t.Fatalf("upsert A: %v", err)
	}
	if err := InsertChunks("backend-go", docA, []ChunkInput{{Text: "timeout auth endpoint", Tokens: 3, Embedding: []float64{1, 0.5, 0.25}, Hash: HashChunk("timeout auth endpoint")}}); err != nil {
		t.Fatalf("insert A: %v", err)
	}

	docB, err := UpsertDocument("qa-tester", "output", "run-b", "erro 500 em login", 0.7, nil)
	if err != nil {
		t.Fatalf("upsert B: %v", err)
	}
	if err := InsertChunks("qa-tester", docB, []ChunkInput{{Text: "erro 500 login", Tokens: 3, Embedding: []float64{1, 0.5, 0.25}, Hash: HashChunk("erro 500 login")}}); err != nil {
		t.Fatalf("insert B: %v", err)
	}

	results, err := SearchWithPolicy(context.Background(), "backend-go", "erro 500 login", 5, 0.1, testEmbedder{}, "shared")
	if err != nil {
		t.Fatalf("search shared: %v", err)
	}
	if len(results) == 0 {
		t.Fatalf("esperava resultados em policy shared")
	}
	foundCross := false
	for _, r := range results {
		if r.Agent == "qa-tester" {
			foundCross = true
			break
		}
	}
	if !foundCross {
		t.Fatalf("policy shared deveria incluir memória de outro agente")
	}
}
