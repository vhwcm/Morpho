package memory

import "strings"

func ExtractKnowledge(task, output string) string {
	parts := make([]string, 0, 3)
	if t := strings.TrimSpace(task); t != "" {
		parts = append(parts, "TAREFA:\n"+t)
	}
	if o := strings.TrimSpace(output); o != "" {
		parts = append(parts, "RESULTADO:\n"+o)
	}
	joined := strings.Join(parts, "\n\n")
	return Sanitize(joined)
}
