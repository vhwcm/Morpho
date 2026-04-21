package agentkit

import (
	"context"
	"strings"

	"github.com/vhwcm/Morpho/internal/gemini"
	"github.com/vhwcm/Morpho/internal/logger"
)

func Run(ctx context.Context, ai AIClient, spec Spec, task string) (string, error) {
	logger.Info("Executando agente", map[string]interface{}{
		"agent": spec.Name,
		"task":  task,
	})

	history := []gemini.ChatMessage{{Role: "user", Content: task}}
	res, err := ai.Chat(ctx, spec.SystemPrompt, history, spec.Tools...)
	if err != nil {
		logger.Error("Erro na execução do agente", err, map[string]interface{}{
			"agent": spec.Name,
		})
		return "", err
	}

	logger.Info("Agente finalizado com sucesso", map[string]interface{}{
		"agent": spec.Name,
	})
	return strings.TrimSpace(res.Message), nil
}

func RunWithResult(ctx context.Context, ai AIClient, spec Spec, history []gemini.ChatMessage) (gemini.ChatResult, error) {
	return ai.Chat(ctx, spec.SystemPrompt, history, spec.Tools...)
}
