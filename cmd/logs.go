package cmd

import (
	"bufio"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vhwcm/Morpho/internal/logger"
)

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Exibe informações do sistema",
}

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Exibe os logs da aplicação",
	Run: func(cmd *cobra.Command, args []string) {
		path := logger.GetLogPath()
		f, err := os.Open(path)
		if err != nil {
			fmt.Printf("Nenhum log encontrado em %s\n", path)
			return
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			fmt.Printf("Erro ao ler logs: %v\n", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(showCmd)
	showCmd.AddCommand(logsCmd)
}
