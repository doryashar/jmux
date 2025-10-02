package cmd

import (
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// attachCmd represents the attach command
var attachCmd = &cobra.Command{
	Use:   "attach [session]",
	Short: "Attach to a tmux session",
	Long:  `Attach to an existing tmux session. If no session specified, attaches to the most recent.`,
	Run: func(cmd *cobra.Command, args []string) {
		sessionName := ""
		if len(args) > 0 {
			sessionName = args[0]
		}

		color.Blue("🔗 Attaching to tmux session...")
		
		err := tmuxMgr.AttachToSession(sessionName)
		if err != nil {
			cmd.Printf("Error attaching to session: %v\n", err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(attachCmd)
}