package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/akunbeben/ssht/internal/config"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Manage multi-device synchronization via Git",
}

var (
	syncPullForce bool
)

var syncSetupCmd = &cobra.Command{
	Use:   "setup [repo-url]",
	Short: "Setup Git synchronization",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoURL := args[0]
		dir, err := config.ConfigDir()
		if err != nil {
			return err
		}

		if _, err := os.Stat(filepath.Join(dir, ".git")); os.IsNotExist(err) {
			if err := runGit(dir, "init"); err != nil {
				return err
			}
		}

		if err := runGit(dir, "remote", "add", "origin", repoURL); err != nil {
			if err := runGit(dir, "remote", "set-url", "origin", repoURL); err != nil {
				return err
			}
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}
		cfg.SyncEnabled = true
		cfg.SyncRepo = repoURL
		if err := config.Save(cfg); err != nil {
			return err
		}

		fmt.Printf("✓ Sync setup complete for: %s\n", repoURL)
		fmt.Println("Tip: Run 'ssht sync pull' to get latest changes or 'ssht sync push' to save your current config.")
		return nil
	},
}

var syncPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push configuration changes to remote",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := config.ConfigDir()
		if err != nil {
			return err
		}

		if _, err := os.Stat(filepath.Join(dir, ".git")); os.IsNotExist(err) {
			return fmt.Errorf("git not initialized. Run 'ssht sync setup' first")
		}

		if err := runGit(dir, "add", "."); err != nil {
			return err
		}

		status, _ := exec.Command("git", "-C", dir, "status", "--porcelain").Output()
		if len(status) == 0 {
			fmt.Println("Everything up-to-date")
			return nil
		}

		hostname, _ := os.Hostname()
		if hostname == "" {
			hostname = "unknown-device"
		}
		msg := fmt.Sprintf("ssht: update configuration from %s at %s", hostname, time.Now().Format("2006-01-02 15:04:05"))
		if err := runGit(dir, "commit", "-m", msg); err != nil {
			return err
		}

		branch := getRemoteBranch(dir)
		if err := runGit(dir, "push", "origin", "HEAD:"+branch); err != nil {
			return fmt.Errorf("failed to push: %w", err)
		}

		fmt.Println("✓ Configuration pushed successfully")
		return nil
	},
}

var syncPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull configuration changes from remote",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := config.ConfigDir()
		if err != nil {
			return err
		}

		if _, err := os.Stat(filepath.Join(dir, ".git")); os.IsNotExist(err) {
			return fmt.Errorf("git not initialized. Run 'ssht sync setup' first")
		}

		branch := getRemoteBranch(dir)

		if syncPullForce {
			fmt.Printf("! Force pull: resetting to origin/%s...\n", branch)
			_ = runGit(dir, "rebase", "--abort")
			_ = runGit(dir, "merge", "--abort")

			if err := runGit(dir, "fetch", "origin", branch); err != nil {
				return fmt.Errorf("failed to fetch: %w", err)
			}
			if err := runGit(dir, "reset", "--hard", "origin/"+branch); err != nil {
				return fmt.Errorf("failed to reset: %w", err)
			}
			// Ensure we are actually on the branch and not detached
			_ = runGit(dir, "checkout", "-B", branch)
		} else {
			if err := runGit(dir, "pull", "--rebase", "--autostash", "origin", branch); err != nil {
				_ = runGit(dir, "rebase", "--abort")
				return fmt.Errorf("failed to pull (rebase conflict). Please resolve manualy in %s", dir)
			}
		}

		fmt.Println("✓ Configuration pulled successfully")
		return nil
	},
}

func getRemoteBranch(dir string) string {
	// Try to find default branch on origin
	out, err := exec.Command("git", "-C", dir, "remote", "show", "origin").Output()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.Contains(line, "HEAD branch:") {
				parts := strings.Fields(line)
				if len(parts) >= 3 {
					return parts[2]
				}
			}
		}
	}

	// Fallback 1: Current local branch
	out, err = exec.Command("git", "-C", dir, "branch", "--show-current").Output()
	if err == nil {
		curr := strings.TrimSpace(string(out))
		if curr != "" {
			return curr
		}
	}

	return "main"
}

func runGit(dir string, args ...string) error {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.AddCommand(syncSetupCmd)
	syncCmd.AddCommand(syncPushCmd)
	syncCmd.AddCommand(syncPullCmd)
	syncPullCmd.Flags().BoolVarP(&syncPullForce, "force", "f", false, "force overwrite local changes with remote")
}
