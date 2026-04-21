package memory

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const schema = `
CREATE TABLE IF NOT EXISTS documents (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  agent TEXT NOT NULL,
  source_type TEXT NOT NULL,
  source_ref TEXT NOT NULL,
  content TEXT NOT NULL,
  importance REAL NOT NULL DEFAULT 0.5,
  ttl_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS chunks (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  document_id INTEGER NOT NULL,
  chunk_text TEXT NOT NULL,
  tokens INTEGER NOT NULL,
  embedding_json TEXT,
  hash TEXT NOT NULL,
  created_at TEXT NOT NULL,
  UNIQUE(document_id, hash),
  FOREIGN KEY(document_id) REFERENCES documents(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS retrieval_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  query TEXT NOT NULL,
  chunk_id INTEGER NOT NULL,
  score REAL NOT NULL,
  used_in_run INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_documents_agent ON documents(agent);
CREATE INDEX IF NOT EXISTS idx_chunks_doc ON chunks(document_id);
`

func DBPath(agent string) (string, error) {
	agent = strings.TrimSpace(agent)
	if agent == "" {
		return "", fmt.Errorf("agente é obrigatório")
	}
	return filepath.Join(".morpho", "memory", agent, "knowledge.db"), nil
}

func EnsureAgentDB(agent string) (*sql.DB, error) {
	path, err := DBPath(agent)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func UpsertDocument(agent, sourceType, sourceRef, content string, importance float64, ttl *time.Time) (int64, error) {
	db, err := EnsureAgentDB(agent)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	ttlValue := ""
	if ttl != nil {
		ttlValue = ttl.UTC().Format(time.RFC3339)
	}

	var id int64
	err = db.QueryRow(`SELECT id FROM documents WHERE agent = ? AND source_type = ? AND source_ref = ? ORDER BY id DESC LIMIT 1`, agent, sourceType, sourceRef).Scan(&id)
	if err == nil {
		_, err = db.Exec(`UPDATE documents SET content = ?, importance = ?, ttl_at = ?, updated_at = ? WHERE id = ?`, content, importance, nullIfEmpty(ttlValue), now, id)
		if err != nil {
			return 0, err
		}
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}

	res, err := db.Exec(`INSERT INTO documents(agent, source_type, source_ref, content, importance, ttl_at, created_at, updated_at) VALUES(?,?,?,?,?,?,?,?)`,
		agent, sourceType, sourceRef, content, importance, nullIfEmpty(ttlValue), now, now)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func InsertChunks(agent string, documentID int64, chunks []ChunkInput) error {
	if len(chunks) == 0 {
		return nil
	}
	db, err := EnsureAgentDB(agent)
	if err != nil {
		return err
	}
	defer db.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO chunks(document_id, chunk_text, tokens, embedding_json, hash, created_at) VALUES(?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, c := range chunks {
		if strings.TrimSpace(c.Text) == "" {
			continue
		}
		emb := ""
		if len(c.Embedding) > 0 {
			b, _ := json.Marshal(c.Embedding)
			emb = string(b)
		}
		if _, err := stmt.Exec(documentID, c.Text, c.Tokens, nullIfEmpty(emb), c.Hash, now); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func ListChunks(agent string) ([]Chunk, error) {
	db, err := EnsureAgentDB(agent)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	now := time.Now().UTC().Format(time.RFC3339)

	rows, err := db.Query(`
SELECT c.id, c.document_id, c.chunk_text, c.tokens, COALESCE(c.embedding_json, ''), c.hash, c.created_at
FROM chunks c
JOIN documents d ON d.id = c.document_id
WHERE d.agent = ?
  AND (d.ttl_at IS NULL OR d.ttl_at > ?)
ORDER BY c.id DESC`, agent, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]Chunk, 0)
	for rows.Next() {
		var c Chunk
		var embJSON string
		var created string
		if err := rows.Scan(&c.ID, &c.DocumentID, &c.Text, &c.Tokens, &embJSON, &c.Hash, &created); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(embJSON), &c.Embedding)
		if ts, err := time.Parse(time.RFC3339, created); err == nil {
			c.CreatedAt = ts
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func LexicalSearch(agent, query string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 5
	}
	db, err := EnsureAgentDB(agent)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	now := time.Now().UTC().Format(time.RFC3339)

	rows, err := db.Query(`
SELECT c.id, c.document_id, c.chunk_text, d.source_ref
FROM chunks c
JOIN documents d ON d.id = c.document_id
WHERE d.agent = ?
  AND (d.ttl_at IS NULL OR d.ttl_at > ?)
  AND lower(c.chunk_text) LIKE lower(?)
ORDER BY c.id DESC
LIMIT ?`, agent, now, "%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]SearchResult, 0)
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.ChunkID, &r.DocumentID, &r.Text, &r.SourceRef); err != nil {
			return nil, err
		}
		r.Score = 0.5
		r.Agent = agent
		out = append(out, r)
	}
	return out, rows.Err()
}

func LogRetrieval(agent, query string, result SearchResult) error {
	db, err := EnsureAgentDB(agent)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(`INSERT INTO retrieval_log(query, chunk_id, score, used_in_run, created_at) VALUES(?,?,?,?,?)`,
		query, result.ChunkID, result.Score, 1, time.Now().UTC().Format(time.RFC3339))
	return err
}

func GetStats(agent string) (Stats, error) {
	db, err := EnsureAgentDB(agent)
	if err != nil {
		return Stats{}, err
	}
	defer db.Close()
	now := time.Now().UTC().Format(time.RFC3339)

	var s Stats
	if err := db.QueryRow(`SELECT COUNT(*) FROM documents WHERE agent = ? AND (ttl_at IS NULL OR ttl_at > ?)`, agent, now).Scan(&s.Documents); err != nil {
		return Stats{}, err
	}
	if err := db.QueryRow(`
SELECT COUNT(*) FROM chunks c
JOIN documents d ON d.id = c.document_id
WHERE d.agent = ?
  AND (d.ttl_at IS NULL OR d.ttl_at > ?)`, agent, now).Scan(&s.Chunks); err != nil {
		return Stats{}, err
	}
	return s, nil
}

func ExpireDocuments(agent string) (int64, error) {
	db, err := EnsureAgentDB(agent)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	res, err := db.Exec(`DELETE FROM documents WHERE agent = ? AND ttl_at IS NOT NULL AND ttl_at <= ?`, agent, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func ListMemoryAgents() ([]string, error) {
	root := filepath.Join(".morpho", "memory")
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out, nil
}

func Prune(agent string, maxDocuments int) error {
	if maxDocuments <= 0 {
		return nil
	}
	db, err := EnsureAgentDB(agent)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(`
DELETE FROM documents
WHERE id IN (
	SELECT id FROM documents
	WHERE agent = ?
	ORDER BY updated_at DESC
	LIMIT -1 OFFSET ?
)`, agent, maxDocuments)
	return err
}

func nullIfEmpty(v string) interface{} {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return v
}
