package gemini

import (
	"context"
	"strings"
)

type MockClient struct{}

func NewMockClient() *MockClient {
	return &MockClient{}
}

func (m *MockClient) Generate(_ context.Context, prompt string) (string, error) {
	lower := strings.ToLower(prompt)
	if strings.Contains(lower, "backend") || strings.Contains(lower, "go") {
		return "Plano de implementação: 1) definir contratos e casos de uso, 2) criar handlers com validação, 3) adicionar testes de unidade e integração.", nil
	}
	if strings.Contains(lower, "review") {
		return "Pontos de revisão: validar tratamento de erros, remover duplicações e garantir cobertura de testes nos fluxos críticos.", nil
	}
	return "Sugestão do agente (mock): decompor a tarefa em passos pequenos, implementar incrementalmente e validar com testes.", nil
}
