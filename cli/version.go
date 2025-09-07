package cli

import (
	"github.com/spf13/cobra"
)

// GetVersionString is a function that should be set from main package
var GetVersionString func() string

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Display version information including commit hash and Go version.`,
	Run: func(cmd *cobra.Command, args []string) {
		if GetVersionString != nil {
			cmd.Println(GetVersionString())
		} else {
			cmd.Println("Version information not available")
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}