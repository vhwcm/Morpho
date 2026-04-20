package config

import (
	"fmt"
	"strings"
)

const (
	EditModeOff    = "off"
	EditModeReview = "review"
	EditModeAuto   = "auto"
)

type AgentEditConfig struct {
	Mode         string   `json:"mode"`
	AllowedPaths []string `json:"allowed_paths"`
}

func NormalizeEditMode(mode string) (string, error) {
	m := strings.ToLower(strings.TrimSpace(mode))
	switch m {
	case "", EditModeOff:
		return EditModeOff, nil
	case EditModeReview:
		return EditModeReview, nil
	case EditModeAuto:
		return EditModeAuto, nil
	default:
		return "", fmt.Errorf("modo de edição inválido: %s (use off|review|auto)", mode)
	}
}

func normalizeAllowedPaths(paths []string) []string {
	result := make([]string, 0, len(paths))
	seen := map[string]struct{}{}
	for _, p := range paths {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func SaveAgentEditMode(mode string) error {
	normalized, err := NormalizeEditMode(mode)
	if err != nil {
		return err
	}

	cfg, err := LoadFileConfig()
	if err != nil {
		return err
	}

	cfg.AgentEditing.Mode = normalized
	cfg.AgentEditing.AllowedPaths = normalizeAllowedPaths(cfg.AgentEditing.AllowedPaths)
	return SaveFileConfig(cfg)
}

func SaveAgentEditAllowedPaths(paths []string) error {
	cfg, err := LoadFileConfig()
	if err != nil {
		return err
	}

	cfg.AgentEditing.AllowedPaths = normalizeAllowedPaths(paths)
	mode, err := NormalizeEditMode(cfg.AgentEditing.Mode)
	if err != nil {
		return err
	}
	cfg.AgentEditing.Mode = mode

	return SaveFileConfig(cfg)
}

func AddAgentEditAllowedPath(path string) error {
	cfg, err := LoadFileConfig()
	if err != nil {
		return err
	}

	paths := append(cfg.AgentEditing.AllowedPaths, path)
	cfg.AgentEditing.AllowedPaths = normalizeAllowedPaths(paths)
	mode, err := NormalizeEditMode(cfg.AgentEditing.Mode)
	if err != nil {
		return err
	}
	cfg.AgentEditing.Mode = mode
	return SaveFileConfig(cfg)
}

func ClearAgentEditAllowedPaths() error {
	cfg, err := LoadFileConfig()
	if err != nil {
		return err
	}

	cfg.AgentEditing.AllowedPaths = nil
	mode, err := NormalizeEditMode(cfg.AgentEditing.Mode)
	if err != nil {
		return err
	}
	cfg.AgentEditing.Mode = mode
	return SaveFileConfig(cfg)
}
