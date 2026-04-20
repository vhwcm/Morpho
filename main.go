package main

import (
	"os"

	"github.com/vhwcm/Morpho/cmd"
	"github.com/vhwcm/Morpho/internal/ui"
)

func main() {
	if err := cmd.Execute(); err != nil {
		ui.ErrorToStderr(err.Error())
		os.Exit(1)
	}
}
