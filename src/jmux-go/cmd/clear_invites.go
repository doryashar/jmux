package cmd

import (
	"os"
	"path/filepath"
	
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// clearInvitesCmd represents the clear-invites command
var clearInvitesCmd = &cobra.Command{
	Use:   "clear-invites",
	Short: "Clear all pending invitations",
	Long: `Clear all pending invitations from your invitations file.
	
This will remove all invitations that haven't been accepted yet.
Invitations are automatically removed when you join a session from the inviting user,
or when they expire after 24 hours.`,
	Run: func(cmd *cobra.Command, args []string) {
		currentUser := os.Getenv("USER")
		if currentUser == "" {
			color.Red("❌ Unable to determine current user")
			return
		}

		userInvitesFile := filepath.Join(cfg.MessagesDir, currentUser+".invites")
		
		// Check if file exists
		if _, err := os.Stat(userInvitesFile); os.IsNotExist(err) {
			color.Yellow("📭 No invitations file found")
			return
		}

		// Clear the file
		if err := os.Truncate(userInvitesFile, 0); err != nil {
			color.Red("❌ Failed to clear invitations: %v", err)
			return
		}

		color.Green("✅ All pending invitations cleared")
		color.Blue("💡 You can view invitations anytime with 'dmux messages'")
	},
}

func init() {
	rootCmd.AddCommand(clearInvitesCmd)
}