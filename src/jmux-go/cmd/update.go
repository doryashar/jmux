package cmd

import (
	"github.com/spf13/cobra"
	"jmux/internal/updater"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update dmux to the latest version",
	Long: `Check for and install the latest version of dmux from GitHub releases.
This will download and replace the current binary with the latest release.`,
	Run: func(cmd *cobra.Command, args []string) {
		force, _ := cmd.Flags().GetBool("force")
		err := updater.CheckAndUpdate(force)
		if err != nil {
			cmd.Printf("Update failed: %v\n", err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.Flags().BoolP("force", "f", false, "Force update even if already up to date")
}