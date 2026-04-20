package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const MessagePrefix = "🦋: "

var (
	prefixStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("62")).
			Padding(0, 1).
			Bold(true)

	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("117")).Bold(true)
	warnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(0, 1)

	tableHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230"))
	tableCellStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
)

func Println(message string) {
	printWithTag("", message)
}

func Header(title string) {
	fmt.Println(headerStyle.Render("🦋 Morpho • " + strings.TrimSpace(title)))
}

func Info(message string) {
	printWithTag(infoStyle.Render("INFO"), message)
}

func Success(message string) {
	printWithTag(successStyle.Render("OK"), message)
}

func Warn(message string) {
	printWithTag(warnStyle.Render("WARN"), message)
}

func ErrorToStderr(message string) {
	line := prefixStyle.Render(MessagePrefix) + "[" + errorStyle.Render("ERROR") + "] " + strings.TrimSpace(message)
	fmt.Fprintln(os.Stderr, line)
}

func Panel(title, body string) {
	header := infoStyle.Render(strings.TrimSpace(title))
	content := strings.TrimSpace(body)
	fmt.Println(panelStyle.Render(header + "\n\n" + content))
}

func Table(headers []string, rows [][]string) {
	if len(headers) == 0 {
		return
	}

	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = lipgloss.Width(h)
	}

	for _, row := range rows {
		for i := 0; i < len(headers) && i < len(row); i++ {
			if w := lipgloss.Width(row[i]); w > widths[i] {
				widths[i] = w
			}
		}
	}

	line := func(left, mid, right string) string {
		parts := make([]string, len(widths))
		for i, w := range widths {
			parts[i] = strings.Repeat("─", w+2)
		}
		return left + strings.Join(parts, mid) + right
	}

	fmt.Println(line("┌", "┬", "┐"))
	fmt.Println(renderRow(headers, widths, tableHeaderStyle))
	fmt.Println(line("├", "┼", "┤"))
	for _, row := range rows {
		cells := make([]string, len(headers))
		for i := 0; i < len(headers); i++ {
			if i < len(row) {
				cells[i] = row[i]
			}
		}
		fmt.Println(renderRow(cells, widths, tableCellStyle))
	}
	fmt.Println(line("└", "┴", "┘"))
}

func renderRow(cells []string, widths []int, style lipgloss.Style) string {
	parts := make([]string, len(widths))
	for i := range widths {
		value := ""
		if i < len(cells) {
			value = cells[i]
		}
		padded := padRight(value, widths[i])
		parts[i] = " " + style.Render(padded) + " "
	}
	return "│" + strings.Join(parts, "│") + "│"
}

func padRight(value string, width int) string {
	current := lipgloss.Width(value)
	if current >= width {
		return value
	}
	return value + strings.Repeat(" ", width-current)
}

func printWithTag(tag, message string) {
	msg := strings.TrimSpace(message)
	prefix := prefixStyle.Render(MessagePrefix)
	if tag == "" {
		fmt.Println(prefix + msg)
		return
	}
	fmt.Println(prefix + "[" + tag + "] " + msg)
}
