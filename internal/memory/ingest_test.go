package memory

import (
	"context"
	"os"
	"testing"
)

type ingestEmbedder struct{}

func (i ingestEmbedder) Embed(_ context.Context, text string) ([]float64, error) {
	return []float64{float64(len(text)), 1, 1}, nil
}

func TestIngestRun(t *testing.T) {
	old, _ := os.Getwd()
	tmp := t.TempDir()
	_ = os.Chdir(tmp)
	t.Cleanup(func() { _ = os.Chdir(old) })

	err := IngestRun(context.Background(), ingestEmbedder{}, IngestInput{
		Agent:    "devops-ci",
		Task:     "investigar timeout",
		Output:   "causa raiz encontrada e mitigação aplicada",
		Source:   "output-1.md",
		TTLHours: 1,
	})
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}

	stats, err := GetStats("devops-ci")
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.Documents == 0 || stats.Chunks == 0 {
		t.Fatalf("esperava documentos/chunks após ingestão: %+v", stats)
	}
}
