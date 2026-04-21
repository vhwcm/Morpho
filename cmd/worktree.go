package cmd

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/vhwcm/Morpho/internal/ui"
)

var (
	dirStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("63")).Bold(true)
	fileStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	treeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

type node struct {
	name     string
	isDir    bool
	children map[string]*node
}

func newNode(name string, isDir bool) *node {
	return &node{
		name:     name,
		isDir:    isDir,
		children: make(map[string]*node),
	}
}

var worktreeCmd = &cobra.Command{
	Use:   "worktree",
	Short: "Mostra a árvore de arquivos do repositório Git atual",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Verificar se é um repositório git
		if !isGitRepo() {
			ui.Warn("Este diretório não é um repositório Git.")
			return nil
		}

		// Obter lista de arquivos respeitando .gitignore
		files, err := getGitFiles()
		if err != nil {
			return fmt.Errorf("falha ao obter arquivos do git: %w", err)
		}

		// Construir a árvore de pastas
		root := newNode(".", true)
		for _, f := range files {
			parts := strings.Split(f, "/")
			// Se houver mais de uma parte, é um arquivo em uma pasta ou uma subpasta
			// Queremos as pastas.
			current := root
			for i := 0; i < len(parts)-1; i++ {
				part := parts[i]
				if _, ok := current.children[part]; !ok {
					current.children[part] = newNode(part, true)
				}
				current = current.children[part]
			}
		}

		ui.Header("Worktree de Pastas (Git)")
		fmt.Println(".")
		renderTree(root, "")

		return nil
	},
}

func isGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func getGitFiles() ([]string, error) {
	cmd := exec.Command("git", "ls-files", "--cached", "--others", "--exclude-standard")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	return lines, nil
}

func renderTree(n *node, prefix string) {
	keys := make([]string, 0, len(n.children))
	for k := range n.children {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for i, key := range keys {
		child := n.children[key]
		isLast := i == len(keys)-1

		connector := "├── "
		newPrefix := prefix + "│   "
		if isLast {
			connector = "└── "
			newPrefix = prefix + "    "
		}

		name := child.name
		if child.isDir || len(child.children) > 0 {
			name = dirStyle.Render(name + "/")
		} else {
			name = fileStyle.Render(name)
		}

		fmt.Printf("%s%s%s\n", treeStyle.Render(prefix), treeStyle.Render(connector), name)
		renderTree(child, newPrefix)
	}
}

func init() {
	rootCmd.AddCommand(worktreeCmd)
}
