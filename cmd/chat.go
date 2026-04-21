package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/vhwcm/Morpho/internal/agentkit"
	"github.com/vhwcm/Morpho/internal/config"
	"github.com/vhwcm/Morpho/internal/gemini"
	"github.com/vhwcm/Morpho/internal/logger"
	"github.com/vhwcm/Morpho/internal/ui"
)

var (
	userMsgStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)

	morphoMsgStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true)

	chatBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	toolCallBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.DoubleBorder()).
				BorderForeground(lipgloss.Color("214")).
				Padding(1, 2).
				Margin(1, 0)

	approveButtonStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("230")).
				Background(lipgloss.Color("28")).
				Bold(true).
				Padding(0, 1)

	denyButtonStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("160")).
			Bold(true).
			Padding(0, 1)

	unselectedButtonStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("250")).
				Background(lipgloss.Color("238")).
				Padding(0, 1)
)

type toolCallConfirmedMsg struct {
	call gemini.FunctionCall
}

type toolCallCancelledMsg struct{}

type chatModel struct {
	agent            agentkit.Spec
	history          []agentkit.ChatMessage
	viewport         viewport.Model
	textInput        textinput.Model
	spinner          spinner.Model
	loading          bool
	err              error
	width, height    int
	ai               agentkit.AIClient
	proposedToolCall *gemini.FunctionCall
	approvalReason   string
	approvalIndex    int
}

type aiResponseMsg struct {
	content       string
	functionCalls []gemini.FunctionCall
	err           error
}

var chatCmd = &cobra.Command{
	Use:   "chat [agente]",
	Short: "Inicia uma conversa com um agente",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentName := "morpho"
		if len(args) > 0 {
			agentName = args[0]
		}

		spec, err := agentkit.LoadSpec(agentName)
		if err != nil {
			// Se não achar o agente, tenta o primeiro disponível ou cria um temporário
			specs, _ := agentkit.ListSpecs()
			if len(specs) > 0 {
				spec = specs[0]
			} else {
				spec = agentkit.Spec{
					Name:         "morpho",
					SystemPrompt: "Você é o Morpho, um assistente prestativo.",
					Model:        "gemini-2.5-flash",
				}
			}
		}

		apiKey := config.GetGeminiAPIKey()
		if apiKey == "" {
			return fmt.Errorf("GEMINI_API_KEY não configurada. Use 'morpho config set-api-key' primeiro")
		}

		if spec.Name == "morpho" && len(spec.Tools) == 0 {
			for _, p := range agentkit.BuiltinPresets(spec.Model) {
				if p.Name == "morpho" {
					spec.Tools = p.Tools
					if strings.TrimSpace(spec.SystemPrompt) == "" {
						spec.SystemPrompt = p.SystemPrompt
					}
					break
				}
			}
		}

		ai, err := gemini.NewClient(apiKey, spec.Model)
		if err != nil {
			return err
		}

		ti := textinput.New()
		ti.Placeholder = "Diga algo para o Morpho..."
		ti.Focus()
		ti.CharLimit = 1000
		ti.Width = 50

		s := spinner.New()
		s.Spinner = spinner.Dot
		s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

		m := chatModel{
			agent:     spec,
			textInput: ti,
			spinner:   s,
			ai:        ai,
			viewport:  viewport.New(80, 20),
		}
		m.viewport.SetContent(m.renderHistory())

		p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
		_, err = p.Run()
		return err
	},
}

func init() {
	rootCmd.AddCommand(chatCmd)
}

func (m chatModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick)
}

func (m chatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
		sCmd  tea.Cmd
	)

	m.textInput, tiCmd = m.textInput.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)
	m.spinner, sCmd = m.spinner.Update(msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - 10
		m.textInput.Width = msg.Width - 10
		m.viewport.SetContent(m.renderHistory())

	case tea.KeyMsg:
		if m.proposedToolCall != nil {
			switch msg.String() {
			case "left", "h", "shift+tab":
				m.approvalIndex = 0
				return m, nil
			case "right", "l", "tab":
				m.approvalIndex = 1
				return m, nil
			case "y", "s":
				m.approvalIndex = 0
				fallthrough
			case "enter":
				if m.approvalIndex == 1 {
					m.proposedToolCall = nil
					m.history = append(m.history, agentkit.ChatMessage{Role: "user", Content: "Comando cancelado pelo usuário."})
					m.viewport.SetContent(m.renderHistory())
					m.viewport.GotoBottom()
					m.loading = true
					return m, m.sendToAI("")
				}
				call := *m.proposedToolCall
				m.proposedToolCall = nil
				m.approvalReason = ""
				m.history = append(m.history, agentkit.ChatMessage{Role: "user", Content: "Aprovado. Pode executar o comando."})
				m.loading = true
				return m, func() tea.Msg {
					return toolCallConfirmedMsg{call: call}
				}
			case "n":
				m.approvalIndex = 1
				m.proposedToolCall = nil
				m.approvalReason = ""
				m.history = append(m.history, agentkit.ChatMessage{Role: "user", Content: "Comando cancelado pelo usuário."})
				m.viewport.SetContent(m.renderHistory())
				m.viewport.GotoBottom()
				m.loading = true
				return m, m.sendToAI("")
			}
		}

		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			if m.loading || strings.TrimSpace(m.textInput.Value()) == "" {
				return m, nil
			}

			input := m.textInput.Value()
			m.history = append(m.history, agentkit.ChatMessage{Role: "user", Content: input})
			m.textInput.SetValue("")
			m.loading = true
			m.viewport.SetContent(m.renderHistory())
			m.viewport.GotoBottom()

			return m, m.sendToAI(input)
		}

	case aiResponseMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.history = append(m.history, agentkit.ChatMessage{Role: "model", Content: "❌ Erro: " + msg.err.Error()})
		} else {
			if msg.content != "" {
				m.history = append(m.history, agentkit.ChatMessage{Role: "model", Content: msg.content})
			}
			if len(msg.functionCalls) > 0 {
				call := msg.functionCalls[0]
				requiresApproval, reason := requiresApproval(call)
				if requiresApproval {
					m.proposedToolCall = &call
					m.approvalReason = reason
					m.approvalIndex = 0
					m.history = append(m.history, agentkit.ChatMessage{Role: "model", Content: fmt.Sprintf("Proposta de execução: %s", call.Name)})
				} else {
					m.loading = true
					m.history = append(m.history, agentkit.ChatMessage{Role: "model", Content: fmt.Sprintf("Execução automática segura: %s", formatToolCall(call))})
					m.viewport.SetContent(m.renderHistory())
					m.viewport.GotoBottom()
					return m, func() tea.Msg {
						return toolCallConfirmedMsg{call: call}
					}
				}
			}
		}
		m.viewport.SetContent(m.renderHistory())
		m.viewport.GotoBottom()

	case toolCallConfirmedMsg:
		return m, m.handleToolCall(msg.call)
	}

	return m, tea.Batch(tiCmd, vpCmd, sCmd)
}

func (m chatModel) historyToTask() string {
	var b strings.Builder
	for _, msg := range m.history {
		role := "Usuário"
		if msg.Role == "model" {
			role = "Assistente"
		}
		b.WriteString(fmt.Sprintf("%s: %s\n", role, msg.Content))
	}
	return b.String()
}

func (m chatModel) handleToolCall(call gemini.FunctionCall) tea.Cmd {
	return func() tea.Msg {
		output, err := m.executeToolCall(call)
		if err != nil {
			output = "Erro: " + err.Error()
		}

		history := append([]agentkit.ChatMessage{}, m.history...)
		history = append(history, agentkit.ChatMessage{
			Role:         "function",
			Content:      output,
			FunctionName: call.Name,
		})

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		res, err := agentkit.RunQueued(ctx, agentkit.QueueRequest{
			AI:             m.ai,
			Spec:           m.agent,
			Task:           "",
			History:        history,
			AttemptTimeout: 60 * time.Second,
		})
		if err != nil {
			return aiResponseMsg{err: err}
		}

		msg := res.Message
		if strings.TrimSpace(msg) == "" {
			msg = "Comando executado."
		}

		return aiResponseMsg{content: msg, functionCalls: res.FunctionCalls}
	}
}

func (m chatModel) executeToolCall(call gemini.FunctionCall) (string, error) {
	switch call.Name {
	case "run_command":
		argsRaw, ok := call.Args["args"].([]interface{})
		if !ok {
			return "", fmt.Errorf("argumentos inválidos para run_command")
		}
		args := make([]string, len(argsRaw))
		for i, v := range argsRaw {
			args[i] = fmt.Sprintf("%v", v)
		}
		return m.executeActualCommand(args)

	case "run_shell_command":
		cmdRaw, ok := call.Args["command"]
		if !ok {
			return "", fmt.Errorf("argumento 'command' obrigatório para run_shell_command")
		}
		command := strings.TrimSpace(fmt.Sprintf("%v", cmdRaw))
		if command == "" {
			return "", fmt.Errorf("comando shell vazio")
		}

		workingDir := ""
		if wdRaw, ok := call.Args["working_dir"]; ok {
			workingDir = strings.TrimSpace(fmt.Sprintf("%v", wdRaw))
		}

		timeoutSeconds := 30
		if toRaw, ok := call.Args["timeout_seconds"]; ok {
			s := strings.TrimSpace(fmt.Sprintf("%v", toRaw))
			if i, err := strconv.Atoi(strings.Split(s, ".")[0]); err == nil && i > 0 {
				timeoutSeconds = i
			}
		}

		return executeShell(command, workingDir, time.Duration(timeoutSeconds)*time.Second)

	default:
		return "", fmt.Errorf("ferramenta desconhecida: %s", call.Name)
	}
}

func (m chatModel) executeActualCommand(args []string) (string, error) {
	logger.Info("Executando comando solicitado pelo agente", map[string]interface{}{"args": args})

	// Se for chat, não podemos rodar recursivamente no mesmo processo facilmente sem quebrar TTY
	if len(args) > 0 && args[0] == "chat" {
		return "Erro: Não é possível iniciar um novo chat de dentro deste chat. Use outros comandos como 'agent create', 'agent list', etc.", nil
	}

	cmd := exec.Command(os.Args[0], args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return truncateOutput(string(output)), fmt.Errorf("falha ao executar: %w", err)
	}
	return truncateOutput(string(output)), nil
}

func executeShell(command, workingDir string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-lc", command)
	if strings.TrimSpace(workingDir) != "" {
		if !filepath.IsAbs(workingDir) {
			if wd, err := os.Getwd(); err == nil {
				workingDir = filepath.Join(wd, workingDir)
			}
		}
		cmd.Dir = workingDir
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}

	if err := cmd.Start(); err != nil {
		return "", err
	}

	outBytes, _ := io.ReadAll(stdout)
	errBytes, _ := io.ReadAll(stderr)
	waitErr := cmd.Wait()

	out := strings.TrimSpace(string(outBytes))
	errOut := strings.TrimSpace(string(errBytes))
	joined := out
	if errOut != "" {
		if joined != "" {
			joined += "\n"
		}
		joined += errOut
	}

	if ctx.Err() == context.DeadlineExceeded {
		return truncateOutput(joined), fmt.Errorf("comando excedeu timeout de %s", timeout)
	}
	if waitErr != nil {
		return truncateOutput(joined), waitErr
	}
	if strings.TrimSpace(joined) == "" {
		joined = "(sem saída)"
	}
	return truncateOutput(joined), nil
}

func truncateOutput(s string) string {
	const max = 20000
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n... [output truncado]"
}

func (m chatModel) sendToAI(input string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		logger.Debug("Iniciando sendToAI no chat", map[string]interface{}{"agent": m.agent.Name})

		res, err := agentkit.RunQueued(ctx, agentkit.QueueRequest{
			AI:             m.ai,
			Spec:           m.agent,
			Task:           "", // Não usado quando passamos history manualmente
			History:        m.history,
			AttemptTimeout: 60 * time.Second,
		})

		if err != nil {
			return aiResponseMsg{content: "", err: err}
		}

		return aiResponseMsg{
			content:       res.Message,
			functionCalls: res.FunctionCalls,
			err:           nil,
		}
	}
}

func (m chatModel) renderHistory() string {
	var b strings.Builder
	for _, msg := range m.history {
		if msg.Role == "user" {
			b.WriteString(userMsgStyle.Render("Você: ") + msg.Content + "\n\n")
		} else {
			b.WriteString(morphoMsgStyle.Render("🦋 Morpho: ") + msg.Content + "\n\n")
		}
	}
	return b.String()
}

func (m chatModel) View() string {
	if m.width == 0 {
		return "Carregando..."
	}

	header := ui.MessagePrefix + m.agent.Name + " Chat"
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63")).Render(header)

	content := m.viewport.View()

	loading := ""
	if m.loading {
		loading = "\n" + m.spinner.View() + " Morpho está pensando..."
	}

	input := chatBoxStyle.Width(m.width - 4).Render(m.textInput.View())
	help := helpStyle.Render(" esc/ctrl+c: sair • enter: enviar")

	toolConfirm := ""
	if m.proposedToolCall != nil {
		approve := unselectedButtonStyle.Render(" Aprovar ")
		deny := unselectedButtonStyle.Render(" Negar ")
		if m.approvalIndex == 0 {
			approve = approveButtonStyle.Render(" Aprovar ")
		} else {
			deny = denyButtonStyle.Render(" Negar ")
		}

		reason := ""
		if strings.TrimSpace(m.approvalReason) != "" {
			reason = "\nMotivo: " + m.approvalReason + "\n"
		}

		toolConfirm = "\n" + toolCallBoxStyle.Render(
			fmt.Sprintf("⚙️ Comando proposto: %s%s\n%s  %s\n\n(Use ←/→ e Enter, ou y/n)", formatToolCall(*m.proposedToolCall), reason, approve, deny),
		)
	}

	return fmt.Sprintf("\n %s\n\n %s\n%s%s\n\n %s\n %s", title, content, loading, toolConfirm, input, help)
}

func formatToolCall(call gemini.FunctionCall) string {
	if call.Name == "run_shell_command" {
		command := strings.TrimSpace(fmt.Sprintf("%v", call.Args["command"]))
		wd := strings.TrimSpace(fmt.Sprintf("%v", call.Args["working_dir"]))
		if wd != "" && wd != "%!v(<nil>)" {
			return fmt.Sprintf("[shell @ %s] %s", wd, command)
		}
		return "[shell] " + command
	}

	argsRaw, ok := call.Args["args"].([]interface{})
	if !ok {
		return call.Name
	}
	parts := make([]string, 0, len(argsRaw)+1)
	parts = append(parts, "morpho")
	for _, a := range argsRaw {
		parts = append(parts, fmt.Sprintf("%v", a))
	}
	return strings.Join(parts, " ")
}

func requiresApproval(call gemini.FunctionCall) (bool, string) {
	switch call.Name {
	case "run_shell_command":
		command := strings.TrimSpace(fmt.Sprintf("%v", call.Args["command"]))
		if command == "" {
			return true, "comando shell vazio ou inválido"
		}
		if isReadOnlyShellCommand(command) {
			return false, "comando de leitura"
		}
		return true, "comando shell potencialmente mutável"

	case "run_command":
		argsRaw, ok := call.Args["args"].([]interface{})
		if !ok || len(argsRaw) == 0 {
			return true, "comando CLI inválido"
		}
		args := make([]string, 0, len(argsRaw))
		for _, a := range argsRaw {
			args = append(args, strings.TrimSpace(fmt.Sprintf("%v", a)))
		}
		if isReadOnlyMorphoCommand(args) {
			return false, "comando de consulta"
		}
		return true, "comando CLI que altera estado"

	default:
		return true, "ferramenta não reconhecida"
	}
}

func isReadOnlyShellCommand(command string) bool {
	c := strings.ToLower(strings.TrimSpace(command))
	if c == "" {
		return false
	}

	unsafeMarkers := []string{" rm ", " mv ", " cp ", " chmod ", " chown ", " sed -i", " tee ", " >", " >>", "| sh", "| bash", "&&", ";"}
	for _, m := range unsafeMarkers {
		if strings.Contains(" "+c+" ", m) {
			return false
		}
	}

	readOnlyPrefixes := []string{
		"ls", "pwd", "whoami", "id", "date", "uname", "env", "printenv",
		"cat ", "head ", "tail ", "grep ", "find ",
		"git status", "git branch", "git log", "git diff",
		"go test", "go list", "morpho model list", "morpho agent list", "morpho agent show",
	}
	for _, p := range readOnlyPrefixes {
		if c == strings.TrimSpace(p) || strings.HasPrefix(c, p) {
			return true
		}
	}
	return false
}

func isReadOnlyMorphoCommand(args []string) bool {
	if len(args) == 0 {
		return false
	}

	join := strings.ToLower(strings.Join(args, " "))
	if strings.HasPrefix(join, "agent list") ||
		strings.HasPrefix(join, "agent show") ||
		strings.HasPrefix(join, "agent output list") ||
		strings.HasPrefix(join, "agent output show") ||
		strings.HasPrefix(join, "agent output last") ||
		strings.HasPrefix(join, "status") ||
		strings.HasPrefix(join, "worktree") ||
		strings.HasPrefix(join, "model list") ||
		strings.HasPrefix(join, "config where") ||
		strings.HasPrefix(join, "config list-models") ||
		strings.HasPrefix(join, "config edit show") ||
		strings.HasPrefix(join, "config memory show") ||
		strings.HasPrefix(join, "agent memory status") ||
		strings.HasPrefix(join, "agent memory search") {
		return true
	}

	return false
}
