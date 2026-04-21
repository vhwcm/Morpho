package cmd

import (
	"github.com/spf13/cobra"
	"github.com/vhwcm/Morpho/internal/logger"
	"github.com/vhwcm/Morpho/internal/ui"
)

var rootCmd = &cobra.Command{
	Use:   "morpho",
	Short: "Sistema agêntico para desenvolvimento de software",
	Long:  "Morpho é uma CLI para criar, editar e executar agentes de desenvolvimento com Gemini de forma simples.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if err := logger.Init(); err != nil {
			ui.ErrorToStderr("Falha ao inicializar logs: " + err.Error())
		}
		logger.Info("Iniciando comando", map[string]interface{}{
			"command": cmd.CommandPath(),
			"args":    args,
		})
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		logger.Close()
	},
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
