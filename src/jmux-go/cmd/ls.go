package cmd

import (
	"github.com/spf13/cobra"
)

// lsCmd represents the ls command (tmux list-sessions)
var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List tmux sessions (enhanced)",
	Long:  `List tmux sessions with dmux enhancements and tips.`,
	Run: func(cmd *cobra.Command, args []string) {
		err := tmuxMgr.ListSessions()
		if err != nil {
			cmd.Printf("Error listing sessions: %v\n", err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(lsCmd)
}