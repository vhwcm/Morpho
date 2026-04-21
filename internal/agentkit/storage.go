package agentkit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/vhwcm/Morpho/internal/logger"
)

var validName = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func agentDir() string {
	return filepath.Join(".morpho", "agents")
}

func specPath(name string) string {
	return filepath.Join(agentDir(), name+".json")
}

func ensureDir() error {
	return os.MkdirAll(agentDir(), 0o755)
}

func SaveSpec(spec Spec) error {
	logger.Debug("Salvando especificação do agente", map[string]interface{}{"name": spec.Name})
	if err := validateSpec(spec); err != nil {
		logger.Error("Validação de especificação falhou", err, map[string]interface{}{"name": spec.Name})
		return err
	}

	if err := ensureDir(); err != nil {
		return err
	}

	now := time.Now().UTC()
	loaded, err := LoadSpec(spec.Name)
	if err == nil {
		spec.CreatedAt = loaded.CreatedAt
	} else {
		spec.CreatedAt = now
	}
	spec.UpdatedAt = now

	p, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		logger.Error("Erro ao serializar especificação do agente", err, map[string]interface{}{"name": spec.Name})
		return err
	}

	err = os.WriteFile(specPath(spec.Name), p, 0o644)
	if err != nil {
		logger.Error("Erro ao escrever arquivo do agente", err, map[string]interface{}{"path": specPath(spec.Name)})
	}
	return err
}

func LoadSpec(name string) (Spec, error) {
	if strings.TrimSpace(name) == "" {
		return Spec{}, fmt.Errorf("nome do agente é obrigatório")
	}

	content, err := os.ReadFile(specPath(name))
	if err != nil {
		if os.IsNotExist(err) {
			return Spec{}, fmt.Errorf("agente '%s' não encontrado", name)
		}
		logger.Error("Erro ao ler arquivo do agente", err, map[string]interface{}{"name": name})
		return Spec{}, err
	}

	var spec Spec
	if err := json.Unmarshal(content, &spec); err != nil {
		logger.Error("Erro ao desserializar agente", err, map[string]interface{}{"name": name})
		return Spec{}, err
	}
	return spec, nil
}

func DeleteSpec(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("nome do agente é obrigatório")
	}
	path := specPath(name)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("agente '%s' não encontrado", name)
	}
	return os.Remove(path)
}

func ListSpecs() ([]Spec, error) {
	if err := ensureDir(); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(agentDir())
	if err != nil {
		return nil, err
	}

	list := make([]Spec, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}

		name := strings.TrimSuffix(e.Name(), ".json")
		spec, err := LoadSpec(name)
		if err != nil {
			continue
		}
		list = append(list, spec)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})

	return list, nil
}

func validateSpec(spec Spec) error {
	if strings.TrimSpace(spec.Name) == "" {
		return fmt.Errorf("nome do agente é obrigatório")
	}
	if !validName.MatchString(spec.Name) {
		return fmt.Errorf("nome inválido: use apenas letras, números, '_' ou '-' ")
	}
	if strings.TrimSpace(spec.SystemPrompt) == "" {
		return fmt.Errorf("prompt do agente é obrigatório")
	}
	if strings.TrimSpace(spec.Model) == "" {
		return fmt.Errorf("modelo do agente é obrigatório")
	}
	return nil
}
