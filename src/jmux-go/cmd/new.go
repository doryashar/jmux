package cmd

import (
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// newCmd represents the new command
var newCmd = &cobra.Command{
	Use:   "new [session-name]",
	Short: "Create a new tmux session",
	Long:  `Create a new tmux session with an optional name.`,
	Run: func(cmd *cobra.Command, args []string) {
		sessionName := ""
		if len(args) > 0 {
			sessionName = args[0]
		}

		color.Blue("🆕 Creating new tmux session...")
		color.Yellow("💡 Tip: Use 'dmux share' to make it shareable")
		
		err := tmuxMgr.CreateSession(sessionName)
		if err != nil {
			cmd.Printf("Error creating session: %v\n", err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(newCmd)
}