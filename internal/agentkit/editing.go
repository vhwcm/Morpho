package agentkit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FileEdit struct {
	Path    string `json:"path"`
	Summary string `json:"summary"`
	Content string `json:"content"`
}

type EditPlan struct {
	Summary string     `json:"summary"`
	Edits   []FileEdit `json:"edits"`
}

type ApplyResult struct {
	Path       string
	BackupPath string
	Created    bool
	Changed    bool
}

func BuildEditTask(userTask string, allowedPaths []string, maxEdits int) string {
	if maxEdits <= 0 {
		maxEdits = 10
	}

	allowed := "qualquer caminho dentro do workspace"
	if len(allowedPaths) > 0 {
		allowed = strings.Join(allowedPaths, ", ")
	}

	return fmt.Sprintf(`Você vai propor alterações de arquivos para a tarefa abaixo.

Tarefa:
%s

Regras obrigatórias:
1) Retorne SOMENTE JSON válido (sem markdown, sem explicações fora do JSON).
2) Formato do JSON:
{
  "summary": "resumo curto",
  "edits": [
    {
      "path": "caminho/relativo/no/workspace.ext",
      "summary": "o que mudou",
      "content": "conteúdo COMPLETO final do arquivo"
    }
  ]
}
3) Gere no máximo %d edições.
4) Só use caminhos relativos.
5) Restrinja alterações para os caminhos permitidos: %s.
6) Se não precisar editar arquivos, retorne edits como array vazio.
`, userTask, maxEdits, allowed)
}

func ParseEditPlan(raw string) (EditPlan, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return EditPlan{}, fmt.Errorf("resposta vazia ao gerar plano de edição")
	}

	var plan EditPlan
	if err := json.Unmarshal([]byte(trimmed), &plan); err == nil {
		return plan, nil
	}

	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start == -1 || end == -1 || end <= start {
		return EditPlan{}, fmt.Errorf("não foi possível extrair JSON do plano de edição")
	}

	candidate := trimmed[start : end+1]
	if err := json.Unmarshal([]byte(candidate), &plan); err != nil {
		return EditPlan{}, fmt.Errorf("json de plano inválido: %w", err)
	}

	return plan, nil
}

func NormalizeRelativePath(path string) (string, error) {
	p := strings.TrimSpace(path)
	if p == "" {
		return "", fmt.Errorf("path vazio")
	}

	p = filepath.ToSlash(p)
	if strings.HasPrefix(p, "/") {
		return "", fmt.Errorf("path absoluto não permitido: %s", path)
	}

	clean := filepath.ToSlash(filepath.Clean(p))
	if clean == "." || clean == "" || strings.HasPrefix(clean, "../") || clean == ".." {
		return "", fmt.Errorf("path fora do workspace não permitido: %s", path)
	}

	return clean, nil
}

func IsPathAllowed(path string, allowedPrefixes []string) bool {
	if len(allowedPrefixes) == 0 {
		return true
	}

	target := filepath.ToSlash(path)
	for _, prefix := range allowedPrefixes {
		p := strings.TrimSpace(filepath.ToSlash(prefix))
		if p == "" {
			continue
		}
		if p == "." {
			return true
		}
		if target == p || strings.HasPrefix(target, p+"/") {
			return true
		}
	}
	return false
}

func ValidateEditPlan(plan EditPlan, allowedPaths []string, maxEdits int) error {
	if maxEdits <= 0 {
		maxEdits = 10
	}
	if len(plan.Edits) > maxEdits {
		return fmt.Errorf("plano excede limite de edições (%d > %d)", len(plan.Edits), maxEdits)
	}

	seen := map[string]struct{}{}
	for i, e := range plan.Edits {
		normalized, err := NormalizeRelativePath(e.Path)
		if err != nil {
			return fmt.Errorf("edição %d inválida: %w", i+1, err)
		}
		if !IsPathAllowed(normalized, allowedPaths) {
			return fmt.Errorf("edição %d fora dos caminhos permitidos: %s", i+1, normalized)
		}
		if strings.TrimSpace(e.Content) == "" {
			return fmt.Errorf("edição %d sem conteúdo final", i+1)
		}
		if _, ok := seen[normalized]; ok {
			return fmt.Errorf("edição duplicada para o mesmo arquivo: %s", normalized)
		}
		seen[normalized] = struct{}{}
	}

	return nil
}

func ApplyFileEdit(workspaceRoot string, edit FileEdit) (ApplyResult, error) {
	normalized, err := NormalizeRelativePath(edit.Path)
	if err != nil {
		return ApplyResult{}, err
	}

	fullPath := filepath.Join(workspaceRoot, normalized)
	result := ApplyResult{Path: normalized}

	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return ApplyResult{}, err
	}

	newContent := []byte(edit.Content)
	if existing, err := os.ReadFile(fullPath); err == nil {
		if string(existing) == string(newContent) {
			result.Changed = false
			return result, nil
		}

		backupPath, err := backupFile(workspaceRoot, normalized, existing)
		if err != nil {
			return ApplyResult{}, err
		}
		result.BackupPath = backupPath
		result.Created = false
	} else if os.IsNotExist(err) {
		result.Created = true
	} else {
		return ApplyResult{}, err
	}

	if err := os.WriteFile(fullPath, newContent, 0o644); err != nil {
		return ApplyResult{}, err
	}

	result.Changed = true
	return result, nil
}

func backupFile(workspaceRoot, relativePath string, content []byte) (string, error) {
	ts := time.Now().UTC().Format("20060102-150405")
	backupRoot := filepath.Join(workspaceRoot, ".morpho", "backups", ts)
	backupPath := filepath.Join(backupRoot, filepath.FromSlash(relativePath)+".bak")
	if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(backupPath, content, 0o644); err != nil {
		return "", err
	}

	rel, err := filepath.Rel(workspaceRoot, backupPath)
	if err != nil {
		return filepath.ToSlash(backupPath), nil
	}
	return filepath.ToSlash(rel), nil
}
