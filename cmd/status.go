package cmd

import (
	"os"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/process"
	"github.com/spf13/cobra"

	"github.com/vhwcm/Morpho/internal/agentkit"
	"github.com/vhwcm/Morpho/internal/ui"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Exibe o status atual de todos os agentes",
	RunE: func(cmd *cobra.Command, args []string) error {
		specs, err := agentkit.ListSpecs()
		if err != nil {
			return err
		}

		ui.Header("Status dos Agentes")

		// Detectar se há processos do morpho rodando
		isRunning := isMorphoRunning()

		headers := []string{"Agente", "Status", "Última Atividade"}
		rows := [][]string{}

		for _, spec := range specs {
			status := "💤 Em repouso"
			lastAct := "Nunca"

			outputs, _ := agentkit.ListOutputs(spec.Name, 1)
			if len(outputs) > 0 {
				last := outputs[0]
				lastAct = last.CreatedAt.Format("02/01 15:04")
				
				// Se rodou nos últimos 5 minutos e o processo geral está rodando, 
				// inferimos que pode estar em atividade (heurística CLI)
				if isRunning && time.Since(last.CreatedAt) < 5*time.Minute {
					status = "⚡ Em atividade"
				} else {
					status = "✅ Tarefa concluída"
				}
			}

			rows = append(rows, []string{
				spec.Name,
				status,
				lastAct,
			})
		}

		ui.Table(headers, rows)
		
		if isRunning {
			ui.Info("Dica: Existe um processo do Morpho em execução no sistema.")
		}

		return nil
	},
}

func isMorphoRunning() bool {
	procs, err := process.Processes()
	if err != nil {
		return false
	}

	myPid := os.Getpid()
	for _, p := range procs {
		if int(p.Pid) == myPid {
			continue
		}
		name, _ := p.Name()
		if strings.Contains(strings.ToLower(name), "morpho") {
			return true
		}
	}
	return false
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
