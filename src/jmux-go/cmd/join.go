package cmd

import (
	"github.com/spf13/cobra"
)

// joinCmd represents the join command
var joinCmd = &cobra.Command{
	Use:   "join <user> [session]",
	Short: "Join another user's shared session",
	Long: `Join another user's shared tmux session.

Examples:
  jmux join alice                    # Join alice's default session
  jmux join bob mysession           # Join bob's specific session`,
	Args: cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		hostUser := args[0]
		sessionName := ""
		if len(args) > 1 {
			sessionName = args[1]
		}

		err := sessMgr.JoinSession(hostUser, sessionName)
		if err != nil {
			cmd.Printf("Error joining session: %v\n", err)
			cmd.Printf("Tip: Try 'jmux sessions' to see available sessions\n")
			cmd.Printf("Usage: jmux join <host-user> [session-name]\n")
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(joinCmd)
}