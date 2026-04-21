package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vhwcm/Morpho/internal/logger"
)

type FileConfig struct {
	GeminiAPIKey string          `json:"gemini_api_key"`
	GeminiModel  string          `json:"gemini_model"`
	AgentEditing AgentEditConfig `json:"agent_editing"`
	Memory       MemoryConfig    `json:"memory"`
}

type MemoryConfig struct {
	Enabled        bool    `json:"enabled"`
	TopK           int     `json:"top_k"`
	MinScore       float64 `json:"min_score"`
	MaxChars       int     `json:"max_chars"`
	CrossAgentRead bool    `json:"cross_agent_read"`
	ReadPolicy     string  `json:"read_policy"`
	TTLHours       int     `json:"ttl_hours"`
}

func configDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "morpho"), nil
}

func configFilePath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func ensureConfigDir() error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0o700)
}

func LoadFileConfig() (FileConfig, error) {
	path, err := configFilePath()
	if err != nil {
		return FileConfig{}, err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return FileConfig{}, nil
		}
		logger.Error("Erro ao ler arquivo de configuração", err, map[string]interface{}{"path": path})
		return FileConfig{}, err
	}

	var cfg FileConfig
	if err := json.Unmarshal(content, &cfg); err != nil {
		logger.Error("Erro ao desserializar configuração", err)
		return FileConfig{}, err
	}
	return cfg, nil
}

func SaveFileConfig(cfg FileConfig) error {
	if err := ensureConfigDir(); err != nil {
		logger.Error("Erro ao criar diretório de configuração", err)
		return err
	}

	path, err := configFilePath()
	if err != nil {
		return err
	}

	payload, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		logger.Error("Erro ao serializar configuração", err)
		return err
	}

	err = os.WriteFile(path, payload, 0o600)
	if err != nil {
		logger.Error("Erro ao escrever arquivo de configuração", err, map[string]interface{}{"path": path})
	}
	return err
}

func SaveGeminiAPIKey(apiKey string) error {
	trimmed := strings.TrimSpace(apiKey)
	if trimmed == "" {
		return fmt.Errorf("api key vazia")
	}

	cfg, err := LoadFileConfig()
	if err != nil {
		return err
	}

	cfg.GeminiAPIKey = trimmed
	return SaveFileConfig(cfg)
}

func SaveGeminiModel(model string) error {
	trimmed := strings.TrimSpace(model)
	if trimmed == "" {
		return fmt.Errorf("modelo vazio")
	}

	cfg, err := LoadFileConfig()
	if err != nil {
		return err
	}

	cfg.GeminiModel = trimmed
	return SaveFileConfig(cfg)
}

func ConfigFilePath() (string, error) {
	return configFilePath()
}
