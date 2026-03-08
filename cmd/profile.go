package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/akunbeben/ssht/internal/config"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage profiles",
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		for _, name := range sortedProfileNames(cfg) {
			marker := " "
			if name == cfg.LastProfile {
				marker = "*"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", marker, name)
		}
		return nil
	},
}

var profileNewCmd = &cobra.Command{
	Use:   "new <name>",
	Short: "Create a profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		name := args[0]
		if _, ok := cfg.Profiles[name]; ok {
			return fmt.Errorf("profile %q already exists", name)
		}
		cfg.Profiles[name] = config.Profile{Name: name, Servers: []config.Server{}}
		cfg.LastProfile = name
		return config.Save(cfg)
	},
}

var profileDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		name := args[0]
		if _, ok := cfg.Profiles[name]; !ok {
			return fmt.Errorf("profile %q not found", name)
		}
		if len(cfg.Profiles) == 1 {
			return fmt.Errorf("cannot delete last profile")
		}
		delete(cfg.Profiles, name)
		if cfg.LastProfile == name {
			for next := range cfg.Profiles {
				cfg.LastProfile = next
				break
			}
		}
		return config.Save(cfg)
	},
}

var (
	vpnTypeFlag     string
	vpnConfPathFlag string
	vpnAutoUpFlag   bool
	vpnAutoDownFlag bool
)

var profileSetVPNCmd = &cobra.Command{
	Use:   "set-vpn <name>",
	Short: "Set VPN config for a profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		name := args[0]
		profile, ok := cfg.Profiles[name]
		if !ok {
			return fmt.Errorf("profile %q not found", name)
		}
		if vpnConfPathFlag == "" {
			profile.VPN = nil
		} else {
			if vpnTypeFlag == "" {
				vpnTypeFlag = "wireguard"
			}
			profile.VPN = &config.VPNConf{
				Type:     vpnTypeFlag,
				ConfPath: vpnConfPathFlag,
				AutoUp:   vpnAutoUpFlag,
				AutoDown: vpnAutoDownFlag,
			}
		}
		cfg.Profiles[name] = profile
		return config.Save(cfg)
	},
}

func init() {
	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileNewCmd)
	profileCmd.AddCommand(profileDeleteCmd)
	profileCmd.AddCommand(profileSetVPNCmd)

	profileSetVPNCmd.Flags().StringVar(&vpnTypeFlag, "type", "wireguard", "vpn type")
	profileSetVPNCmd.Flags().StringVar(&vpnConfPathFlag, "conf", "", "wireguard conf path; empty to clear vpn")
	profileSetVPNCmd.Flags().BoolVar(&vpnAutoUpFlag, "auto-up", true, "auto up before ssh connect")
	profileSetVPNCmd.Flags().BoolVar(&vpnAutoDownFlag, "auto-down", false, "auto down after ssh exit")
}
