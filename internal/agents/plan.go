package agents

import (
	"context"
	"fmt"
	"strings"
)

func RunPlanAgent(ctx context.Context, ai AIClient, problem string) (PlanResult, error) {
	prompt := fmt.Sprintf("Gere um plano curto de investigação SRE para o problema: %s", problem)
	if ai == nil {
		return PlanResult{Strategy: "Plano padrão: coletar evidências de logs e métricas, depois priorizar hipóteses por impacto."}, nil
	}

	out, err := ai.Generate(ctx, prompt)
	if err != nil {
		return PlanResult{}, err
	}

	return PlanResult{Strategy: strings.TrimSpace(out)}, nil
}
