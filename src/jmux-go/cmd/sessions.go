package cmd

import (
	"github.com/spf13/cobra"
)

// sessionsCmd represents the sessions command
var sessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "List all active shared sessions",
	Long:  `Display a list of all currently active shared sessions.`,
	Run: func(cmd *cobra.Command, args []string) {
		err := sessMgr.ListSessions()
		if err != nil {
			cmd.Printf("Error listing sessions: %v\n", err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(sessionsCmd)
}