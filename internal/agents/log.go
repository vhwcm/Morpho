package agents

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/vhwcm/Morpho/internal/gemini"
)

func RunLogAgent(ctx context.Context, ai AIClient, logFile string) (LogResult, error) {
	f, err := os.Open(logFile)
	if err != nil {
		return LogResult{}, fmt.Errorf("falha ao abrir log %s: %w", logFile, err)
	}
	defer f.Close()

	re := regexp.MustCompile(`(?i)(error|fatal|panic|timeout|\b500\b)`) //nolint:lll

	const maxTail = 400
	tail := make([]string, 0, maxTail)
	var matches []string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return LogResult{}, ctx.Err()
		default:
		}

		line := scanner.Text()
		if len(tail) == maxTail {
			tail = tail[1:]
		}
		tail = append(tail, line)

		if re.MatchString(line) {
			if len(matches) < 25 {
				matches = append(matches, line)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return LogResult{}, err
	}

	summary := fmt.Sprintf("Foram analisadas as últimas %d linhas de %s.", len(tail), logFile)
	if ai != nil {
		prompt := fmt.Sprintf("Resuma em 1-2 frases os principais riscos deste conjunto de logs:\n%s", strings.Join(matches, "\n"))
		if out, err := ai.Chat(ctx, "", []gemini.ChatMessage{{Role: "user", Content: prompt}}); err == nil && strings.TrimSpace(out.Message) != "" {
			summary = out.Message
		}
	}

	if len(matches) == 0 {
		summary += " Nenhum padrão crítico encontrado (error/fatal/panic/timeout/500)."
	}

	return LogResult{
		Summary:      summary,
		ErrorMatches: matches,
	}, nil
}
