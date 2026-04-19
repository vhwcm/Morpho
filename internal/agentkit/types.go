package agentkit

import (
	"context"
	"time"
)

type AIClient interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

type Spec struct {
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	SystemPrompt string    `json:"system_prompt"`
	Model        string    `json:"model"`
	Tags         []string  `json:"tags"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
