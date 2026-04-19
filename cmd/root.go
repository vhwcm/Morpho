package cmd

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "gopher",
	Short: "Sistema agêntico para desenvolvimento de software",
	Long:  "Gopher é uma CLI para criar, editar e executar agentes de desenvolvimento com Gemini de forma simples.",
}

func Execute() error {
	return rootCmd.Execute()
}
