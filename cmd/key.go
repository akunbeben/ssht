package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	k "github.com/akunbeben/ssht/internal/key"
)

var keyCmd = &cobra.Command{
	Use:   "key",
	Short: "Manage SSH public keys",
}

var keyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List .pub keys in ~/.ssh",
	RunE: func(cmd *cobra.Command, args []string) error {
		keys, err := k.ScanPubkeys()
		if err != nil {
			return err
		}
		for _, path := range keys {
			fmt.Fprintln(cmd.OutOrStdout(), filepath.Base(path))
		}
		return nil
	},
}

var keyCopyCmd = &cobra.Command{
	Use:   "copy [name]",
	Short: "Copy public key content to clipboard",
	Args:  cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		keys, err := k.ScanPubkeys()
		if err != nil {
			return err
		}
		if len(keys) == 0 {
			return fmt.Errorf("no public keys found in ~/.ssh")
		}

		target := keys[0]
		if len(args) == 1 {
			name := args[0]
			found := false
			for _, path := range keys {
				if filepath.Base(path) == name {
					target = path
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("pubkey %q not found", name)
			}
		}

		content, err := k.CopyToClipboard(target)
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "copied %s\n", filepath.Base(target))
		fmt.Fprintln(cmd.OutOrStdout(), content)
		return nil
	},
}

var (
	newKeyPath    string
	newKeyComment string
)

var keyNewCmd = &cobra.Command{
	Use:   "new",
	Short: "Generate a new ed25519 keypair",
	RunE: func(cmd *cobra.Command, args []string) error {
		return k.Generate(newKeyPath, newKeyComment)
	},
}

func init() {
	keyCmd.AddCommand(keyListCmd)
	keyCmd.AddCommand(keyCopyCmd)
	keyCmd.AddCommand(keyNewCmd)

	keyNewCmd.Flags().StringVar(&newKeyPath, "out", "", "output private key path")
	keyNewCmd.Flags().StringVar(&newKeyComment, "comment", "ssht", "ssh key comment")
}
