package cmd

import (
	"github.com/spf13/cobra"
)

var (
	joinView     bool
	joinRogue    bool
	joinPassword string
)

// joinCmd represents the join command
var joinCmd = &cobra.Command{
	Use:   "join <user> [session]",
	Short: "Join another user's shared session",
	Long: `Join another user's shared tmux session.

Join Modes:
  Default: Use the session's configured mode (pair/view/rogue)
  --view:  Force view-only mode (read-only, regardless of session mode)
  --rogue: Force rogue mode (independent control, regardless of session mode)

Security Options:
  --password: Password for secure sessions

Examples:
  dmux join alice                    # Join alice's default session with its configured mode
  dmux join bob mysession           # Join bob's specific session with its configured mode
  dmux join alice --view            # Join alice's session in read-only mode
  dmux join bob mysession --rogue   # Join bob's session in rogue mode
  dmux join alice --password mypass # Join alice's secure session with password`,
	Args: cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		// Validate mutually exclusive flags
		if joinView && joinRogue {
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
		if joinView {
			modeOverride = "view"
		} else if joinRogue {
			modeOverride = "rogue"
		}

		err := sessMgr.JoinSession(hostUser, sessionName, modeOverride, joinPassword)
		if err != nil {
			cmd.Printf("Error joining session: %v\n", err)
			cmd.Printf("Tip: Try 'dmux sessions' to see available sessions\n")
			cmd.Printf("Usage: dmux join <host-user> [session-name]\n")
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(joinCmd)
	
	joinCmd.Flags().BoolVar(&joinView, "view", false, "Force view-only mode (read-only)")
	joinCmd.Flags().BoolVar(&joinRogue, "rogue", false, "Force rogue mode (independent control)")
	joinCmd.Flags().StringVar(&joinPassword, "password", "", "Password for secure sessions")
}