package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
	BuiltBy = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of ssht",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("ssht version %s\n", Version)
		fmt.Printf("commit: %s\n", Commit)
		fmt.Printf("built at: %s\n", Date)
		fmt.Printf("built by: %s\n", BuiltBy)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
