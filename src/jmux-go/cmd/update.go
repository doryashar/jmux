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
This will download and replace the current binary with the latest release.

Auto-updates are enabled by default. To disable auto-updates, set:
  export DMUX_AUTO_UPDATE=false

When auto-updates are enabled, dmux will automatically download and install 
updates without prompting when they become available.`,
	Run: func(cmd *cobra.Command, args []string) {
		force, _ := cmd.Flags().GetBool("force")
		
		// Show auto-update status
		if cfg.AutoUpdate {
			cmd.Printf("Auto-updates: enabled (set DMUX_AUTO_UPDATE=false to disable)\n\n")
		} else {
			cmd.Printf("Auto-updates: disabled (set DMUX_AUTO_UPDATE=true to enable)\n\n")
		}
		
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