package cmd

import (
	"github.com/spf13/cobra"
)

var (
	shareName    string
	sharePrivate bool
	shareInvite  []string
	shareView    bool
	shareRogue   bool
)

// shareCmd represents the share command
var shareCmd = &cobra.Command{
	Use:   "share [session-name]",
	Short: "Share the current tmux session",
	Long: `Share the current tmux session with other users.

Sharing Modes:
  Default: Standard shared session (all users have full control)
  --view:  View-only mode (joining users can only observe, read-only)
  --rogue: Rogue mode (joining users get independent control within same tmux server)

Examples:
  dmux share                              # Share current session publicly
  dmux share tomere                       # Share with name 'tomere'
  dmux share --name mysession             # Share with custom name
  dmux share --view                       # Share in read-only mode
  dmux share --rogue                      # Share in rogue mode (independent sessions)
  dmux share --private --invite user1,user2  # Private session with invites`,
	Run: func(cmd *cobra.Command, args []string) {
		// Use positional argument if provided, otherwise use flag
		sessionName := shareName
		if len(args) > 0 {
			sessionName = args[0]
		}
		
		// Validate mutually exclusive flags
		if shareView && shareRogue {
			cmd.Printf("Error: --view and --rogue flags are mutually exclusive\n")
			return
		}

		// Determine sharing mode
		var shareMode string
		switch {
		case shareView:
			shareMode = "view"
		case shareRogue:
			shareMode = "rogue"
		default:
			shareMode = "pair"
		}

		err := sessMgr.StartShare(sessionName, sharePrivate, shareInvite, shareMode)
		if err != nil {
			cmd.Printf("Error starting share: %v\n", err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(shareCmd)

	shareCmd.Flags().StringVar(&shareName, "name", "", "Custom session name")
	shareCmd.Flags().BoolVar(&sharePrivate, "private", false, "Create private session")
	shareCmd.Flags().StringSliceVar(&shareInvite, "invite", []string{}, "Users to invite (comma-separated)")
	shareCmd.Flags().BoolVar(&shareView, "view", false, "Share in view-only mode (read-only for joining users)")
	shareCmd.Flags().BoolVar(&shareRogue, "rogue", false, "Share in rogue mode (independent control for joining users)")
}