package config

import "os"

type Env struct {
	GeminiAPIKey string
	GeminiModel  string
	AgentEditing AgentEditConfig
}

func Load() Env {
	fileCfg, _ := LoadFileConfig()

	model := os.Getenv("GEMINI_MODEL")
	if model == "" {
		model = fileCfg.GeminiModel
	}
	if model == "" {
		model = "gemini-2.0-flash"
	}

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		apiKey = fileCfg.GeminiAPIKey
	}

	mode, err := NormalizeEditMode(fileCfg.AgentEditing.Mode)
	if err != nil {
		mode = EditModeOff
	}

	return Env{
		GeminiAPIKey: apiKey,
		GeminiModel:  model,
		AgentEditing: AgentEditConfig{
			Mode:         mode,
			AllowedPaths: normalizeAllowedPaths(fileCfg.AgentEditing.AllowedPaths),
		},
	}
}
