package agentkit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseAndValidateEditPlan(t *testing.T) {
	raw := `{"summary":"ok","edits":[{"path":"internal/config/env.go","summary":"ajuste","content":"package config"}]}`
	plan, err := ParseEditPlan(raw)
	if err != nil {
		t.Fatalf("erro ao parsear plano: %v", err)
	}
	if plan.Summary != "ok" || len(plan.Edits) != 1 {
		t.Fatalf("plano inesperado: %+v", plan)
	}

	if err := ValidateEditPlan(plan, []string{"internal/config"}, 5); err != nil {
		t.Fatalf("plano deveria ser válido: %v", err)
	}

	if err := ValidateEditPlan(plan, []string{"cmd"}, 5); err == nil {
		t.Fatalf("esperava erro para path fora da allowlist")
	}
}

func TestParseEditPlanWithWrappedText(t *testing.T) {
	raw := "Segue plano:\n```json\n{\"summary\":\"ok\",\"edits\":[]}\n```"
	plan, err := ParseEditPlan(raw)
	if err != nil {
		t.Fatalf("deveria extrair JSON mesmo com texto extra: %v", err)
	}
	if plan.Summary != "ok" {
		t.Fatalf("summary inesperado: %s", plan.Summary)
	}
}

func TestNormalizeRelativePathAndAllowlist(t *testing.T) {
	n, err := NormalizeRelativePath("internal/agentkit/storage.go")
	if err != nil || n != "internal/agentkit/storage.go" {
		t.Fatalf("normalização inesperada: %s err=%v", n, err)
	}

	if _, err := NormalizeRelativePath("../passwd"); err == nil {
		t.Fatalf("esperava erro para path fora do workspace")
	}
	if _, err := NormalizeRelativePath("/etc/passwd"); err == nil {
		t.Fatalf("esperava erro para path absoluto")
	}

	if !IsPathAllowed("internal/agentkit/storage.go", []string{"internal"}) {
		t.Fatalf("path deveria ser permitido")
	}
	if IsPathAllowed("README.md", []string{"internal"}) {
		t.Fatalf("path não deveria ser permitido")
	}
}

func TestApplyFileEditCreatesAndBackups(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("erro ao obter cwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("erro ao trocar cwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	res, err := ApplyFileEdit(tmp, FileEdit{Path: "internal/new_file.go", Content: "package x\n"})
	if err != nil {
		t.Fatalf("erro ao criar arquivo: %v", err)
	}
	if !res.Created || !res.Changed {
		t.Fatalf("resultado de criação inesperado: %+v", res)
	}

	target := filepath.Join(tmp, "internal", "new_file.go")
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("erro ao ler arquivo criado: %v", err)
	}
	if string(content) != "package x\n" {
		t.Fatalf("conteúdo inesperado: %q", string(content))
	}

	res2, err := ApplyFileEdit(tmp, FileEdit{Path: "internal/new_file.go", Content: "package y\n"})
	if err != nil {
		t.Fatalf("erro ao atualizar arquivo: %v", err)
	}
	if res2.Created || !res2.Changed {
		t.Fatalf("resultado de update inesperado: %+v", res2)
	}
	if res2.BackupPath == "" {
		t.Fatalf("update deveria gerar backup")
	}
	if !strings.HasPrefix(res2.BackupPath, ".morpho/backups/") {
		t.Fatalf("backup path inesperado: %s", res2.BackupPath)
	}

	res3, err := ApplyFileEdit(tmp, FileEdit{Path: "internal/new_file.go", Content: "package y\n"})
	if err != nil {
		t.Fatalf("erro ao aplicar conteúdo idêntico: %v", err)
	}
	if res3.Changed {
		t.Fatalf("conteúdo idêntico não deveria marcar changed")
	}
}
