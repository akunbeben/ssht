package cmd

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/akunbeben/ssht/internal/config"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Manage servers",
}

var serverListCmd = &cobra.Command{
	Use:   "list",
	Short: "List servers in active profile",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, profileName, profile, err := loadActiveProfile()
		if err != nil {
			return err
		}
		_ = cfg
		fmt.Fprintf(cmd.OutOrStdout(), "profile: %s\n", profileName)
		for _, s := range profile.Servers {
			port := s.Port
			if port == 0 {
				port = 22
			}
			fmt.Fprintf(cmd.OutOrStdout(), "- %s %s@%s:%d\n", s.Name, s.User, s.Host, port)
		}
		return nil
	},
}

var (
	addName string
	addHost string
	addPort int
	addUser string
	addKey  string
	addTags string
	addNote string
)

var serverAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a server to active profile",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, profileName, profile, err := loadActiveProfile()
		if err != nil {
			return err
		}
		if addName == "" || addHost == "" || addUser == "" {
			return fmt.Errorf("--name, --host, and --user are required")
		}
		for _, s := range profile.Servers {
			if s.Name == addName {
				return fmt.Errorf("server name %q already exists in profile %q", addName, profileName)
			}
		}
		if addPort <= 0 {
			addPort = 22
		}
		tags := []string{}
		if addTags != "" {
			for _, t := range strings.Split(addTags, ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					tags = append(tags, t)
				}
			}
		}
		profile.Servers = append(profile.Servers, config.Server{
			ID:      uuid.NewString(),
			Name:    addName,
			Host:    addHost,
			Port:    addPort,
			User:    addUser,
			KeyPath: addKey,
			Tags:    tags,
			Note:    addNote,
		})

		exKey := fmt.Sprintf("%s@%s:%d", addUser, addHost, addPort)
		profile.ImportExceptions = removeStr(profile.ImportExceptions, exKey)

		cfg.Profiles[profileName] = profile
		return config.Save(cfg)
	},
}

var serverRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove server by name",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, profileName, profile, err := loadActiveProfile()
		if err != nil {
			return err
		}
		name := args[0]
		next := make([]config.Server, 0, len(profile.Servers))
		var removedServer *config.Server
		for _, s := range profile.Servers {
			if s.Name == name {
				copy := s
				removedServer = &copy
				continue
			}
			next = append(next, s)
		}
		if removedServer == nil {
			return fmt.Errorf("server %q not found", name)
		}

		port := removedServer.Port
		if port == 0 {
			port = 22
		}
		exKey := fmt.Sprintf("%s@%s:%d", removedServer.User, removedServer.Host, port)
		if !containsStr(profile.ImportExceptions, exKey) {
			profile.ImportExceptions = append(profile.ImportExceptions, exKey)
		}

		profile.Servers = next
		cfg.Profiles[profileName] = profile
		return config.Save(cfg)
	},
}

func init() {
	serverCmd.AddCommand(serverListCmd)
	serverCmd.AddCommand(serverAddCmd)
	serverCmd.AddCommand(serverRemoveCmd)

	serverAddCmd.Flags().StringVar(&addName, "name", "", "server name")
	serverAddCmd.Flags().StringVar(&addHost, "host", "", "server host or ip")
	serverAddCmd.Flags().IntVar(&addPort, "port", 22, "server port")
	serverAddCmd.Flags().StringVar(&addUser, "user", "", "ssh user")
	serverAddCmd.Flags().StringVar(&addKey, "key", "", "private key path")
	serverAddCmd.Flags().StringVar(&addTags, "tags", "", "comma separated tags")
	serverAddCmd.Flags().StringVar(&addNote, "note", "", "optional note")
}

func loadActiveProfile() (*config.Config, string, config.Profile, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, "", config.Profile{}, err
	}
	profileName := profileFlag
	if profileName == "" {
		profileName = cfg.LastProfile
	}
	profile, ok := cfg.Profiles[profileName]
	if !ok {
		return nil, "", config.Profile{}, fmt.Errorf("profile %q not found", profileName)
	}
	if cfg.LastProfile != profileName {
		cfg.LastProfile = profileName
		if err := config.Save(cfg); err != nil {
			return nil, "", config.Profile{}, err
		}
	}
	return cfg, profileName, profile, nil
}

func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func removeStr(slice []string, s string) []string {
	result := make([]string, 0, len(slice))
	for _, v := range slice {
		if v != s {
			result = append(result, v)
		}
	}
	return result
}
