package cmd

import (
	"github.com/spf13/cobra"
	"github.com/vhwcm/Morpho/internal/ui"
)

var rootCmd = &cobra.Command{
	Use:   "morpho",
	Short: "Sistema agêntico para desenvolvimento de software",
	Long:  "Morpho é uma CLI para criar, editar e executar agentes de desenvolvimento com Gemini de forma simples.",
}

func init() {
	origHelp := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(c *cobra.Command, args []string) {
		if c == rootCmd || c.Name() == "help" && len(args) == 0 {
			ui.ShowCustomHelp(rootCmd, args)
		} else {
			origHelp(c, args)
		}
	})

	rootCmd.Run = func(c *cobra.Command, args []string) {
		ui.ShowCustomHelp(c, args)
	}
}

func Execute() error {
	return rootCmd.Execute()
}
