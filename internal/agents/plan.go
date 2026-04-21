package agents

import (
	"context"
	"fmt"
	"strings"

	"github.com/vhwcm/Morpho/internal/gemini"
)

func RunPlanAgent(ctx context.Context, ai AIClient, problem string) (PlanResult, error) {
	prompt := fmt.Sprintf("Gere um plano curto de investigação SRE para o problema: %s", problem)
	if ai == nil {
		return PlanResult{Strategy: "Plano padrão: coletar evidências de logs e métricas, depois priorizar hipóteses por impacto."}, nil
	}

	res, err := ai.Chat(ctx, "", []gemini.ChatMessage{{Role: "user", Content: prompt}})
	if err != nil {
		return PlanResult{}, err
	}

	return PlanResult{Strategy: strings.TrimSpace(res.Message)}, nil
}
