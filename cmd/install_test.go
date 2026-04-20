package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCopyFile(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src.bin")
	dst := filepath.Join(tmp, "dst.bin")

	content := []byte("morpho-binary")
	if err := os.WriteFile(src, content, 0o755); err != nil {
		t.Fatalf("erro ao preparar src: %v", err)
	}

	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile falhou: %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("erro ao ler dst: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("conteúdo copiado difere: got=%q want=%q", string(got), string(content))
	}

	info, err := os.Stat(dst)
	if err != nil {
		t.Fatalf("erro ao stat dst: %v", err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("arquivo destino deveria ser executável")
	}
}

func TestInstallToLocalBin(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("erro ao criar home fake: %v", err)
	}
	t.Setenv("HOME", home)

	source := filepath.Join(tmp, "morpho-src")
	if err := os.WriteFile(source, []byte("binary-content"), 0o755); err != nil {
		t.Fatalf("erro ao criar source: %v", err)
	}

	localBinDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", localBinDir+":/usr/bin")

	target, inPath, err := installToLocalBin(source)
	if err != nil {
		t.Fatalf("installToLocalBin falhou: %v", err)
	}

	expectedTarget := filepath.Join(localBinDir, "morpho")
	if target != expectedTarget {
		t.Fatalf("target inesperado: got=%s want=%s", target, expectedTarget)
	}
	if !inPath {
		t.Fatalf("~/.local/bin deveria ser detectado no PATH")
	}

	copied, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("erro ao ler binário copiado: %v", err)
	}
	if string(copied) != "binary-content" {
		t.Fatalf("conteúdo copiado inválido: %q", string(copied))
	}
}

func TestIsDirInPath(t *testing.T) {
	dir := "/tmp/morpho-bin"
	t.Setenv("PATH", strings.Join([]string{"/usr/local/bin", dir, "/usr/bin"}, ":"))
	if !isDirInPath(dir) {
		t.Fatalf("diretório deveria ser encontrado no PATH")
	}
	if isDirInPath("/tmp/nao-existe") {
		t.Fatalf("diretório não deveria ser encontrado no PATH")
	}
}
