package cmd

import (
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	askSharePassword string
	askSharePrivate  bool
	askShareMode     string
)

// askShareCmd represents the ask-share command
var askShareCmd = &cobra.Command{
	Use:   "ask-share [users...]",
	Short: "Request session sharing from other users (reverse sharing)",
	Long: `Ask other users to share their sessions with you. This starts a listening server
on your machine and sends invitation messages to the specified users.

When they join, they will share their tmux session with you.

Examples:
  dmux ask-share alice bob          # Ask alice and bob to share with you
  dmux ask-share --password secret  # Ask with password protection
  dmux ask-share --private alice    # Private sharing (only alice can join)
  dmux ask-share --mode view alice  # Ask alice to share in view-only mode`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			color.Yellow("⚠️  No users specified. Use: dmux ask-share <users...>")
			return
		}

		// Parse mode
		if askShareMode != "" && askShareMode != "pair" && askShareMode != "view" && askShareMode != "rogue" {
			color.Red("❌ Invalid mode: %s. Must be 'pair', 'view', or 'rogue'", askShareMode)
			return
		}

		err := sessMgr.StartReverseShare(args, askSharePassword, askSharePrivate, askShareMode)
		if err != nil {
			color.Red("❌ Error starting reverse share: %v", err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(askShareCmd)
	
	askShareCmd.Flags().StringVarP(&askSharePassword, "password", "p", "", "Set password for secure sharing")
	askShareCmd.Flags().BoolVar(&askSharePrivate, "private", false, "Create private sharing session")
	askShareCmd.Flags().StringVarP(&askShareMode, "mode", "m", "pair", "Sharing mode: pair, view, or rogue")
}