package cmd

import (
	"github.com/spf13/cobra"
	"jmux/internal/updater"
)

// checkUpdatesCmd represents the check-updates command
var checkUpdatesCmd = &cobra.Command{
	Use:    "check-updates",
	Short:  "Check for available updates",
	Long:   `Check if there are any updates available without installing them.`,
	Hidden: true, // Hidden command for testing
	Run: func(cmd *cobra.Command, args []string) {
		// Force a check regardless of timing - use auto-update setting
		updater.CheckForUpdatesIfNeeded(cfg.ConfigDir, cfg.AutoUpdate)
	},
}

func init() {
	rootCmd.AddCommand(checkUpdatesCmd)
}