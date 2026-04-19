package main

import (
	"os"

	"github.com/vhwcm/Gopher/cmd"
	"github.com/vhwcm/Gopher/internal/ui"
)

func main() {
	if err := cmd.Execute(); err != nil {
		ui.ErrorToStderr(err.Error())
		os.Exit(1)
	}
}
