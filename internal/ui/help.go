package ui

import (
	_ "embed"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

//go:embed morphoLogo.txt
var morphoArt string


var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true).
			MarginBottom(0)

	descStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			MarginBottom(1)

	usageHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("41")).
				Bold(true).
				MarginBottom(1).
				MarginTop(1)

	commandHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("43")).
				Bold(true).
				MarginBottom(1)

	commandNameStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("212")).
				Bold(true).
				Width(14)

	commandDescStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("250"))

	artStyle = lipgloss.NewStyle().
			MarginRight(3)

	logoBlueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))

	panelStyleHelp = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(1, 2).
			MarginLeft(2).
			MarginBottom(1)
)

func ShowCustomHelp(cmd *cobra.Command, args []string) {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w == 0 {
		w = 80 // Default reasonable width
	}

	title := titleStyle.Render("🦋 " + cmd.Short)
	desc := lipgloss.NewStyle().Width(w - 4).Foreground(lipgloss.Color("243")).Render(cmd.Long)

	art := artStyle.Render(colorizeLogo(renderTransparentLogo(morphoArt)))
	commandsSection := buildCommandsSection(cmd)

	var panelContent string
	// Se couber a arte + comandos + margens (8 caracteres de folga), mostra horizontal
	if w >= lipgloss.Width(art)+lipgloss.Width(commandsSection)+10 {
		panelContent = lipgloss.JoinHorizontal(lipgloss.Top, art, commandsSection)
	} else {
		// Se não couber horizontalmente, removemos a logo (morphoArt) conforme solicitado
		// e mostramos apenas a seção de comandos.
		panelContent = commandsSection
	}

	layout := panelStyleHelp.Render(panelContent)

	fmt.Println()
	fmt.Println("  " + title)
	fmt.Println("  " + desc)
	fmt.Println(layout)
	fmt.Println()
}

func renderTransparentLogo(raw string) string {
	normalized := strings.ReplaceAll(raw, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")

	minIndent := -1
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}

	if minIndent > 0 {
		for i, line := range lines {
			if len(line) >= minIndent {
				lines[i] = line[minIndent:]
			}
		}
	}

	for i, line := range lines {
		line = strings.ReplaceAll(line, ".", " ")
		lines[i] = strings.TrimRight(line, " ")
	}

	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}

func colorizeLogo(raw string) string {
	var out strings.Builder

	for _, r := range raw {
		switch {
		case r == '\n':
			out.WriteRune(r)
		case r == '.':
			out.WriteRune(' ')
		case r == ' ':
			out.WriteRune(' ')
		default:
			out.WriteString(logoBlueStyle.Render(string(r)))
		}
	}

	return out.String()
}

func buildCommandsSection(cmd *cobra.Command) string {
	var builder strings.Builder

	// Usage
	builder.WriteString(usageHeaderStyle.Render("Usage:"))
	builder.WriteString("\n")
	builder.WriteString(fmt.Sprintf("  %s [command]\n\n", cmd.Use))

	builder.WriteString(commandHeaderStyle.Render("Available Commands:"))
	builder.WriteString("\n")

	for _, c := range cmd.Commands() {
		if c.IsAvailableCommand() || c.Name() == "help" {
			name := commandNameStyle.Render(c.Name())
			desc := commandDescStyle.Render(ShortOrUsage(c))
			builder.WriteString(fmt.Sprintf("  %s %s\n", name, desc))
		}
	}

	builder.WriteString("\n")
	builder.WriteString(commandHeaderStyle.Render("Flags:"))
	builder.WriteString("\n")
	builder.WriteString("  " + commandNameStyle.Render("-h, --help") + " " + commandDescStyle.Render("help for " + cmd.Use) + "\n")

	return builder.String()
}

func ShortOrUsage(c *cobra.Command) string {
	if c.Short != "" {
		return c.Short
	}
	return c.Use
}
