package cmd

import (
	"github.com/spf13/cobra"
)

// stopCmd represents the stop command
var stopCmd = &cobra.Command{
	Use:   "stop [sessions...]",
	Short: "Stop sharing sessions",
	Long: `Stop sharing one or more sessions.

Examples:
  jmux stop                    # Stop all shared sessions
  jmux stop session1 session2 # Stop specific sessions`,
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