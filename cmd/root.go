package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"sort"

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
		summary, err := autoImportHistory(cfg, profileName)
		if err == nil && summary.Added > 0 {
			fmt.Fprintf(cmd.ErrOrStderr(), "auto-import: added %d server(s) from history\n", summary.Added)
			profile = cfg.Profiles[profileName]
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
				if err := ssh.Connect(*action.Server, profile.VPN); err != nil {
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
