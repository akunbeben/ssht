package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/akunbeben/ssht/internal/config"
	"github.com/akunbeben/ssht/internal/ssh"
	"github.com/akunbeben/ssht/internal/tui"
	"github.com/akunbeben/ssht/internal/vpn"
)

var profileFlag string

var rootCmd = &cobra.Command{
	Use:   "ssht",
	Short: "SSH & VPN manager",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		profileName := profileFlag
		if profileName == "" {
			profileName = cfg.LastProfile
		}
		profile, ok := cfg.Profiles[profileName]
		if !ok {
			return fmt.Errorf("profile %q not found", profileName)
		}
		candidates, err := findHistoryCandidates(cfg, profileName)
		if err == nil && len(candidates) > 0 {
			fmt.Fprintf(cmd.ErrOrStderr(), "Found %d new server(s) in shell history.\n", len(candidates))
			if importHistoryPrompt(fmt.Sprintf("Import them to profile %q? [y/N] ", profileName)) {
				summary, err := mergeImportedServers(cfg, profileName, profile, candidates, true)
				if err == nil && summary.Added > 0 {
					fmt.Fprintf(cmd.ErrOrStderr(), "✓ imported %d server(s)\n", summary.Added)
					profile = cfg.Profiles[profileName]
				}
			} else {
				if importHistoryPrompt("Skip these permanently? [y/N] ") {
					if err := skipImportedServers(cfg, profileName, candidates); err == nil {
						fmt.Fprintln(cmd.ErrOrStderr(), "✓ servers added to skip list")
					}
				}
			}
		}

		for {
			action, err := tui.Run(cfg, profileName, profile)
			if err != nil {
				return err
			}

			// Update state in case profile was switched in TUI.
			profileName = action.ProfileName
			profile = cfg.Profiles[profileName]

			switch action.Type {
			case tui.ActionNone:
				return nil
			case tui.ActionToggleVPN:
				if err := toggleVPN(profile); err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
					fmt.Fprintf(os.Stderr, "Press Enter to continue...")
					bufio.NewReader(os.Stdin).ReadString('\n')
				}
			case tui.ActionConnect:
				if action.Server == nil {
					return errors.New("no server selected")
				}
				if err := ssh.Connect(*action.Server, profile.VPN, cfg.PrivacyMode); err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
					fmt.Fprintf(os.Stderr, "Press Enter to continue...")
					bufio.NewReader(os.Stdin).ReadString('\n')
				}
			default:
				return nil
			}
		}
	},
}

// Execute runs the root cobra command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&profileFlag, "profile", "p", "", "profile to use")
	rootCmd.AddCommand(profileCmd)
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(keyCmd)
}

func toggleVPN(profile config.Profile) error {
	if profile.VPN == nil {
		return errors.New("vpn is not configured for this profile")
	}
	iface := vpn.InterfaceName(profile.VPN.ConfPath)
	if vpn.IsActive(iface) {
		fmt.Fprintf(os.Stderr, "Stopping VPN (%s)...\n", iface)
		return vpn.Down(profile.VPN.ConfPath)
	}
	fmt.Fprintf(os.Stderr, "Starting SYSTEM VPN (global)...\n")
	fmt.Fprintf(os.Stderr, "Tip: To use an isolated VPN just for this SSH session (no sudo/global impact), just press Enter in the TUI.\n")
	return vpn.Up(profile.VPN.ConfPath)
}

func sortedProfileNames(cfg *config.Config) []string {
	names := make([]string, 0, len(cfg.Profiles))
	for name := range cfg.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func importHistoryPrompt(label string) bool {
	fmt.Fprintf(os.Stderr, "%s", label)
	reader := bufio.NewReader(os.Stdin)
	ans, _ := reader.ReadString('\n')
	ans = strings.ToLower(strings.TrimSpace(ans))
	return ans == "y" || ans == "yes"
}
