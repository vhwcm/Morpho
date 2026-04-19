package config

import "os"

type Env struct {
	GeminiAPIKey string
	GeminiModel  string
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

	return Env{
		GeminiAPIKey: apiKey,
		GeminiModel:  model,
	}
}
