package key

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/atotto/clipboard"
)

func ScanPubkeys() ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home dir: %w", err)
	}
	sshDir := filepath.Join(home, ".ssh")
	entries, err := os.ReadDir(sshDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("read ~/.ssh: %w", err)
	}

	keys := make([]string, 0)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".pub") {
			keys = append(keys, filepath.Join(sshDir, name))
		}
	}
	sort.Strings(keys)
	return keys, nil
}

func CopyToClipboard(pubkeyPath string) (string, error) {
	data, err := os.ReadFile(pubkeyPath)
	if err != nil {
		return "", fmt.Errorf("read pubkey: %w", err)
	}
	content := strings.TrimSpace(string(data))
	if err := clipboard.WriteAll(content); err != nil {
		return "", fmt.Errorf("copy to clipboard: %w", err)
	}
	return content, nil
}

func Generate(outputPath, comment string) error {
	if outputPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("resolve home dir: %w", err)
		}
		outputPath = filepath.Join(home, ".ssh", "id_ed25519")
	}
	if comment == "" {
		comment = "ssht"
	}
	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-f", outputPath, "-N", "", "-C", comment)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
