package agentkit

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/vhwcm/Morpho/internal/gemini"
)

type fakeAI struct {
	errs      []error
	responses []string
	calls     int
	prompts   []string
}

func (f *fakeAI) Generate(_ context.Context, prompt string) (string, error) {
	f.calls++
	f.prompts = append(f.prompts, prompt)
	idx := f.calls - 1

	if idx < len(f.errs) && f.errs[idx] != nil {
		return "", f.errs[idx]
	}

	if idx < len(f.responses) {
		return f.responses[idx], nil
	}
	if len(f.responses) > 0 {
		return f.responses[len(f.responses)-1], nil
	}
	return "ok", nil
}

func withTempWD(t *testing.T) {
	t.Helper()

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("falha ao ler cwd: %v", err)
	}

	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("falha ao trocar cwd: %v", err)
	}

	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})
}

func TestBuiltinPresetsModelSelection(t *testing.T) {
	presets := BuiltinPresets("")
	if len(presets) == 0 {
		t.Fatalf("esperava presets padrão")
	}
	for _, p := range presets {
		if p.Model != "gemini-2.0-flash" {
			t.Fatalf("modelo padrão inesperado: %s", p.Model)
		}
	}

	custom := BuiltinPresets("gemini-2.5-flash")
	for _, p := range custom {
		if p.Model != "gemini-2.5-flash" {
			t.Fatalf("modelo customizado não aplicado: %s", p.Model)
		}
	}
}

func TestSeedPresetsForceAndOverwrite(t *testing.T) {
	withTempWD(t)

	expected := len(BuiltinPresets("gemini-2.0-flash"))

	created, err := SeedPresets(false, "gemini-2.0-flash")
	if err != nil {
		t.Fatalf("erro ao iniciar presets: %v", err)
	}
	if created != expected {
		t.Fatalf("quantidade criada inesperada: got=%d want=%d", created, expected)
	}

	created, err = SeedPresets(false, "gemini-2.5-flash")
	if err != nil {
		t.Fatalf("erro ao reiniciar presets sem force: %v", err)
	}
	if created != 0 {
		t.Fatalf("sem --force deveria criar 0 presets, got=%d", created)
	}

	backend, err := LoadSpec("backend-go")
	if err != nil {
		t.Fatalf("erro ao carregar spec backend-go: %v", err)
	}
	if backend.Model != "gemini-2.0-flash" {
		t.Fatalf("modelo não deveria ter mudado sem force: %s", backend.Model)
	}

	created, err = SeedPresets(true, "gemini-2.5-flash")
	if err != nil {
		t.Fatalf("erro ao reiniciar presets com force: %v", err)
	}
	if created != expected {
		t.Fatalf("com --force deveria recriar todos, got=%d want=%d", created, expected)
	}

	backend, err = LoadSpec("backend-go")
	if err != nil {
		t.Fatalf("erro ao recarregar spec backend-go: %v", err)
	}
	if backend.Model != "gemini-2.5-flash" {
		t.Fatalf("modelo deveria ter sido sobrescrito: %s", backend.Model)
	}
}

func TestSaveLoadAndListSpecs(t *testing.T) {
	withTempWD(t)

	if err := SaveSpec(Spec{Name: "invalido nome", SystemPrompt: "x", Model: "m"}); err == nil {
		t.Fatalf("esperava erro para nome inválido")
	}

	if err := SaveSpec(Spec{Name: "zeta", SystemPrompt: "prompt z", Model: "m"}); err != nil {
		t.Fatalf("erro ao salvar spec zeta: %v", err)
	}
	if err := SaveSpec(Spec{Name: "alpha", SystemPrompt: "prompt a", Model: "m"}); err != nil {
		t.Fatalf("erro ao salvar spec alpha: %v", err)
	}

	spec, err := LoadSpec("alpha")
	if err != nil {
		t.Fatalf("erro ao carregar alpha: %v", err)
	}
	if spec.Name != "alpha" {
		t.Fatalf("nome inesperado: %s", spec.Name)
	}
	if spec.CreatedAt.IsZero() || spec.UpdatedAt.IsZero() {
		t.Fatalf("timestamps deveriam estar preenchidos")
	}

	createdAt := spec.CreatedAt
	updatedAt := spec.UpdatedAt
	time.Sleep(5 * time.Millisecond)

	spec.Description = "alterado"
	if err := SaveSpec(spec); err != nil {
		t.Fatalf("erro ao atualizar spec: %v", err)
	}

	updated, err := LoadSpec("alpha")
	if err != nil {
		t.Fatalf("erro ao recarregar alpha: %v", err)
	}
	if !updated.CreatedAt.Equal(createdAt) {
		t.Fatalf("created_at não deveria mudar")
	}
	if !updated.UpdatedAt.After(updatedAt) {
		t.Fatalf("updated_at deveria ser mais recente")
	}

	list, err := ListSpecs()
	if err != nil {
		t.Fatalf("erro ao listar specs: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("quantidade inesperada de specs: %d", len(list))
	}
	if list[0].Name != "alpha" || list[1].Name != "zeta" {
		t.Fatalf("lista deveria estar ordenada por nome: %+v", []string{list[0].Name, list[1].Name})
	}
}

func TestOutputsReadListAndSharedContext(t *testing.T) {
	withTempWD(t)

	if _, err := SaveAgentOutput("", "tarefa", "resultado"); err == nil {
		t.Fatalf("esperava erro para agente vazio")
	}

	if _, err := SaveAgentOutput("backend-go", "Implementar API JWT", "  concluído  "); err != nil {
		t.Fatalf("erro ao salvar output backend-go: %v", err)
	}
	if _, err := SaveAgentOutput("qa-tester", "Cobrir testes de autenticação", "cenários sugeridos"); err != nil {
		t.Fatalf("erro ao salvar output qa-tester: %v", err)
	}

	records, err := ListOutputs("backend-go", 10)
	if err != nil {
		t.Fatalf("erro ao listar outputs: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("esperava 1 output de backend-go, got=%d", len(records))
	}
	if !strings.HasSuffix(records[0].FileName, ".md") {
		t.Fatalf("arquivo deveria ser markdown: %s", records[0].FileName)
	}

	content, err := ReadOutput("backend-go", records[0].FileName)
	if err != nil {
		t.Fatalf("erro ao ler output: %v", err)
	}
	if !strings.Contains(content, "## Resultado") || !strings.Contains(content, "concluído") {
		t.Fatalf("conteúdo inesperado no output: %s", content)
	}

	shared, err := BuildSharedContext("backend-go", 10, 10000)
	if err != nil {
		t.Fatalf("erro ao construir contexto compartilhado: %v", err)
	}
	if strings.Contains(shared, "Agent: backend-go") {
		t.Fatalf("contexto não deveria incluir outputs do agente atual")
	}
	if !strings.Contains(shared, "Agent: qa-tester") {
		t.Fatalf("contexto deveria incluir output de outro agente")
	}

	if _, err := SaveAgentOutput("frontend-react", "Refinar UI", "ok"); err != nil {
		t.Fatalf("erro ao salvar output frontend-react: %v", err)
	}
	limited, err := BuildSharedContext("backend-go", 1, 10000)
	if err != nil {
		t.Fatalf("erro ao construir contexto limitado: %v", err)
	}
	if strings.Count(limited, "### Agent:") != 1 {
		t.Fatalf("maxEntries=1 deveria incluir apenas um bloco")
	}
}

func TestRunUsesPromptAndTrimsOutput(t *testing.T) {
	ai := &fakeAI{responses: []string{"  resposta final  "}}
	spec := Spec{Name: "a", SystemPrompt: "seja objetivo", Model: "m"}

	out, err := Run(context.Background(), ai, spec, "resolver bug")
	if err != nil {
		t.Fatalf("erro no run: %v", err)
	}
	if out != "resposta final" {
		t.Fatalf("run deveria aplicar trim no output: %q", out)
	}
	if ai.calls != 1 {
		t.Fatalf("generate deveria ser chamado uma vez")
	}
	if !strings.Contains(ai.prompts[0], "Instruções do agente") || !strings.Contains(ai.prompts[0], "resolver bug") {
		t.Fatalf("prompt montado não contém contexto esperado: %s", ai.prompts[0])
	}
}

func TestQueueRetriesAndStopsOnNonRetryable(t *testing.T) {
	spec := Spec{Name: "queue-agent", SystemPrompt: "prompt", Model: "m"}

	t.Run("retryable error eventually succeeds", func(t *testing.T) {
		q := NewQueueManager(8, 2, time.Millisecond)
		ai := &fakeAI{
			errs:      []error{context.DeadlineExceeded, context.DeadlineExceeded, nil},
			responses: []string{"", "", "ok"},
		}

		out, err := q.Enqueue(context.Background(), QueueRequest{AI: ai, Spec: spec, Task: "executar"})
		if err != nil {
			t.Fatalf("deveria ter sucesso após retries: %v", err)
		}
		if out != "ok" {
			t.Fatalf("output inesperado: %s", out)
		}
		if ai.calls != 3 {
			t.Fatalf("deveria tentar 3 vezes, tentou %d", ai.calls)
		}
	})

	t.Run("non-retryable error fails fast", func(t *testing.T) {
		q := NewQueueManager(8, 5, time.Millisecond)
		ai := &fakeAI{errs: []error{errors.New("falha fatal")}}

		_, err := q.Enqueue(context.Background(), QueueRequest{AI: ai, Spec: spec, Task: "executar"})
		if err == nil {
			t.Fatalf("esperava erro para falha não reprocessável")
		}
		if ai.calls != 1 {
			t.Fatalf("não deveria fazer retry para erro não reprocessável, calls=%d", ai.calls)
		}
	})
}

func TestIsRetryableQueueError(t *testing.T) {
	if !isRetryableQueueError(context.DeadlineExceeded) {
		t.Fatalf("deadline exceeded deveria ser retryable")
	}
	if !isRetryableQueueError(&gemini.APIError{StatusCode: 429, Body: "rate limit"}) {
		t.Fatalf("429 deveria ser retryable")
	}
	if isRetryableQueueError(&gemini.APIError{StatusCode: 400, Body: "bad request"}) {
		t.Fatalf("400 não deveria ser retryable")
	}
	if !isRetryableQueueError(errors.New("request timeout")) {
		t.Fatalf("mensagem com timeout deveria ser retryable")
	}
}
