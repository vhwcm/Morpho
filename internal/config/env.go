package config

import "os"

type Env struct {
	GeminiAPIKey string
	GeminiModel  string
	AgentEditing AgentEditConfig
	Memory       MemoryConfig
}

func Load() Env {
	fileCfg, _ := LoadFileConfig()

	model := os.Getenv("GEMINI_MODEL")
	if model == "" {
		model = fileCfg.GeminiModel
	}
	if model == "" {
		model = "gemini-2.5-flash"
	}

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		apiKey = fileCfg.GeminiAPIKey
	}

	mode, err := NormalizeEditMode(fileCfg.AgentEditing.Mode)
	if err != nil {
		mode = EditModeOff
	}

	memory := fileCfg.Memory
	if memory.TTLHours <= 0 {
		memory.TTLHours = 720
	}
	if memory.TopK <= 0 {
		memory.TopK = 6
	}
	if memory.MinScore == 0 {
		memory.MinScore = 0.25
	}
	if memory.MaxChars <= 0 {
		memory.MaxChars = 3000
	}
	if !memory.Enabled {
		if os.Getenv("MORPHO_MEMORY_ENABLED") == "1" || os.Getenv("MORPHO_MEMORY_ENABLED") == "true" {
			memory.Enabled = true
		}
	}
	if os.Getenv("MORPHO_MEMORY_ENABLED") == "0" || os.Getenv("MORPHO_MEMORY_ENABLED") == "false" {
		memory.Enabled = false
	}
	if fileCfg.Memory == (MemoryConfig{}) && os.Getenv("MORPHO_MEMORY_ENABLED") == "" {
		memory.Enabled = true
	}

	if stringsToBool(os.Getenv("MORPHO_MEMORY_CROSS_AGENT")) {
		memory.CrossAgentRead = true
	}
	if stringsToFalseBool(os.Getenv("MORPHO_MEMORY_CROSS_AGENT")) {
		memory.CrossAgentRead = false
	}

	policy := memory.ReadPolicy
	if envPolicy := os.Getenv("MORPHO_MEMORY_READ_POLICY"); envPolicy != "" {
		policy = envPolicy
	}
	if policy == "" {
		if memory.CrossAgentRead {
			policy = MemoryReadPolicyShared
		} else {
			policy = MemoryReadPolicySelf
		}
	}
	normalizedPolicy, err := NormalizeMemoryReadPolicy(policy)
	if err != nil {
		normalizedPolicy = MemoryReadPolicySelf
	}
	memory.ReadPolicy = normalizedPolicy
	memory.CrossAgentRead = normalizedPolicy == MemoryReadPolicyShared

	return Env{
		GeminiAPIKey: apiKey,
		GeminiModel:  model,
		AgentEditing: AgentEditConfig{
			Mode:         mode,
			AllowedPaths: normalizeAllowedPaths(fileCfg.AgentEditing.AllowedPaths),
		},
		Memory: memory,
	}
}

func stringsToBool(v string) bool {
	s := v
	return s == "1" || s == "true" || s == "TRUE" || s == "True"
}

func stringsToFalseBool(v string) bool {
	s := v
	return s == "0" || s == "false" || s == "FALSE" || s == "False"
}

func GetGeminiAPIKey() string {
	return Load().GeminiAPIKey
}
