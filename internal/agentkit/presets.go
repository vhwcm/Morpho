package agentkit

import (
	"os"
	"strings"

	"github.com/vhwcm/Morpho/internal/gemini"
)

func BuiltinPresets(defaultModel string) []Spec {
	model := strings.TrimSpace(defaultModel)
	if model == "" {
		model = "gemini-2.5-flash"
	}

	morphoTools := []gemini.Tool{
		{
			FunctionDeclarations: []gemini.FunctionDeclaration{
				{
					Name:        "run_command",
					Description: "Executa um comando da Morpho CLI. Use argumentos separados (ex: [\"agent\", \"list\"]).",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"args": map[string]interface{}{
								"type": "array",
								"items": map[string]interface{}{
									"type": "string",
								},
								"description": "Lista de argumentos do comando",
							},
						},
						"required": []string{"args"},
					},
				},
				{
					Name:        "run_shell_command",
					Description: "Executa um comando de terminal no workspace (estilo Gemini CLI). Sempre peça confirmação do usuário antes de executar.",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"command": map[string]interface{}{
								"type":        "string",
								"description": "Comando shell completo para executar",
							},
							"working_dir": map[string]interface{}{
								"type":        "string",
								"description": "Diretório de execução (opcional). Ex: . ou ./internal",
							},
							"timeout_seconds": map[string]interface{}{
								"type":        "number",
								"description": "Timeout em segundos (opcional, padrão 30)",
							},
						},
						"required": []string{"command"},
					},
				},
			},
		},
	}

	return []Spec{
		{
			Name:        "morpho",
			Description: "Especialista no ecossistema Morpho e coordenação de agentes.",
			SystemPrompt: "Você é o Morpho, o assistente central e orquestrador deste sistema de agentes. " +
				"Sua missão é ajudar o usuário a gerenciar, criar e executar agentes de desenvolvimento. " +
				"Você tem acesso às ferramentas 'run_command' (CLI Morpho) e 'run_shell_command' (terminal). " +
				"Sempre que precisar executar uma ação (criar agente, listar, ver status), use a ferramenta. " +
				"IMPORTANTE: Sempre peça confirmação explícita ao usuário antes de executar comandos que alterem o sistema (como criar ou editar agentes). " +
				"Para fluxos complexos, você pode delegar tarefas para especialistas: " +
				"1) prompt-engineer: Para criar o prompt perfeito para novos agentes; " +
				"2) backend-go, frontend-react, code-reviewer, qa-tester e devops-ci para tarefas técnicas. " +
				"Ao criar um agente, primeiro peça ao prompt-engineer para gerar o prompt e depois proponha o comando 'agent create'.",
			Model: model,
			Tags:  []string{"morpho", "orchestrator", "guide"},
			Tools: morphoTools,
		},
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
		{
			Name:        "prompt-engineer",
			Description: "Especialista em criar prompts de sistema otimizados para novos agentes.",
			SystemPrompt: "Você é um Engenheiro de Prompt sênior especializado em modelos Gemini. " +
				"Sua tarefa é criar 'System Prompts' detalhados, estruturados e eficazes para novos agentes especialistas. " +
				"Ao receber uma descrição de um novo agente, gere um prompt que inclua: " +
				"1) Persona/Papel claro; " +
				"2) Objetivos e responsabilidades; " +
				"3) Tom de voz e estilo de resposta; " +
				"4) Restrições e o que NÃO fazer; " +
				"5) Exemplos de como o agente deve raciocinar. " +
				"Sempre responda apenas com o texto do prompt sugerido, pronto para ser copiado.",
			Model: model,
			Tags:  []string{"meta", "prompt", "engineering"},
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
