package cmd

import (
	"github.com/spf13/cobra"
)

// stopCmd represents the stop command
var stopCmd = &cobra.Command{
	Use:   "stop [sessions...]",
	Short: "Stop sharing sessions",
	Long: `Stop sharing one or more sessions.

When run without arguments:
- If inside a tmux session: stops sharing the current session only
- If outside tmux: stops all shared sessions

Examples:
  dmux stop                    # Stop current session (if in tmux) or all sessions
  dmux stop session1 session2 # Stop specific sessions`,
	Run: func(cmd *cobra.Command, args []string) {
		err := sessMgr.StopShare(args)
		if err != nil {
			cmd.Printf("Error stopping sessions: %v\n", err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}