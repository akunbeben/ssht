package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ExpandHome resolves ~/ prefix into an absolute path.
func ExpandHome(path string) (string, error) {
	if !strings.HasPrefix(path, "~/") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, path[2:]), nil
}
