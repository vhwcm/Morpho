package agentkit

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type OutputRecord struct {
	Agent     string
	FileName  string
	FilePath  string
	CreatedAt time.Time
}

var invalidFileChars = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

func outputsDir() string {
	return filepath.Join(".morpho", "outputs")
}

func agentOutputDir(agentName string) string {
	return filepath.Join(outputsDir(), agentName)
}

func ensureOutputsDir() error {
	return os.MkdirAll(outputsDir(), 0o755)
}

func ensureAgentOutputDir(agentName string) error {
	if err := ensureOutputsDir(); err != nil {
		return err
	}
	return os.MkdirAll(agentOutputDir(agentName), 0o755)
}

func SaveAgentOutput(agentName, task, output string) (string, error) {
	if strings.TrimSpace(agentName) == "" {
		return "", fmt.Errorf("nome do agente é obrigatório")
	}
	if strings.TrimSpace(output) == "" {
		return "", fmt.Errorf("output vazio")
	}

	if err := ensureAgentOutputDir(agentName); err != nil {
		return "", err
	}

	timestamp := time.Now().UTC().Format("20060102-150405")
	title := sanitizeTitle(task)
	fileName := fmt.Sprintf("%s-%s.md", timestamp, title)
	fullPath := filepath.Join(agentOutputDir(agentName), fileName)

	content := fmt.Sprintf("# Output de Conclusão\n\n- Agent: %s\n- Criado em (UTC): %s\n- Tarefa: %s\n\n## Resultado\n\n%s\n", agentName, time.Now().UTC().Format(time.RFC3339), strings.TrimSpace(task), strings.TrimSpace(output))

	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		return "", err
	}

	return fullPath, nil
}

func ListOutputs(agentName string, limit int) ([]OutputRecord, error) {
	if err := ensureOutputsDir(); err != nil {
		return nil, err
	}

	records := make([]OutputRecord, 0)

	if strings.TrimSpace(agentName) != "" {
		entries, err := os.ReadDir(agentOutputDir(agentName))
		if err != nil {
			if os.IsNotExist(err) {
				return records, nil
			}
			return nil, err
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			fullPath := filepath.Join(agentOutputDir(agentName), e.Name())
			info, err := e.Info()
			if err != nil {
				continue
			}
			records = append(records, OutputRecord{Agent: agentName, FileName: e.Name(), FilePath: fullPath, CreatedAt: info.ModTime()})
		}
	} else {
		agents, err := os.ReadDir(outputsDir())
		if err != nil {
			if os.IsNotExist(err) {
				return records, nil
			}
			return nil, err
		}

		for _, dir := range agents {
			if !dir.IsDir() {
				continue
			}
			agentRecords, err := ListOutputs(dir.Name(), 0)
			if err != nil {
				continue
			}
			records = append(records, agentRecords...)
		}
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].CreatedAt.After(records[j].CreatedAt)
	})

	if limit > 0 && len(records) > limit {
		return records[:limit], nil
	}

	return records, nil
}

func ReadOutput(agentName, fileName string) (string, error) {
	if strings.TrimSpace(agentName) == "" || strings.TrimSpace(fileName) == "" {
		return "", fmt.Errorf("agente e arquivo são obrigatórios")
	}

	cleanName := filepath.Base(fileName)
	fullPath := filepath.Join(agentOutputDir(agentName), cleanName)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

func BuildSharedContext(currentAgent string, maxEntries, maxChars int) (string, error) {
	if maxEntries <= 0 {
		maxEntries = 4
	}
	if maxChars <= 0 {
		maxChars = 5000
	}

	all, err := ListOutputs("", 0)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	used := 0
	entries := 0

	for _, rec := range all {
		if rec.Agent == currentAgent {
			continue
		}
		content, err := os.ReadFile(rec.FilePath)
		if err != nil {
			continue
		}

		chunk := fmt.Sprintf("\n### Agent: %s | Arquivo: %s\n%s\n", rec.Agent, rec.FileName, strings.TrimSpace(string(content)))
		if used+len(chunk) > maxChars {
			remaining := maxChars - used
			if remaining <= 0 {
				break
			}
			chunk = chunk[:remaining]
		}

		b.WriteString(chunk)
		used += len(chunk)
		entries++

		if entries >= maxEntries || used >= maxChars {
			break
		}
	}

	return strings.TrimSpace(b.String()), nil
}

func sanitizeTitle(task string) string {
	t := strings.ToLower(strings.TrimSpace(task))
	if t == "" {
		return "task"
	}
	if len(t) > 48 {
		t = t[:48]
	}
	t = invalidFileChars.ReplaceAllString(t, "-")
	t = strings.Trim(t, "-")
	if t == "" {
		return "task"
	}
	return t
}
