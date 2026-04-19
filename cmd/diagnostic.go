package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/vhwcm/Gopher/internal/agentkit"
	"github.com/vhwcm/Gopher/internal/config"
	"github.com/vhwcm/Gopher/internal/gemini"
	"github.com/vhwcm/Gopher/internal/ui"
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
	agentTags        string
	presetsForce     bool

	modelTimeout time.Duration
	modelShowAll bool
	modelValue   string

	outputListLimit int

	configAPIKey string
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
			ui.Warn("Nenhum agente encontrado. Use 'gopher presets init' para iniciar com agentes pré-selecionados.")
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
			agentModel = "gemini-2.0-flash"
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
		spec, err := agentkit.LoadSpec(args[0])
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
	Use:   "show [nome-agente] [arquivo]",
	Short: "Exibe o conteúdo de um output específico",
	Args:  cobra.ExactArgs(2),
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

		task := args[1]
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

		savedPath, err := agentkit.SaveAgentOutput(spec.Name, args[1], out)
		if err != nil {
			return err
		}

		ui.Panel(fmt.Sprintf("Resposta do agente '%s'", spec.Name), out)
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
	Short: "Define o modelo padrão do Gopher",
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
	Short: "Gerencia configurações locais do Gopher",
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
		count, err := agentkit.SeedPresets(presetsForce)
		if err != nil {
			return err
		}
		if count == 0 {
			ui.Warn("Nenhum preset novo foi criado (já existiam). Use --force para sobrescrever.")
			return nil
		}
		ui.Success(fmt.Sprintf("Presets inicializados: %d agente(s).", count))
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

	presetsCmd.AddCommand(presetsInitCmd)
	modelCmd.AddCommand(modelListCmd)
	modelCmd.AddCommand(modelSetCmd)
	modelCmd.AddCommand(modelSetAgentCmd)
	configCmd.AddCommand(configSetAPIKeyCmd)
	configCmd.AddCommand(configWhereCmd)
	presetsInitCmd.Flags().BoolVar(&presetsForce, "force", false, "sobrescreve presets já existentes")
	modelListCmd.Flags().DurationVar(&modelTimeout, "timeout", 20*time.Second, "timeout para consulta de modelos")
	modelListCmd.Flags().BoolVar(&modelShowAll, "all", false, "lista todos os modelos retornados pela API")
	modelSetCmd.Flags().StringVar(&modelValue, "value", "", "modelo padrão")
	configSetAPIKeyCmd.Flags().StringVar(&configAPIKey, "key", "", "api key do Gemini")
	agentOutputListCmd.Flags().IntVar(&outputListLimit, "limit", 30, "limite de outputs retornados")

	agentCreateCmd.Flags().StringVar(&agentDescription, "description", "", "descrição curta do agente")
	agentCreateCmd.Flags().StringVar(&agentPrompt, "prompt", "", "prompt de sistema do agente")
	agentCreateCmd.Flags().StringVar(&agentModel, "model", "gemini-2.0-flash", "modelo Gemini")
	agentCreateCmd.Flags().StringVar(&agentTags, "tags", "", "tags separadas por vírgula")

	agentEditCmd.Flags().StringVar(&agentDescription, "description", "", "nova descrição")
	agentEditCmd.Flags().StringVar(&agentPrompt, "prompt", "", "novo prompt")
	agentEditCmd.Flags().StringVar(&agentModel, "model", "", "novo modelo")
	agentEditCmd.Flags().StringVar(&agentTags, "tags", "", "novas tags separadas por vírgula")

	agentRunCmd.Flags().DurationVar(&runTimeout, "timeout", 60*time.Second, "timeout por tentativa de execução")
	agentRunCmd.Flags().BoolVar(&runMockAI, "mock", false, "executa sem chamadas reais ao Gemini")
	agentRunCmd.Flags().IntVar(&queueRetry, "queue-retries", 3, "quantidade de retries para timeout/rate-limit")
	agentRunCmd.Flags().DurationVar(&queueDelay, "queue-delay", 2*time.Second, "espera base entre retries da fila")
	agentRunCmd.Flags().IntVar(&runContextEntries, "context-entries", 4, "quantidade de outputs de outros agentes a usar como contexto")
	agentRunCmd.Flags().IntVar(&runContextChars, "context-chars", 5000, "limite de caracteres de contexto compartilhado")
	agentRunCmd.Flags().BoolVar(&runNoContext, "no-shared-context", false, "desativa contexto de outputs de outros agentes")
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
