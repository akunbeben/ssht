package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/akunbeben/ssht/cmd"
	"github.com/akunbeben/ssht/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if cfg.SyncEnabled {
		dir, _ := config.ConfigDir()
		// Auto-pull with autostash and rebase
		_ = exec.Command("git", "-C", dir, "pull", "--rebase", "--autostash", "origin", "HEAD").Run()
	}

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if cfg.SyncEnabled {
		dir, _ := config.ConfigDir()
		// Only commit and push if there are changes
		exec.Command("git", "-C", dir, "add", ".").Run()
		status, _ := exec.Command("git", "-C", dir, "status", "--porcelain").Output()
		if len(status) > 0 {
			hostname, _ := os.Hostname()
			if hostname == "" {
				hostname = "unknown-device"
			}
			msg := fmt.Sprintf("ssht: auto sync from %s at %s", hostname, time.Now().Format("2006-01-02 15:04:05"))
			if err := exec.Command("git", "-C", dir, "commit", "-m", msg).Run(); err == nil {
				_ = exec.Command("git", "-C", dir, "push", "origin", "HEAD").Run()
			}
		}
	}
}
