package main

import (
	"os"

	"github.com/vhwcm/Morpho/cmd"
	"github.com/vhwcm/Morpho/internal/logger"
	"github.com/vhwcm/Morpho/internal/ui"
)

func main() {
	defer logger.RecoverPanic()
	if err := cmd.Execute(); err != nil {
		logger.Error("Execução encerrada com erro", err)
		ui.ErrorToStderr(err.Error())
		os.Exit(1)
	}
}
