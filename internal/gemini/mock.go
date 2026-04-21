package gemini

import (
	"context"
	"hash/fnv"
	"strings"
)

type MockClient struct{}

func NewMockClient() *MockClient {
	return &MockClient{}
}

func (m *MockClient) Generate(ctx context.Context, prompt string) (string, error) {
	res, err := m.Chat(ctx, "", []ChatMessage{{Role: "user", Content: prompt}})
	if err != nil {
		return "", err
	}
	return res.Message, nil
}

func (m *MockClient) Chat(ctx context.Context, systemPrompt string, history []ChatMessage, tools ...Tool) (ChatResult, error) {
	if len(history) == 0 {
		return ChatResult{Message: "Olá, em que posso ajudar?"}, nil
	}
	
	lastMsg := strings.ToLower(history[len(history)-1].Content)
	
	if strings.Contains(lastMsg, "backend") || strings.Contains(lastMsg, "go") {
		return ChatResult{Message: "Plano de implementação: 1) definir contratos e casos de uso, 2) criar handlers com validação, 3) adicionar testes de unidade e integração."}, nil
	}
	if strings.Contains(lastMsg, "review") {
		return ChatResult{Message: "Pontos de revisão: validar tratamento de erros, remover duplicações e garantir cobertura de testes nos fluxos críticos."}, nil
	}
	return ChatResult{Message: "Sugestão do agente (mock): decompor a tarefa em passos pequenos, implementar incrementalmente e validar com testes."}, nil
}

func (m *MockClient) Embed(_ context.Context, text string) ([]float64, error) {
	base := strings.TrimSpace(strings.ToLower(text))
	if base == "" {
		return []float64{0, 0, 0, 0, 0, 0, 0, 0}, nil
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(base))
	v := h.Sum64()
	out := make([]float64, 8)
	for i := 0; i < len(out); i++ {
		part := (v >> (i * 8)) & 0xFF
		out[i] = float64(part) / 255.0
	}
	return out, nil
}
