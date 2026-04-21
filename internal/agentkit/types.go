package agentkit

import (
	"context"
	"time"

	"github.com/vhwcm/Morpho/internal/gemini"
)

type AIClient interface {
	Generate(ctx context.Context, prompt string) (string, error)
	Chat(ctx context.Context, systemPrompt string, history []gemini.ChatMessage, tools ...gemini.Tool) (gemini.ChatResult, error)
	Embed(ctx context.Context, text string) ([]float64, error)
}

type Spec struct {
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	SystemPrompt string    `json:"system_prompt"`
	Model        string    `json:"model"`
	Tags         []string  `json:"tags"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Tools        []gemini.Tool `json:"tools,omitempty"`
}

type ChatMessage = gemini.ChatMessage
