package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/vhwcm/Morpho/internal/agentkit"
	"github.com/vhwcm/Morpho/internal/config"
	"github.com/vhwcm/Morpho/internal/gemini"
	"github.com/vhwcm/Morpho/internal/memory"
	"github.com/vhwcm/Morpho/internal/ui"
)

var (
	memoryPruneMaxDocs int
	memorySearchTopK   int
	memorySearchMin    float64
)

var agentMemoryCmd = &cobra.Command{
	Use:   "memory",
	Short: "Gerencia base de memória semântica por agente",
}

var agentMemoryStatusCmd = &cobra.Command{
	Use:   "status [nome-agente]",
	Short: "Exibe estatísticas da memória do agente",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		stats, err := memory.GetStats(strings.TrimSpace(args[0]))
		if err != nil {
			return err
		}
		ui.Header("Status da memória")
		ui.Table([]string{"Agente", "Documentos", "Chunks"}, [][]string{{args[0], fmt.Sprintf("%d", stats.Documents), fmt.Sprintf("%d", stats.Chunks)}})
		return nil
	},
}

var agentMemorySearchCmd = &cobra.Command{
	Use:   "search [nome-agente] [consulta]",
	Short: "Busca semântica na memória do agente",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		spec, err := agentkit.LoadSpec(strings.TrimSpace(args[0]))
		if err != nil {
			return err
		}
		cfg := config.Load()
		var client agentkit.AIClient
		client, err = gemini.NewClient(cfg.GeminiAPIKey, spec.Model)
		if err != nil {
			if cfg.GeminiAPIKey == "" {
				client = gemini.NewMockClient()
			} else {
				return err
			}
		}

		topK := memorySearchTopK
		if topK <= 0 {
			topK = cfg.Memory.TopK
		}
		min := memorySearchMin
		if min <= 0 {
			min = cfg.Memory.MinScore
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 20*time.Second)
		defer cancel()
		results, err := memory.SearchWithPolicy(ctx, spec.Name, strings.TrimSpace(args[1]), topK, min, client, cfg.Memory.ReadPolicy)
		if err != nil {
			return err
		}
		if len(results) == 0 {
			ui.Warn("Nenhum trecho relevante encontrado.")
			return nil
		}

		rows := make([][]string, 0, len(results))
		for i, r := range results {
			text := strings.TrimSpace(r.Text)
			if len(text) > 120 {
				text = text[:120] + "..."
			}
			rows = append(rows, []string{fmt.Sprintf("%d", i+1), fmt.Sprintf("%.3f", r.Score), r.Agent, text})
		}
		ui.Header("Resultados de memória")
		ui.Table([]string{"#", "Score", "Agente", "Trecho"}, rows)
		return nil
	},
}

var agentMemoryReindexCmd = &cobra.Command{
	Use:   "reindex [nome-agente]",
	Short: "Reindexa embeddings dos chunks existentes",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentName := strings.TrimSpace(args[0])
		cfg := config.Load()
		spec, err := agentkit.LoadSpec(agentName)
		if err != nil {
			return err
		}
		var client agentkit.AIClient
		client, err = gemini.NewClient(cfg.GeminiAPIKey, spec.Model)
		if err != nil {
			if cfg.GeminiAPIKey == "" {
				client = gemini.NewMockClient()
			} else {
				return err
			}
		}
		chunks, err := memory.ListChunks(agentName)
		if err != nil {
			return err
		}
		if len(chunks) == 0 {
			ui.Warn("Nenhum chunk para reindexar.")
			return nil
		}

		for _, ch := range chunks {
			select {
			case <-cmd.Context().Done():
				return cmd.Context().Err()
			default:
			}
			emb, err := client.Embed(cmd.Context(), ch.Text)
			if err != nil {
				continue
			}
			_ = memory.InsertChunks(agentName, ch.DocumentID, []memory.ChunkInput{{
				Text:      ch.Text,
				Tokens:    ch.Tokens,
				Embedding: emb,
				Hash:      ch.Hash,
			}})
		}
		ui.Success("Reindexação concluída.")
		return nil
	},
}

var agentMemoryPruneCmd = &cobra.Command{
	Use:   "prune [nome-agente]",
	Short: "Mantém somente os documentos mais recentes na memória",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		if err := memory.Prune(strings.TrimSpace(args[0]), memoryPruneMaxDocs); err != nil {
			return err
		}
		ui.Success("Limpeza da memória concluída.")
		return nil
	},
}

func init() {
	agentCmd.AddCommand(agentMemoryCmd)
	agentMemoryCmd.AddCommand(agentMemoryStatusCmd)
	agentMemoryCmd.AddCommand(agentMemorySearchCmd)
	agentMemoryCmd.AddCommand(agentMemoryReindexCmd)
	agentMemoryCmd.AddCommand(agentMemoryPruneCmd)

	agentMemoryPruneCmd.Flags().IntVar(&memoryPruneMaxDocs, "max-docs", 200, "quantidade máxima de documentos mantidos")
	agentMemorySearchCmd.Flags().IntVar(&memorySearchTopK, "topk", 0, "top-k de resultados")
	agentMemorySearchCmd.Flags().Float64Var(&memorySearchMin, "min-score", 0, "score mínimo")
}
