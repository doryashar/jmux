package cmd

import (
	"github.com/spf13/cobra"
)

var (
	shareName    string
	sharePrivate bool
	shareInvite  []string
)

// shareCmd represents the share command
var shareCmd = &cobra.Command{
	Use:   "share [session-name]",
	Short: "Share the current tmux session",
	Long: `Share the current tmux session with other users.

Examples:
  dmux share                              # Share current session publicly
  dmux share tomere                       # Share with name 'tomere'
  dmux share --name mysession             # Share with custom name
  dmux share --private --invite user1,user2  # Private session with invites`,
	Run: func(cmd *cobra.Command, args []string) {
		// Use positional argument if provided, otherwise use flag
		sessionName := shareName
		if len(args) > 0 {
			sessionName = args[0]
		}
		
		err := sessMgr.StartShare(sessionName, sharePrivate, shareInvite)
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
}