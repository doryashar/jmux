package cmd

import (
	"github.com/spf13/cobra"
)

var (
	joinShareView     bool
	joinShareRogue    bool
	joinSharePassword string
)

// joinShareCmd represents the join-share command for reverse sharing
var joinShareCmd = &cobra.Command{
	Use:   "join-share <user> [session]",
	Short: "Join a reverse sharing session (respond to ask-share)",
	Long: `Join a reverse sharing session where another user has requested to share with you.
This command is used to respond to reverse sharing invitations from 'dmux ask-share'.

When someone runs 'dmux ask-share', they start listening for connections and send you
an invitation. Use this command to connect to their reverse sharing session.

Join Modes:
  Default: Use the session's configured mode (pair/view/rogue)
  --view:  Force view-only mode (read-only)
  --rogue: Force rogue mode (independent control)

Security Options:
  --password: Password for secure sessions

Examples:
  dmux join-share alice                    # Join alice's reverse sharing session
  dmux join-share bob reverse-123456       # Join bob's specific reverse session
  dmux join-share alice --view            # Join alice's reverse session in read-only mode
  dmux join-share bob --password secret   # Join bob's secure reverse session`,
	Args: cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		// Validate mutually exclusive flags
		if joinShareView && joinShareRogue {
			cmd.Printf("Error: --view and --rogue flags are mutually exclusive\n")
			return
		}

		hostUser := args[0]
		sessionName := ""
		if len(args) > 1 {
			sessionName = args[1]
		}

		// Determine join mode override
		var modeOverride string
		if joinShareView {
			modeOverride = "view"
		} else if joinShareRogue {
			modeOverride = "rogue"
		}

		// Use the regular join session but for reverse sharing
		err := sessMgr.JoinReverseShare(hostUser, sessionName, modeOverride, joinSharePassword)
		if err != nil {
			cmd.Printf("Error joining reverse sharing session: %v\n", err)
			cmd.Printf("Tip: Check if the user has sent you a reverse sharing invitation\n")
			cmd.Printf("Usage: dmux join-share <host-user> [session-name]\n")
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(joinShareCmd)
	
	joinShareCmd.Flags().BoolVar(&joinShareView, "view", false, "Force view-only mode (read-only)")
	joinShareCmd.Flags().BoolVar(&joinShareRogue, "rogue", false, "Force rogue mode (independent control)")
	joinShareCmd.Flags().StringVar(&joinSharePassword, "password", "", "Password for secure sessions")
}