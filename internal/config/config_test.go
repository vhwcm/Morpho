package config

import (
	"os"
	"path/filepath"
	"testing"
)

func setupConfigEnv(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
	return dir
}

func TestSaveAndLoadFileConfig(t *testing.T) {
	base := setupConfigEnv(t)

	in := FileConfig{
		GeminiAPIKey: "key-123",
		GeminiModel:  "gemini-2.5-flash",
	}
	if err := SaveFileConfig(in); err != nil {
		t.Fatalf("erro ao salvar config: %v", err)
	}

	out, err := LoadFileConfig()
	if err != nil {
		t.Fatalf("erro ao carregar config: %v", err)
	}
	if out.GeminiAPIKey != in.GeminiAPIKey || out.GeminiModel != in.GeminiModel {
		t.Fatalf("config carregada difere da salva: got=%+v want=%+v", out, in)
	}

	path, err := ConfigFilePath()
	if err != nil {
		t.Fatalf("erro ao obter caminho do config: %v", err)
	}
	expected := filepath.Join(base, "morpho", "config.json")
	if path != expected {
		t.Fatalf("caminho inesperado: got=%s want=%s", path, expected)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("arquivo de config deveria existir: %v", err)
	}
}

func TestSaveGeminiHelpers(t *testing.T) {
	setupConfigEnv(t)

	if err := SaveGeminiAPIKey("   "); err == nil {
		t.Fatalf("esperava erro para api key vazia")
	}
	if err := SaveGeminiModel("   "); err == nil {
		t.Fatalf("esperava erro para modelo vazio")
	}

	if err := SaveGeminiAPIKey("  api-key  "); err != nil {
		t.Fatalf("erro ao salvar api key: %v", err)
	}
	if err := SaveGeminiModel("  gemini-2.0-flash  "); err != nil {
		t.Fatalf("erro ao salvar modelo: %v", err)
	}

	cfg, err := LoadFileConfig()
	if err != nil {
		t.Fatalf("erro ao carregar config: %v", err)
	}
	if cfg.GeminiAPIKey != "api-key" {
		t.Fatalf("api key deveria estar trimada: %q", cfg.GeminiAPIKey)
	}
	if cfg.GeminiModel != "gemini-2.0-flash" {
		t.Fatalf("modelo deveria estar trimado: %q", cfg.GeminiModel)
	}
}

func TestLoadPrecedenceAndDefaults(t *testing.T) {
	setupConfigEnv(t)

	if err := SaveFileConfig(FileConfig{GeminiAPIKey: "file-key", GeminiModel: "file-model"}); err != nil {
		t.Fatalf("erro ao preparar config em arquivo: %v", err)
	}

	env := Load()
	if env.GeminiAPIKey != "file-key" || env.GeminiModel != "file-model" {
		t.Fatalf("deveria usar valores do arquivo quando env vars estão vazias: %+v", env)
	}

	t.Setenv("GEMINI_API_KEY", "env-key")
	t.Setenv("GEMINI_MODEL", "env-model")
	env = Load()
	if env.GeminiAPIKey != "env-key" || env.GeminiModel != "env-model" {
		t.Fatalf("env vars deveriam ter precedência: %+v", env)
	}

	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GEMINI_MODEL", "")
	if err := SaveFileConfig(FileConfig{}); err != nil {
		t.Fatalf("erro ao limpar config em arquivo: %v", err)
	}
	env = Load()
	if env.GeminiModel != "gemini-2.0-flash" {
		t.Fatalf("deveria usar modelo padrão quando nada está configurado: %s", env.GeminiModel)
	}
	if env.GeminiAPIKey != "" {
		t.Fatalf("api key deveria permanecer vazia quando não configurada")
	}
	if env.AgentEditing.Mode != EditModeOff {
		t.Fatalf("modo padrão de edição deveria ser off, got=%s", env.AgentEditing.Mode)
	}
}

func TestAgentEditConfigHelpers(t *testing.T) {
	setupConfigEnv(t)

	if err := SaveAgentEditMode("review"); err != nil {
		t.Fatalf("erro ao salvar mode review: %v", err)
	}
	if err := AddAgentEditAllowedPath("internal"); err != nil {
		t.Fatalf("erro ao adicionar path internal: %v", err)
	}
	if err := AddAgentEditAllowedPath("cmd"); err != nil {
		t.Fatalf("erro ao adicionar path cmd: %v", err)
	}

	loaded := Load()
	if loaded.AgentEditing.Mode != EditModeReview {
		t.Fatalf("mode esperado review, got=%s", loaded.AgentEditing.Mode)
	}
	if len(loaded.AgentEditing.AllowedPaths) != 2 {
		t.Fatalf("esperava 2 paths permitidos, got=%d", len(loaded.AgentEditing.AllowedPaths))
	}

	if err := SaveAgentEditAllowedPaths([]string{"internal/agentkit", "internal/agentkit", ""}); err != nil {
		t.Fatalf("erro ao substituir allowlist: %v", err)
	}
	loaded = Load()
	if len(loaded.AgentEditing.AllowedPaths) != 1 || loaded.AgentEditing.AllowedPaths[0] != "internal/agentkit" {
		t.Fatalf("allowlist deduplicada inesperada: %+v", loaded.AgentEditing.AllowedPaths)
	}

	if err := ClearAgentEditAllowedPaths(); err != nil {
		t.Fatalf("erro ao limpar allowlist: %v", err)
	}
	loaded = Load()
	if len(loaded.AgentEditing.AllowedPaths) != 0 {
		t.Fatalf("allowlist deveria estar vazia")
	}

	if err := SaveAgentEditMode("invalid"); err == nil {
		t.Fatalf("esperava erro para mode inválido")
	}
}
