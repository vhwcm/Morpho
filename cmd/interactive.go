package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

type fieldSpec struct {
	Label       string
	Placeholder string
}

type menuItem struct {
	Title       string
	Description string
	Fields      []fieldSpec
	BuildArgs   func(values map[string]string) []string
	Exit        bool
}

type viewState int

const (
	stateMenu viewState = iota
	stateForm
	stateRunning
	stateOutput
)

type commandFinishedMsg struct {
	Output string
	Err    error
	Args   []string
}

type interactiveModel struct {
	items []menuItem

	state       viewState
	selected    int
	activeItem  int
	fieldIndex  int
	fieldValues map[string]string
	input       textinput.Model

	lastArgs   []string
	lastOutput string
	lastError  error
}

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	hintStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	menuStyle  = lipgloss.NewStyle().Padding(1, 2)
	boxStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("63")).Padding(1)

	selectedButtonStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("62")).Bold(true).Padding(0, 1)
	buttonStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Padding(0, 1)
)

var interactiveCmd = &cobra.Command{
	Use:     "interactive",
	Aliases: []string{"i"},
	Short:   "Inicia modo iterativo com navegação por botões",
	RunE: func(cmd *cobra.Command, _ []string) error {
		items := buildInteractiveItems()
		m := interactiveModel{
			items:      items,
			state:      stateMenu,
			activeItem: -1,
		}

		p := tea.NewProgram(m, tea.WithAltScreen())
		_, err := p.Run()
		if err != nil {
			return err
		}

		_ = cmd
		return nil
	},
}

func init() {
	rootCmd.AddCommand(interactiveCmd)
}

func buildInteractiveItems() []menuItem {
	return []menuItem{
		{
			Title:       "Listar agentes",
			Description: "Exibe todos os agentes cadastrados",
			BuildArgs: func(_ map[string]string) []string {
				return []string{"agent", "list"}
			},
		},
		{
			Title:       "Executar agente",
			Description: "Roda um agente com uma tarefa",
			Fields: []fieldSpec{
				{Label: "nome_agente", Placeholder: "ex: backend-go"},
				{Label: "tarefa", Placeholder: "ex: Criar plano para autenticação JWT"},
			},
			BuildArgs: func(v map[string]string) []string {
				return []string{"agent", "run", strings.TrimSpace(v["nome_agente"]), strings.TrimSpace(v["tarefa"])}
			},
		},
		{
			Title:       "Listar outputs",
			Description: "Mostra outputs de todos os agentes",
			BuildArgs: func(_ map[string]string) []string {
				return []string{"agent", "output", "list"}
			},
		},
		{
			Title:       "Visualizar output",
			Description: "Exibe conteúdo de um output específico",
			Fields: []fieldSpec{
				{Label: "nome_agente", Placeholder: "ex: backend-go"},
				{Label: "arquivo", Placeholder: "ex: 20260419-101500-task.md"},
			},
			BuildArgs: func(v map[string]string) []string {
				return []string{"agent", "output", "show", strings.TrimSpace(v["nome_agente"]), strings.TrimSpace(v["arquivo"])}
			},
		},
		{
			Title:       "Inicializar presets",
			Description: "Cria agentes pré-selecionados",
			BuildArgs: func(_ map[string]string) []string {
				return []string{"presets", "init"}
			},
		},
		{
			Title:       "Listar modelos Gemini",
			Description: "Consulta modelos disponíveis",
			BuildArgs: func(_ map[string]string) []string {
				return []string{"model", "list"}
			},
		},
		{
			Title:       "Setar API key",
			Description: "Salva chave fora do repositório",
			Fields:      []fieldSpec{{Label: "api_key", Placeholder: "cole a chave Gemini"}},
			BuildArgs: func(v map[string]string) []string {
				return []string{"config", "set-api-key", strings.TrimSpace(v["api_key"])}
			},
		},
		{
			Title:       "Sair",
			Description: "Encerrar modo iterativo",
			Exit:        true,
		},
	}
}

func (m interactiveModel) Init() tea.Cmd { return nil }

func (m interactiveModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case stateMenu:
			return m.updateMenuKeys(msg)
		case stateForm:
			return m.updateFormKeys(msg)
		case stateOutput:
			return m.updateOutputKeys(msg)
		case stateRunning:
			if msg.String() == "ctrl+c" || msg.String() == "q" {
				return m, tea.Quit
			}
		}

	case commandFinishedMsg:
		m.lastOutput = msg.Output
		m.lastError = msg.Err
		m.lastArgs = msg.Args
		m.state = stateOutput
		return m, nil
	}

	if m.state == stateForm {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m interactiveModel) updateMenuKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.selected > 0 {
			m.selected--
		}
		return m, nil
	case "down", "j":
		if m.selected < len(m.items)-1 {
			m.selected++
		}
		return m, nil
	case "enter":
		item := m.items[m.selected]
		if item.Exit {
			return m, tea.Quit
		}

		if len(item.Fields) == 0 {
			args := item.BuildArgs(nil)
			m.state = stateRunning
			return m, runCommandCmd(args)
		}

		m.activeItem = m.selected
		m.fieldIndex = 0
		m.fieldValues = map[string]string{}
		m.input = textinput.New()
		m.input.Placeholder = item.Fields[0].Placeholder
		m.input.Prompt = item.Fields[0].Label + ": "
		m.input.Focus()
		m.state = stateForm
		return m, nil
	}

	return m, nil
}

func (m interactiveModel) updateFormKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	item := m.items[m.activeItem]
	switch msg.String() {
	case "esc":
		m.state = stateMenu
		return m, nil
	case "enter", "tab":
		field := item.Fields[m.fieldIndex]
		m.fieldValues[field.Label] = strings.TrimSpace(m.input.Value())

		if m.fieldIndex == len(item.Fields)-1 {
			args := item.BuildArgs(m.fieldValues)
			m.state = stateRunning
			return m, runCommandCmd(args)
		}

		m.fieldIndex++
		m.input.SetValue("")
		m.input.Placeholder = item.Fields[m.fieldIndex].Placeholder
		m.input.Prompt = item.Fields[m.fieldIndex].Label + ": "
		m.input.Focus()
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m interactiveModel) updateOutputKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "b", "esc":
		m.state = stateMenu
		return m, nil
	case "r":
		if len(m.lastArgs) > 0 {
			m.state = stateRunning
			return m, runCommandCmd(m.lastArgs)
		}
	}
	return m, nil
}

func runCommandCmd(args []string) tea.Cmd {
	return func() tea.Msg {
		out, err := runSelf(args)
		return commandFinishedMsg{Output: out, Err: err, Args: args}
	}
}

func runSelf(args []string) (string, error) {
	cmd := exec.Command(os.Args[0], args...)
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("comando falhou (%s): %w", strings.Join(args, " "), err)
	}
	return string(output), nil
}

func (m interactiveModel) View() string {
	switch m.state {
	case stateMenu:
		return m.renderMenuView()
	case stateForm:
		return m.renderFormView()
	case stateRunning:
		return boxStyle.Render("🦋 Executando comando...\n\nAguarde a conclusão.")
	case stateOutput:
		return m.renderOutputView()
	default:
		return ""
	}
}

func (m interactiveModel) renderMenuView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("🦋 Morpho • Modo Iterativo"))
	b.WriteString("\n")
	b.WriteString(hintStyle.Render("Use ↑/↓ para navegar, Enter para executar, q para sair."))
	b.WriteString("\n\n")

	for i, item := range m.items {
		prefix := "  "
		style := buttonStyle
		if i == m.selected {
			prefix = "▶ "
			style = selectedButtonStyle
		}
		line := prefix + item.Title + " — " + item.Description
		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}

	return menuStyle.Render(b.String())
}

func (m interactiveModel) renderFormView() string {
	item := m.items[m.activeItem]
	field := item.Fields[m.fieldIndex]
	title := titleStyle.Render("🦋 Preencha os campos")
	hint := hintStyle.Render("Enter/Tab para próximo campo, Esc para cancelar")
	body := fmt.Sprintf("Ação: %s\nCampo atual: %s\n\n%s", item.Title, field.Label, m.input.View())
	return boxStyle.Render(title + "\n" + hint + "\n\n" + body)
}

func (m interactiveModel) renderOutputView() string {
	title := titleStyle.Render("🦋 Resultado do comando")
	args := strings.Join(m.lastArgs, " ")
	status := "✅ sucesso"
	if m.lastError != nil {
		status = "❌ erro: " + m.lastError.Error()
	}

	output := strings.TrimSpace(m.lastOutput)
	if output == "" {
		output = "(sem output)"
	}

	footer := hintStyle.Render("Pressione b para voltar ao menu, r para executar novamente, q para sair")
	return boxStyle.Render(fmt.Sprintf("%s\nComando: %s\nStatus: %s\n\n%s\n\n%s", title, args, status, output, footer))
}
