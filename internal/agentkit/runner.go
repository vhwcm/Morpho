package agentkit

import (
	"context"
	"fmt"
	"strings"
)

func Run(ctx context.Context, ai AIClient, spec Spec, task string) (string, error) {
	prompt := fmt.Sprintf(
		"Instruções do agente:\n%s\n\nTarefa do usuário:\n%s\n\nResponda em português com foco prático e objetivo.",
		spec.SystemPrompt,
		task,
	)

	out, err := ai.Generate(ctx, prompt)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(out), nil
}
