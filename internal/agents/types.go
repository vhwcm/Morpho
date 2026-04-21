package agents

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/vhwcm/Morpho/internal/gemini"
)

const MessagePrefix = "🦋: "

type AIClient interface {
	Generate(ctx context.Context, prompt string) (string, error)
	Chat(ctx context.Context, systemPrompt string, history []gemini.ChatMessage, tools ...gemini.Tool) (gemini.ChatResult, error)
}

type DiagnosticInput struct {
	Problem string
	LogFile string
	AI      AIClient
}

type PlanResult struct {
	Strategy string
}

type LogResult struct {
	Summary      string
	ErrorMatches []string
}

type MetricsResult struct {
	CPUPercent   float64
	MemoryUsedMB float64
	Goroutines   int
	Summary      string
}

type DiagnosticReport struct {
	Problem  string
	Plan     PlanResult
	Logs     LogResult
	Metrics  MetricsResult
	Solution string
	Duration time.Duration
}

func (r DiagnosticReport) String() string {
	var b strings.Builder
	b.WriteString(prefixedLine("=== MorphoSRE Diagnostic Report ==="))
	b.WriteString(prefixedLine(fmt.Sprintf("Problema: %s", r.Problem)))
	b.WriteString("\n")
	b.WriteString(prefixedLine("[Plan Agent]"))
	b.WriteString(prefixedLine(r.Plan.Strategy))
	b.WriteString("\n")
	b.WriteString(prefixedLine("[Log Agent]"))
	b.WriteString(prefixedLine(r.Logs.Summary))
	b.WriteString(prefixedLine(fmt.Sprintf("Ocorrências relevantes: %d", len(r.Logs.ErrorMatches))))
	b.WriteString("\n")
	b.WriteString(prefixedLine("[Metrics Agent]"))
	b.WriteString(prefixedLine(fmt.Sprintf("CPU: %.2f%% | Memória: %.2f MB | Goroutines: %d", r.Metrics.CPUPercent, r.Metrics.MemoryUsedMB, r.Metrics.Goroutines)))
	b.WriteString(prefixedLine(r.Metrics.Summary))
	b.WriteString("\n")
	b.WriteString(prefixedLine("[Solution Agent]"))
	b.WriteString(prefixedLine(r.Solution))
	b.WriteString("\n")
	b.WriteString(prefixedLine(fmt.Sprintf("Tempo total: %s", r.Duration.Round(time.Millisecond))))
	return b.String()
}

func prefixedLine(message string) string {
	return MessagePrefix + strings.TrimSpace(message) + "\n"
}
