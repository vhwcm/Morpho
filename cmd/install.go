package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vhwcm/Morpho/internal/ui"
)

var installOutputPath string
var installLinkLocalBin bool

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Compila o Morpho e gera binário local",
	Long:  "Compila o projeto atual e gera o binário na pasta bin, por padrão em bin/morpho.",
	RunE: func(cmd *cobra.Command, _ []string) error {
		target := filepath.Clean(installOutputPath)
		if target == "" || target == "." {
			return fmt.Errorf("caminho de saída inválido")
		}

		dir := filepath.Dir(target)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}

		ui.Header("Instalação do Morpho")
		ui.Info(fmt.Sprintf("Compilando binário em: %s", target))

		buildCmd := exec.CommandContext(cmd.Context(), "go", "build", "-o", target, ".")
		buildCmd.Env = os.Environ()
		output, err := buildCmd.CombinedOutput()
		if err != nil {
			if len(output) > 0 {
				ui.Panel("Saída do build", string(output))
			}
			return fmt.Errorf("falha ao compilar: %w", err)
		}

		absTarget, _ := filepath.Abs(target)
		ui.Success("Instalação concluída.")
		ui.Info("Nome do binário: morpho")
		ui.Info(fmt.Sprintf("Binário gerado em: %s", absTarget))
		ui.Info("Use agora: ./bin/morpho help")

		if installLinkLocalBin {
			localBinPath, inPath, err := installToLocalBin(absTarget)
			if err != nil {
				ui.Warn(fmt.Sprintf("Não foi possível publicar em ~/.local/bin: %v", err))
			} else {
				ui.Success(fmt.Sprintf("Binário publicado em: %s", localBinPath))
				if inPath {
					ui.Info("Você pode executar diretamente: morpho help")
				} else {
					ui.Warn("~/.local/bin não está no PATH atual.")
					ui.Info("Adicione ao shell e reabra o terminal:")
					ui.Panel("PATH", "export PATH=\"$HOME/.local/bin:$PATH\"")
				}
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().StringVar(&installOutputPath, "output", filepath.Join("bin", "morpho"), "caminho do binário de saída")
	installCmd.Flags().BoolVar(&installLinkLocalBin, "link-local-bin", true, "também publica em ~/.local/bin/morpho")
}

func installToLocalBin(source string) (string, bool, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false, err
	}

	localBinDir := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(localBinDir, 0o755); err != nil {
		return "", false, err
	}

	target := filepath.Join(localBinDir, "morpho")
	if err := copyFile(source, target); err != nil {
		return "", false, err
	}

	inPath := isDirInPath(localBinDir)
	return target, inPath, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return out.Sync()
}

func isDirInPath(dir string) bool {
	pathEntries := strings.Split(os.Getenv("PATH"), ":")
	for _, entry := range pathEntries {
		if strings.TrimSpace(entry) == dir {
			return true
		}
	}
	return false
}
