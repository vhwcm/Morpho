package config

import (
	"fmt"
	"strings"
)

const (
	MemoryReadPolicySelf   = "self"
	MemoryReadPolicyShared = "shared"
)

func NormalizeMemoryReadPolicy(policy string) (string, error) {
	p := strings.ToLower(strings.TrimSpace(policy))
	switch p {
	case "", MemoryReadPolicySelf:
		return MemoryReadPolicySelf, nil
	case MemoryReadPolicyShared:
		return MemoryReadPolicyShared, nil
	default:
		return "", fmt.Errorf("read policy inválida: %s (use self|shared)", policy)
	}
}

func SaveMemoryReadPolicy(policy string) error {
	normalized, err := NormalizeMemoryReadPolicy(policy)
	if err != nil {
		return err
	}
	cfg, err := LoadFileConfig()
	if err != nil {
		return err
	}
	cfg.Memory.ReadPolicy = normalized
	cfg.Memory.CrossAgentRead = normalized == MemoryReadPolicyShared
	return SaveFileConfig(cfg)
}

func SaveMemoryTTLHours(hours int) error {
	if hours <= 0 {
		return fmt.Errorf("ttl_hours deve ser maior que zero")
	}
	cfg, err := LoadFileConfig()
	if err != nil {
		return err
	}
	cfg.Memory.TTLHours = hours
	return SaveFileConfig(cfg)
}
