package agentkit

import (
	"os"
	"strings"
)

func BuiltinPresets(defaultModel string) []Spec {
	model := strings.TrimSpace(defaultModel)
	if model == "" {
		model = "gemini-2.0-flash"
	}

	return []Spec{
		{
			Name:         "backend-go",
			Description:  "Especialista em backend Go, arquitetura e performance.",
			SystemPrompt: "Você é um engenheiro backend Go sênior. Proponha soluções idiomáticas, simples e testáveis.",
			Model:        model,
			Tags:         []string{"go", "backend", "api"},
		},
		{
			Name:         "frontend-react",
			Description:  "Especialista em React, UX e componentização.",
			SystemPrompt: "Você é um engenheiro frontend React. Foque em clareza, acessibilidade e componentização.",
			Model:        model,
			Tags:         []string{"react", "frontend", "ui"},
		},
		{
			Name:         "code-reviewer",
			Description:  "Revisor técnico para qualidade e segurança.",
			SystemPrompt: "Você é um revisor de código rigoroso. Identifique bugs, riscos e melhorias objetivas.",
			Model:        model,
			Tags:         []string{"review", "quality", "security"},
		},
		{
			Name:         "qa-tester",
			Description:  "Especialista em estratégia de testes e cobertura.",
			SystemPrompt: "Você é um QA engineer. Sugira cenários de teste práticos e priorize riscos.",
			Model:        model,
			Tags:         []string{"qa", "test", "automation"},
		},
		{
			Name:         "devops-ci",
			Description:  "Especialista em CI/CD e automação de entrega.",
			SystemPrompt: "Você é um engenheiro DevOps. Sugira pipelines simples, seguras e observáveis.",
			Model:        model,
			Tags:         []string{"devops", "ci", "cd"},
		},
	}
}

func SeedPresets(force bool, defaultModel string) (int, error) {
	presets := BuiltinPresets(defaultModel)

	created := 0
	for _, p := range presets {
		_, err := os.Stat(specPath(p.Name))
		if err == nil && !force {
			continue
		}
		if err := SaveSpec(p); err != nil {
			return created, err
		}
		created++
	}

	return created, nil
}
