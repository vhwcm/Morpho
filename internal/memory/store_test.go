package memory

import (
	"os"
	"testing"
	"time"
)

func withTempWD(t *testing.T) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatalf("cwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
}

func TestStoreUpsertAndStats(t *testing.T) {
	withTempWD(t)
	agent := "backend-go"
	docID, err := UpsertDocument(agent, "output", "run-1", "conteúdo inicial", 0.8, nil)
	if err != nil {
		t.Fatalf("upsert doc: %v", err)
	}
	if docID <= 0 {
		t.Fatalf("doc id inválido: %d", docID)
	}

	err = InsertChunks(agent, docID, []ChunkInput{{Text: "primeiro chunk", Tokens: 2, Hash: HashChunk("primeiro chunk")}})
	if err != nil {
		t.Fatalf("insert chunk: %v", err)
	}

	stats, err := GetStats(agent)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.Documents != 1 || stats.Chunks != 1 {
		t.Fatalf("stats inesperado: %+v", stats)
	}
}

func TestExpireDocumentsByTTL(t *testing.T) {
	withTempWD(t)
	agent := "frontend-react"
	expiredAt := time.Now().UTC().Add(-1 * time.Hour)
	docID, err := UpsertDocument(agent, "output", "expired-run", "conteúdo expirado", 0.4, &expiredAt)
	if err != nil {
		t.Fatalf("upsert doc ttl: %v", err)
	}
	if err := InsertChunks(agent, docID, []ChunkInput{{Text: "chunk expirado", Tokens: 2, Hash: HashChunk("chunk expirado")}}); err != nil {
		t.Fatalf("insert chunk ttl: %v", err)
	}

	affected, err := ExpireDocuments(agent)
	if err != nil {
		t.Fatalf("expire docs: %v", err)
	}
	if affected == 0 {
		t.Fatalf("esperava remover ao menos 1 documento expirado")
	}

	stats, err := GetStats(agent)
	if err != nil {
		t.Fatalf("stats pós-expiração: %v", err)
	}
	if stats.Documents != 0 || stats.Chunks != 0 {
		t.Fatalf("esperava memória vazia após expiração, got=%+v", stats)
	}
}
