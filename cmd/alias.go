package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vhwcm/Morpho/internal/ui"
)

var aliasCmd = &cobra.Command{
	Use:   "alias [nome] [comando]",
	Short: "Adiciona um alias ao seu ~/.bashrc",
	Long:  "Adiciona uma linha de alias ao arquivo ~/.bashrc do usuário para facilitar o uso de comandos frequentes.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := strings.TrimSpace(args[0])
		value := strings.TrimSpace(args[1])

		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("não foi possível encontrar o diretório home: %w", err)
		}

		bashrcPath := filepath.Join(home, ".bashrc")
		
		// Verificar se o alias já existe para evitar duplicatas simples
		content, err := os.ReadFile(bashrcPath)
		if err == nil {
			if strings.Contains(string(content), fmt.Sprintf("alias %s=", name)) {
				ui.Warn(fmt.Sprintf("O alias '%s' já parece existir no seu .bashrc. Adicionando assim mesmo.", name))
			}
		}

		aliasLine := fmt.Sprintf("\nalias %s='%s' # adicionado pelo Morpho\n", name, value)

		f, err := os.OpenFile(bashrcPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return fmt.Errorf("erro ao abrir .bashrc: %w", err)
		}
		defer f.Close()

		if _, err := f.WriteString(aliasLine); err != nil {
			return fmt.Errorf("erro ao escrever no .bashrc: %w", err)
		}

		ui.Header("Gerenciamento de Alias")
		ui.Success(fmt.Sprintf("Alias '%s' adicionado com sucesso ao seu .bashrc", name))
		ui.Info(fmt.Sprintf("Linha adicionada: alias %s='%s'", name, value))
		ui.Warn("Importante: Para que o alias funcione nesta sessão, execute:")
		ui.Panel("Comando", fmt.Sprintf("source %s", bashrcPath))

		return nil
	},
}

func init() {
	rootCmd.AddCommand(aliasCmd)
}
