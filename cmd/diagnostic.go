package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/vhwcm/Morpho/internal/agentkit"
	"github.com/vhwcm/Morpho/internal/config"
	"github.com/vhwcm/Morpho/internal/gemini"
	"github.com/vhwcm/Morpho/internal/memory"
	"github.com/vhwcm/Morpho/internal/ui"
)

var (
	runTimeout        time.Duration
	runMockAI         bool
	queueRetry        int
	queueDelay        time.Duration
	runContextEntries int
	runContextChars   int
	runNoContext      bool

	agentDescription string
	agentPrompt      string
	agentModel       string
	agentName        string
	agentTags        string
	presetsForce     bool
	presetsModel     string

	modelTimeout time.Duration
	modelShowAll bool
	modelValue   string

	outputListLimit int

	configAPIKey string

	runEditEnabled  bool
	runEditMode     string
	runEditPaths    string
	runEditMaxEdits int
	runEditYes      bool

	runRAGEnabled  bool
	runNoRAG       bool
	runRAGTopK     int
	runRAGMinScore float64

	configEditMode  string
	configEditPaths string

	configMemoryReadPolicy string
	configMemoryTTLHours   int
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Gerencia agentes de desenvolvimento",
}

var agentListCmd = &cobra.Command{
	Use:   "list",
	Short: "Lista agentes cadastrados",
	RunE: func(_ *cobra.Command, _ []string) error {
		ui.Header("Agentes cadastrados")

		agents, err := agentkit.ListSpecs()
		if err != nil {
			return err
		}

		if len(agents) == 0 {
			ui.Warn("Nenhum agente encontrado. Use 'morpho presets init' para iniciar com agentes pré-selecionados.")
			return nil
		}

		rows := make([][]string, 0, len(agents))
		for _, a := range agents {
			tags := "-"
			if len(a.Tags) > 0 {
				tags = strings.Join(a.Tags, ",")
			}
			rows = append(rows, []string{a.Name, a.Model, tags})
		}
		ui.Table([]string{"Agente", "Modelo", "Tags"}, rows)
		ui.Success(fmt.Sprintf("%d agente(s) listado(s).", len(rows)))

		return nil
	},
}

var agentShowCmd = &cobra.Command{
	Use:   "show [nome]",
	Short: "Exibe especificação de um agente",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		a, err := agentkit.LoadSpec(args[0])
		if err != nil {
			return err
		}

		ui.Header("Detalhes do agente")
		tags := "-"
		if len(a.Tags) > 0 {
			tags = strings.Join(a.Tags, ",")
		}
		ui.Table(
			[]string{"Campo", "Valor"},
			[][]string{
				{"Nome", a.Name},
				{"Descrição", a.Description},
				{"Modelo", a.Model},
				{"Tags", tags},
			},
		)
		ui.Panel("System Prompt", a.SystemPrompt)
		return nil
	},
}

var agentCreateCmd = &cobra.Command{
	Use:   "create [nome]",
	Short: "Cria um novo agente via CLI",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		name := args[0]
		if agentPrompt == "" {
			return errors.New("informe --prompt para criar o agente")
		}
		if agentModel == "" {
			agentModel = "gemini-2.5-flash"
		}

		spec := agentkit.Spec{
			Name:         name,
			Description:  agentDescription,
			SystemPrompt: agentPrompt,
			Model:        agentModel,
			Tags:         splitTags(agentTags),
		}

		if err := agentkit.SaveSpec(spec); err != nil {
			return err
		}

		ui.Success(fmt.Sprintf("Agente '%s' criado com sucesso.", name))
		return nil
	},
}

var agentEditCmd = &cobra.Command{
	Use:   "edit [nome]",
	Short: "Edita especificações de um agente existente",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		oldName := args[0]
		spec, err := agentkit.LoadSpec(oldName)
		if err != nil {
			return err
		}

		if cmd.Flags().Changed("description") {
			spec.Description = agentDescription
		}
		if cmd.Flags().Changed("prompt") {
			spec.SystemPrompt = agentPrompt
		}
		if cmd.Flags().Changed("model") {
			spec.Model = agentModel
		}
		if cmd.Flags().Changed("tags") {
			spec.Tags = splitTags(agentTags)
		}

		if cmd.Flags().Changed("name") {
			newName := strings.TrimSpace(agentName)
			if newName != "" && newName != oldName {
				if err := agentkit.DeleteSpec(oldName); err != nil {
					return fmt.Errorf("erro ao remover agente antigo: %w", err)
				}
				spec.Name = newName
				ui.Info(fmt.Sprintf("Agente renomeado de '%s' para '%s'.", oldName, newName))
			}
		}

		if err := agentkit.SaveSpec(spec); err != nil {
			return err
		}

		ui.Success(fmt.Sprintf("Agente '%s' atualizado.", spec.Name))
		return nil
	},
}

var agentSetModelCmd = &cobra.Command{
	Use:   "set-model [nome] [modelo]",
	Short: "Configura o modelo Gemini de um agente",
	Args:  cobra.ExactArgs(2),
	RunE: func(_ *cobra.Command, args []string) error {
		spec, err := agentkit.LoadSpec(args[0])
		if err != nil {
			return err
		}

		spec.Model = strings.TrimSpace(args[1])
		if err := agentkit.SaveSpec(spec); err != nil {
			return err
		}

		ui.Success(fmt.Sprintf("Modelo do agente '%s' atualizado para '%s'.", spec.Name, spec.Model))
		return nil
	},
}

var agentOutputCmd = &cobra.Command{
	Use:   "output",
	Short: "Gerencia os outputs gerados pelos agentes",
}

var agentOutputListCmd = &cobra.Command{
	Use:   "list [nome-agente]",
	Short: "Lista outputs salvos (de um agente específico ou de todos)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		agentName := ""
		if len(args) == 1 {
			agentName = strings.TrimSpace(args[0])
		}

		records, err := agentkit.ListOutputs(agentName, outputListLimit)
		if err != nil {
			return err
		}

		ui.Header("Outputs dos agentes")
		if len(records) == 0 {
			ui.Warn("Nenhum output encontrado.")
			return nil
		}

		rows := make([][]string, 0, len(records))
		for _, r := range records {
			rows = append(rows, []string{r.Agent, r.FileName, r.CreatedAt.Format(time.RFC3339)})
		}
		ui.Table([]string{"Agente", "Arquivo", "Criado em (UTC)"}, rows)
		ui.Success(fmt.Sprintf("%d output(s) listado(s).", len(rows)))
		return nil
	},
}

var agentOutputShowCmd = &cobra.Command{
	Use:     "show [nome-agente] [arquivo]",
	Aliases: []string{"view"},
	Short:   "Exibe o conteúdo de um output específico",
	Args:    cobra.ExactArgs(2),
	RunE: func(_ *cobra.Command, args []string) error {
		content, err := agentkit.ReadOutput(args[0], args[1])
		if err != nil {
			return err
		}

		ui.Header("Visualização de output")
		ui.Panel(fmt.Sprintf("%s / %s", args[0], args[1]), content)
		return nil
	},
}

var agentOutputLastCmd = &cobra.Command{
	Use:   "last [nome-agente]",
	Short: "Exibe o conteúdo do output mais recente de um agente",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		records, err := agentkit.ListOutputs(args[0], 1)
		if err != nil {
			return err
		}

		if len(records) == 0 {
			ui.Warn(fmt.Sprintf("Nenhum output encontrado para o agente '%s'.", args[0]))
			return nil
		}

		last := records[0]
		content, err := agentkit.ReadOutput(last.Agent, last.FileName)
		if err != nil {
			return err
		}

		ui.Header("Último output do agente")
		ui.Panel(fmt.Sprintf("%s / %s", last.Agent, last.FileName), content)
		return nil
	},
}

var globalViewCmd = &cobra.Command{
	Use:   "view [nome-agente] [arquivo]",
	Short: "Alias rápido para visualizar output de um agente",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return agentOutputShowCmd.RunE(cmd, args)
	},
}

var agentRunCmd = &cobra.Command{
	Use:   "run [nome] [tarefa]",
	Short: "Executa um agente para uma tarefa de desenvolvimento",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		spec, err := agentkit.LoadSpec(args[0])
		if err != nil {
			return err
		}

		cfg := config.Load()

		var ai agentkit.AIClient
		if runMockAI {
			ai = gemini.NewMockClient()
		} else {
			client, err := gemini.NewClient(cfg.GeminiAPIKey, spec.Model)
			if err != nil {
				if errors.Is(err, gemini.ErrMissingAPIKey) {
					return fmt.Errorf("GEMINI_API_KEY não definida: use --mock para execução offline")
				}
				return err
			}
			ai = client
		}

		agentkit.ConfigureDefaultQueue(queueRetry, queueDelay)
		ui.Header("Execução de agente")
		ui.Info(fmt.Sprintf("Agente: %s | Modelo: %s | Retries: %d | Delay: %s", spec.Name, spec.Model, queueRetry, queueDelay))

		originalTask := args[1]
		task := originalTask

		ragEnabled := cfg.Memory.Enabled
		if runRAGEnabled {
			ragEnabled = true
		}
		if runNoRAG {
			ragEnabled = false
		}

		topK := cfg.Memory.TopK
		if runRAGTopK > 0 {
			topK = runRAGTopK
		}
		minScore := cfg.Memory.MinScore
		if runRAGMinScore > 0 {
			minScore = runRAGMinScore
		}

		if ragEnabled {
			ragResults, err := memory.SearchWithPolicy(cmd.Context(), spec.Name, originalTask, topK, minScore, ai, cfg.Memory.ReadPolicy)
			if err != nil {
				ui.Warn(fmt.Sprintf("RAG indisponível nesta execução: %s", err.Error()))
			} else if len(ragResults) > 0 {
				ragCtx := memory.BuildRAGContext(ragResults, cfg.Memory.MaxChars)
				if ragCtx != "" {
					task += "\n\nContexto RAG relevante:\n" + ragCtx
					ui.Info(fmt.Sprintf("Contexto RAG anexado (%d memória(s), policy=%s).", len(ragResults), cfg.Memory.ReadPolicy))
				}
			}
		}

		if !runNoContext {
			shared, err := agentkit.BuildSharedContext(spec.Name, runContextEntries, runContextChars)
			if err != nil {
				return err
			}
			if shared != "" {
				task = task + "\n\nContexto de outputs de outros agentes:\n" + shared
				ui.Info("Contexto compartilhado de outputs anexado à execução.")
			}
		}

		out, err := agentkit.RunQueued(cmd.Context(), agentkit.QueueRequest{
			AI:             ai,
			Spec:           spec,
			Task:           task,
			AttemptTimeout: runTimeout,
		})
		if err != nil {
			return err
		}

		ui.Panel(fmt.Sprintf("Resposta do agente '%s'", spec.Name), out.Message)

		if runEditEnabled {
			mode := strings.TrimSpace(runEditMode)
			if mode == "" {
				mode = cfg.AgentEditing.Mode
			}

			normalizedMode, err := config.NormalizeEditMode(mode)
			if err != nil {
				return err
			}

			if normalizedMode == config.EditModeOff {
				ui.Warn("Edição por agentes está desativada (mode=off). Use 'morpho config edit set-mode review|auto' ou --edit-mode.")
			} else {
				allowedPaths := cfg.AgentEditing.AllowedPaths
				if strings.TrimSpace(runEditPaths) != "" {
					allowedPaths = splitCSV(runEditPaths)
				}

				workspaceRoot, err := os.Getwd()
				if err != nil {
					return err
				}

				editTask := agentkit.BuildEditTask(originalTask, allowedPaths, runEditMaxEdits)
				editTask += "\n\nContexto adicional da execução anterior:\n" + out.Message

				planRaw, err := agentkit.RunQueued(cmd.Context(), agentkit.QueueRequest{
					AI:             ai,
					Spec:           spec,
					Task:           editTask,
					AttemptTimeout: runTimeout,
				})
				if err != nil {
					return fmt.Errorf("falha ao gerar plano de edição: %w", err)
				}

				plan, err := agentkit.ParseEditPlan(planRaw.Message)
				if err != nil {
					return err
				}

				if err := agentkit.ValidateEditPlan(plan, allowedPaths, runEditMaxEdits); err != nil {
					return err
				}

				ui.Header("Plano de edição de arquivos")
				if strings.TrimSpace(plan.Summary) != "" {
					ui.Info(plan.Summary)
				}

				if len(plan.Edits) == 0 {
					ui.Warn("Nenhuma edição proposta pelo agente.")
				} else {
					previewRows := make([][]string, 0, len(plan.Edits))
					for i, edit := range plan.Edits {
						normalized, _ := agentkit.NormalizeRelativePath(edit.Path)
						previewRows = append(previewRows, []string{fmt.Sprintf("%d", i+1), normalized, strings.TrimSpace(edit.Summary)})
					}
					ui.Table([]string{"#", "Arquivo", "Resumo"}, previewRows)

					appliedRows := make([][]string, 0, len(plan.Edits))
					skipped := 0
					for i, edit := range plan.Edits {
						normalized, _ := agentkit.NormalizeRelativePath(edit.Path)
						approved := true
						if normalizedMode == config.EditModeReview && !runEditYes {
							approved, err = promptApproval(fmt.Sprintf("Aplicar edição %d/%d em %s?", i+1, len(plan.Edits), normalized))
							if err != nil {
								return err
							}
						}
						if !approved {
							skipped++
							continue
						}

						edit.Path = normalized
						res, err := agentkit.ApplyFileEdit(workspaceRoot, edit)
						if err != nil {
							return err
						}
						status := "sem mudança"
						if res.Changed {
							if res.Created {
								status = "criado"
							} else {
								status = "atualizado"
							}
						}

						backup := "-"
						if res.BackupPath != "" {
							backup = filepath.ToSlash(res.BackupPath)
						}
						appliedRows = append(appliedRows, []string{res.Path, status, backup})
					}

					if len(appliedRows) > 0 {
						ui.Header("Resultado da aplicação")
						ui.Table([]string{"Arquivo", "Status", "Backup"}, appliedRows)
					}
					if skipped > 0 {
						ui.Warn(fmt.Sprintf("%d edição(ões) ignorada(s) por aprovação manual.", skipped))
					}
				}
			}
		}

		savedPath, err := agentkit.SaveAgentOutput(spec.Name, args[1], out.Message)
		if err != nil {
			return err
		}

		if ragEnabled {
			err = memory.IngestRun(cmd.Context(), ai, memory.IngestInput{
				Agent:    spec.Name,
				Task:     originalTask,
				Output:   out.Message,
				Source:   savedPath,
				MaxChars: cfg.Memory.MaxChars,
				TTLHours: cfg.Memory.TTLHours,
			})
			if err != nil {
				ui.Warn(fmt.Sprintf("Falha ao ingerir memória do agente: %s", err.Error()))
			} else {
				ui.Info("Memória do agente atualizada com esta execução.")
			}
		}

		ui.Info(fmt.Sprintf("Output salvo em: %s", savedPath))
		ui.Success("Execução finalizada com sucesso.")
		return nil
	},
}

var presetsCmd = &cobra.Command{
	Use:   "presets",
	Short: "Gerencia agentes pré-selecionados",
}

var modelCmd = &cobra.Command{
	Use:   "model",
	Short: "Consulta modelos disponíveis do Gemini",
}

var modelSetCmd = &cobra.Command{
	Use:   "set [modelo]",
	Short: "Define o modelo padrão do Morpho",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		value := strings.TrimSpace(modelValue)
		if len(args) == 1 {
			value = strings.TrimSpace(args[0])
		}
		if value == "" {
			return fmt.Errorf("informe o modelo por argumento ou --value")
		}

		if err := config.SaveGeminiModel(value); err != nil {
			return err
		}

		ui.Success(fmt.Sprintf("Modelo padrão atualizado para '%s'.", value))
		ui.Info("Esse modelo será usado por padrão quando aplicável.")
		return nil
	},
}

var modelSetAgentCmd = &cobra.Command{
	Use:   "set-agent [nome-agente] [modelo]",
	Short: "Define o modelo de um agente específico",
	Args:  cobra.ExactArgs(2),
	RunE: func(_ *cobra.Command, args []string) error {
		spec, err := agentkit.LoadSpec(args[0])
		if err != nil {
			return err
		}

		spec.Model = strings.TrimSpace(args[1])
		if err := agentkit.SaveSpec(spec); err != nil {
			return err
		}

		ui.Success(fmt.Sprintf("Modelo do agente '%s' atualizado para '%s'.", spec.Name, spec.Model))
		return nil
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Gerencia configurações locais do Morpho",
}

var configSetAPIKeyCmd = &cobra.Command{
	Use:   "set-api-key [api_key]",
	Short: "Define a GEMINI_API_KEY em arquivo local fora do repositório",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		key := strings.TrimSpace(configAPIKey)
		if len(args) == 1 {
			key = strings.TrimSpace(args[0])
		}
		if key == "" {
			return fmt.Errorf("informe a api key por argumento ou --key")
		}

		if err := config.SaveGeminiAPIKey(key); err != nil {
			return err
		}

		path, err := config.ConfigFilePath()
		if err != nil {
			return err
		}

		ui.Success("API key salva com sucesso em arquivo local seguro.")
		ui.Info(fmt.Sprintf("Arquivo de configuração: %s", path))
		ui.Warn("Esse arquivo fica fora do repositório.")
		return nil
	},
}

var configWhereCmd = &cobra.Command{
	Use:   "where",
	Short: "Mostra onde a configuração local é armazenada",
	RunE: func(_ *cobra.Command, _ []string) error {
		path, err := config.ConfigFilePath()
		if err != nil {
			return err
		}
		ui.Header("Configuração local")
		ui.Info(fmt.Sprintf("Arquivo: %s", path))
		ui.Warn("Local externo ao repositório para evitar commit de segredos.")
		return nil
	},
}

var configListModelsCmd = &cobra.Command{
	Use:   "list-models",
	Short: "Lista os modelos disponíveis para sua API Key",
	RunE: func(_ *cobra.Command, _ []string) error {
		apiKey := config.GetGeminiAPIKey()
		if apiKey == "" {
			return fmt.Errorf("GEMINI_API_KEY não configurada")
		}

		c, err := gemini.NewClient(apiKey, "")
		if err != nil {
			return err
		}

		models, err := c.ListModels(context.Background())
		if err != nil {
			return err
		}

		ui.Header("Modelos Disponíveis")
		var rows [][]string
		for _, m := range models {
			methods := strings.Join(m.SupportedGenerationMethods, ", ")
			rows = append(rows, []string{m.Name, m.DisplayName, methods})
		}
		ui.Table([]string{"ID", "Nome", "Métodos"}, rows)
		return nil
	},
}

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Gerencia política de edição de arquivos por agentes",
}

var configMemoryCmd = &cobra.Command{
	Use:   "memory",
	Short: "Gerencia política de memória semântica (RAG)",
}

var configMemoryShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Exibe a configuração atual da memória semântica",
	RunE: func(_ *cobra.Command, _ []string) error {
		cfg := config.Load()
		ui.Header("Configuração de memória semântica")
		ui.Table(
			[]string{"Campo", "Valor"},
			[][]string{
				{"Enabled", fmt.Sprintf("%t", cfg.Memory.Enabled)},
				{"Read Policy", cfg.Memory.ReadPolicy},
				{"Cross Agent Read", fmt.Sprintf("%t", cfg.Memory.CrossAgentRead)},
				{"TTL Hours", fmt.Sprintf("%d", cfg.Memory.TTLHours)},
				{"Top K", fmt.Sprintf("%d", cfg.Memory.TopK)},
				{"Min Score", fmt.Sprintf("%.3f", cfg.Memory.MinScore)},
				{"Max Chars", fmt.Sprintf("%d", cfg.Memory.MaxChars)},
			},
		)
		ui.Info("Read policy: self (somente memória do próprio agente) | shared (permite memória de outros agentes).")
		return nil
	},
}

var configMemorySetReadPolicyCmd = &cobra.Command{
	Use:   "set-read-policy [self|shared]",
	Short: "Define a política explícita de leitura entre agentes",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		policy := strings.TrimSpace(configMemoryReadPolicy)
		if len(args) == 1 {
			policy = strings.TrimSpace(args[0])
		}
		if policy == "" {
			return fmt.Errorf("informe a policy por argumento ou --policy")
		}
		if err := config.SaveMemoryReadPolicy(policy); err != nil {
			return err
		}
		normalized, _ := config.NormalizeMemoryReadPolicy(policy)
		ui.Success(fmt.Sprintf("Read policy de memória atualizada para '%s'.", normalized))
		return nil
	},
}

var configMemorySetTTLCmd = &cobra.Command{
	Use:   "set-ttl-hours [horas]",
	Short: "Define retenção por TTL em horas para memória persistida",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		ttlHours := configMemoryTTLHours
		if len(args) == 1 {
			v, err := strconv.Atoi(strings.TrimSpace(args[0]))
			if err != nil {
				return fmt.Errorf("ttl inválido: %w", err)
			}
			ttlHours = v
		}
		if ttlHours <= 0 {
			return fmt.Errorf("ttl em horas deve ser maior que zero")
		}
		if err := config.SaveMemoryTTLHours(ttlHours); err != nil {
			return err
		}
		ui.Success(fmt.Sprintf("TTL da memória atualizado para %d hora(s).", ttlHours))
		return nil
	},
}

var configEditShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Exibe a configuração atual de edição por agentes",
	RunE: func(_ *cobra.Command, _ []string) error {
		cfg := config.Load()
		allowed := "(todos os caminhos do workspace)"
		if len(cfg.AgentEditing.AllowedPaths) > 0 {
			allowed = strings.Join(cfg.AgentEditing.AllowedPaths, ", ")
		}

		ui.Header("Configuração de edição por agentes")
		ui.Table(
			[]string{"Campo", "Valor"},
			[][]string{
				{"Mode", cfg.AgentEditing.Mode},
				{"Allowed Paths", allowed},
			},
		)
		ui.Info("Modes: off (desligado), review (aprovação por edição), auto (aplica direto).")
		return nil
	},
}

var configEditSetModeCmd = &cobra.Command{
	Use:   "set-mode [off|review|auto]",
	Short: "Define o modo de aplicação das edições de arquivos",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		mode := strings.TrimSpace(configEditMode)
		if len(args) == 1 {
			mode = strings.TrimSpace(args[0])
		}
		if mode == "" {
			return fmt.Errorf("informe o modo por argumento ou --mode")
		}

		if err := config.SaveAgentEditMode(mode); err != nil {
			return err
		}

		normalized, _ := config.NormalizeEditMode(mode)
		ui.Success(fmt.Sprintf("Modo de edição atualizado para '%s'.", normalized))
		return nil
	},
}

var configEditSetPathsCmd = &cobra.Command{
	Use:   "set-paths [paths_csv]",
	Short: "Define os caminhos permitidos para edição (prefixos relativos)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		raw := strings.TrimSpace(configEditPaths)
		if len(args) == 1 {
			raw = strings.TrimSpace(args[0])
		}
		paths := splitCSV(raw)
		if err := config.SaveAgentEditAllowedPaths(paths); err != nil {
			return err
		}

		if len(paths) == 0 {
			ui.Warn("Lista de caminhos permitidos limpa. Agentes poderão editar qualquer caminho relativo no workspace.")
			return nil
		}

		ui.Success("Caminhos permitidos atualizados.")
		ui.Info(strings.Join(paths, ", "))
		return nil
	},
}

var configEditAddPathCmd = &cobra.Command{
	Use:   "add-path [path]",
	Short: "Adiciona um caminho permitido para edição",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		if err := config.AddAgentEditAllowedPath(args[0]); err != nil {
			return err
		}
		ui.Success("Caminho permitido adicionado com sucesso.")
		return nil
	},
}

var configEditClearPathsCmd = &cobra.Command{
	Use:   "clear-paths",
	Short: "Remove todas as restrições de caminhos permitidos",
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := config.ClearAgentEditAllowedPaths(); err != nil {
			return err
		}
		ui.Success("Restrições de caminhos removidas.")
		return nil
	},
}

var modelListCmd = &cobra.Command{
	Use:   "list",
	Short: "Lista modelos disponíveis no Gemini",
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx, cancel := context.WithTimeout(cmd.Context(), modelTimeout)
		defer cancel()

		cfg := config.Load()
		client, err := gemini.NewClient(cfg.GeminiAPIKey, cfg.GeminiModel)
		if err != nil {
			if errors.Is(err, gemini.ErrMissingAPIKey) {
				return fmt.Errorf("GEMINI_API_KEY não definida")
			}
			return err
		}

		models, err := client.ListModels(ctx)
		if err != nil {
			return err
		}

		ui.Header("Modelos Gemini disponíveis")
		rows := make([][]string, 0, len(models))
		count := 0
		for _, m := range models {
			if !modelShowAll && !m.SupportsGenerateContent() {
				continue
			}

			methods := ""
			if len(m.SupportedGenerationMethods) > 0 {
				methods = fmt.Sprintf(" | methods: %s", strings.Join(m.SupportedGenerationMethods, ","))
			}

			name := strings.TrimPrefix(m.Name, "models/")
			display := m.DisplayName
			if display == "" {
				display = "-"
			}
			generate := "não"
			if m.SupportsGenerateContent() {
				generate = "sim"
			}

			rows = append(rows, []string{name, display, generate, methods})
			count++
		}

		if count == 0 {
			if modelShowAll {
				ui.Warn("Nenhum modelo retornado pela API Gemini.")
			} else {
				ui.Warn("Nenhum modelo com suporte a generateContent. Use --all para listar todos.")
			}
			return nil
		}

		ui.Table([]string{"Modelo", "Display", "generateContent", "Métodos"}, rows)
		ui.Success(fmt.Sprintf("%d modelo(s) listado(s).", count))

		return nil
	},
}

var presetsInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Cria um conjunto inicial de agentes prontos para uso",
	RunE: func(_ *cobra.Command, _ []string) error {
		model := strings.TrimSpace(presetsModel)
		if model == "" {
			model = config.Load().GeminiModel
		}

		count, err := agentkit.SeedPresets(presetsForce, model)
		if err != nil {
			return err
		}
		if count == 0 {
			ui.Warn("Nenhum preset novo foi criado (já existiam). Use --force para sobrescrever.")
			return nil
		}
		ui.Info(fmt.Sprintf("Modelo aplicado aos presets: %s", model))
		ui.Success(fmt.Sprintf("Presets inicializados: %d agente(s).", count))
		return nil
	},
}

var presetsListCmd = &cobra.Command{
	Use:   "list",
	Short: "Lista os agentes predefinidos disponíveis",
	RunE: func(_ *cobra.Command, _ []string) error {
		model := strings.TrimSpace(presetsModel)
		if model == "" {
			model = config.Load().GeminiModel
		}

		presets := agentkit.BuiltinPresets(model)
		ui.Header("Presets disponíveis")

		rows := make([][]string, 0, len(presets))
		for _, p := range presets {
			tags := "-"
			if len(p.Tags) > 0 {
				tags = strings.Join(p.Tags, ",")
			}

			status := "não inicializado"
			if _, err := agentkit.LoadSpec(p.Name); err == nil {
				status = "inicializado"
			}

			rows = append(rows, []string{p.Name, p.Model, tags, status})
		}

		ui.Table([]string{"Preset", "Modelo", "Tags", "Status"}, rows)
		ui.Info("Use 'morpho presets init --force --model <modelo>' para reaplicar presets com outro modelo.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(agentCmd)
	rootCmd.AddCommand(presetsCmd)
	rootCmd.AddCommand(modelCmd)
	rootCmd.AddCommand(configCmd)

	agentCmd.AddCommand(agentListCmd)
	agentCmd.AddCommand(agentShowCmd)
	agentCmd.AddCommand(agentCreateCmd)
	agentCmd.AddCommand(agentEditCmd)
	agentCmd.AddCommand(agentSetModelCmd)
	agentCmd.AddCommand(agentOutputCmd)
	agentCmd.AddCommand(agentRunCmd)
	agentOutputCmd.AddCommand(agentOutputListCmd)
	agentOutputCmd.AddCommand(agentOutputShowCmd)
	agentOutputCmd.AddCommand(agentOutputLastCmd)
	rootCmd.AddCommand(globalViewCmd)

	presetsCmd.AddCommand(presetsInitCmd)
	presetsCmd.AddCommand(presetsListCmd)
	modelCmd.AddCommand(modelListCmd)
	modelCmd.AddCommand(modelSetCmd)
	modelCmd.AddCommand(modelSetAgentCmd)
	configCmd.AddCommand(configSetAPIKeyCmd)
	configCmd.AddCommand(configListModelsCmd)
	configCmd.AddCommand(configWhereCmd)
	configCmd.AddCommand(configEditCmd)
	configCmd.AddCommand(configMemoryCmd)
	configEditCmd.AddCommand(configEditShowCmd)
	configEditCmd.AddCommand(configEditSetModeCmd)
	configEditCmd.AddCommand(configEditSetPathsCmd)
	configEditCmd.AddCommand(configEditAddPathCmd)
	configEditCmd.AddCommand(configEditClearPathsCmd)
	configMemoryCmd.AddCommand(configMemoryShowCmd)
	configMemoryCmd.AddCommand(configMemorySetReadPolicyCmd)
	configMemoryCmd.AddCommand(configMemorySetTTLCmd)
	presetsInitCmd.Flags().BoolVar(&presetsForce, "force", false, "sobrescreve presets já existentes")
	presetsInitCmd.Flags().StringVar(&presetsModel, "model", "", "modelo aplicado aos presets (padrão: modelo global configurado)")
	presetsListCmd.Flags().StringVar(&presetsModel, "model", "", "modelo para pré-visualização dos presets (padrão: modelo global configurado)")
	modelListCmd.Flags().DurationVar(&modelTimeout, "timeout", 20*time.Second, "timeout para consulta de modelos")
	modelListCmd.Flags().BoolVar(&modelShowAll, "all", false, "lista todos os modelos retornados pela API")
	modelSetCmd.Flags().StringVar(&modelValue, "value", "", "modelo padrão")
	configSetAPIKeyCmd.Flags().StringVar(&configAPIKey, "key", "", "api key do Gemini")
	agentOutputListCmd.Flags().IntVar(&outputListLimit, "limit", 30, "limite de outputs retornados")

	agentCreateCmd.Flags().StringVar(&agentDescription, "description", "", "descrição curta do agente")
	agentCreateCmd.Flags().StringVar(&agentPrompt, "prompt", "", "prompt de sistema do agente")
	agentCreateCmd.Flags().StringVar(&agentModel, "model", "gemini-2.5-flash", "modelo Gemini")
	agentCreateCmd.Flags().StringVar(&agentTags, "tags", "", "tags separadas por vírgula")

	agentEditCmd.Flags().StringVar(&agentDescription, "description", "", "nova descrição")
	agentEditCmd.Flags().StringVar(&agentPrompt, "prompt", "", "novo prompt")
	agentEditCmd.Flags().StringVar(&agentModel, "model", "", "novo modelo")
	agentEditCmd.Flags().StringVar(&agentName, "name", "", "novo nome do agente")
	agentEditCmd.Flags().StringVar(&agentTags, "tags", "", "novas tags separadas por vírgula")

	agentRunCmd.Flags().DurationVar(&runTimeout, "timeout", 60*time.Second, "timeout por tentativa de execução")
	agentRunCmd.Flags().BoolVar(&runMockAI, "mock", false, "executa sem chamadas reais ao Gemini")
	agentRunCmd.Flags().IntVar(&queueRetry, "queue-retries", 3, "quantidade de retries para timeout/rate-limit")
	agentRunCmd.Flags().DurationVar(&queueDelay, "queue-delay", 2*time.Second, "espera base entre retries da fila")
	agentRunCmd.Flags().IntVar(&runContextEntries, "context-entries", 4, "quantidade de outputs de outros agentes a usar como contexto")
	agentRunCmd.Flags().IntVar(&runContextChars, "context-chars", 5000, "limite de caracteres de contexto compartilhado")
	agentRunCmd.Flags().BoolVar(&runNoContext, "no-shared-context", false, "desativa contexto de outputs de outros agentes")
	agentRunCmd.Flags().BoolVar(&runEditEnabled, "edit", false, "habilita fluxo de edição de arquivos via agente")
	agentRunCmd.Flags().StringVar(&runEditMode, "edit-mode", "", "override do modo de edição nesta execução (off|review|auto)")
	agentRunCmd.Flags().StringVar(&runEditPaths, "edit-paths", "", "override de caminhos permitidos (csv)")
	agentRunCmd.Flags().IntVar(&runEditMaxEdits, "edit-max", 8, "limite máximo de arquivos editados por execução")
	agentRunCmd.Flags().BoolVar(&runEditYes, "yes", false, "aprova automaticamente todas as edições em mode=review")
	agentRunCmd.Flags().BoolVar(&runRAGEnabled, "rag", false, "habilita recuperação semântica (RAG) nesta execução")
	agentRunCmd.Flags().BoolVar(&runNoRAG, "no-rag", false, "desabilita recuperação semântica (RAG) nesta execução")
	agentRunCmd.Flags().IntVar(&runRAGTopK, "rag-topk", 0, "override do top-k de memórias recuperadas")
	agentRunCmd.Flags().Float64Var(&runRAGMinScore, "rag-min-score", 0, "override da similaridade mínima (0-1)")

	configEditSetModeCmd.Flags().StringVar(&configEditMode, "mode", "", "modo de edição (off|review|auto)")
	configEditSetPathsCmd.Flags().StringVar(&configEditPaths, "paths", "", "caminhos permitidos separados por vírgula")
	configMemorySetReadPolicyCmd.Flags().StringVar(&configMemoryReadPolicy, "policy", "", "policy de leitura (self|shared)")
	configMemorySetTTLCmd.Flags().IntVar(&configMemoryTTLHours, "hours", 0, "ttl em horas")
}

func splitTags(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func promptApproval(message string) (bool, error) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("🦋: [INPUT] %s [y/N]: ", strings.TrimSpace(message))
		line, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				return false, nil
			}
			return false, err
		}

		answer := strings.ToLower(strings.TrimSpace(line))
		switch answer {
		case "y", "yes", "s", "sim":
			return true, nil
		case "", "n", "no", "nao", "não":
			return false, nil
		default:
			ui.Warn("Resposta inválida. Use y/s para sim ou n para não.")
		}
	}
}
