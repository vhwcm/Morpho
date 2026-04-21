package agents

import (
	"context"
	"fmt"
	"strings"

	"github.com/vhwcm/Morpho/internal/gemini"
)

func RunSolutionAgent(ctx context.Context, ai AIClient, input DiagnosticInput, plan PlanResult, logs LogResult, metrics MetricsResult) (string, error) {
	prompt := fmt.Sprintf(
		"Com base no contexto abaixo, gere uma sugestão objetiva de mitigação em português.\nProblema: %s\nPlano: %s\nLogs: %s\nMétricas: CPU %.2f%%, Memória %.2f MB",
		input.Problem,
		plan.Strategy,
		logs.Summary,
		metrics.CPUPercent,
		metrics.MemoryUsedMB,
	)

	if ai == nil {
		return "Solução padrão: priorize endpoints com erro 500, valide timeouts externos e aplique mitigação de carga.", nil
	}

	res, err := ai.Chat(ctx, "", []gemini.ChatMessage{{Role: "user", Content: prompt}})
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(res.Message), nil
}
