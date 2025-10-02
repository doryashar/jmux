package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show jmux sharing status",
	Long:  `Show the current sharing status including active sessions and connection info.`,
	Run: func(cmd *cobra.Command, args []string) {
		showStatus()
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func showStatus() {
	currentUser := os.Getenv("USER")
	if currentUser == "" {
		color.Red("❌ Unable to determine current user")
		return
	}

	color.Blue("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	color.Green("JMUX Sharing Status")
	color.Blue("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Check if we're in a tmux session
	if tmuxSession := os.Getenv("TMUX"); tmuxSession != "" {
		color.Green("✓ Currently in tmux session")
		
		// Get current session name
		if sessionName, err := tmuxMgr.GetCurrentSession(); err == nil {
			fmt.Printf("  Session: %s\n", sessionName)
		}
	} else {
		color.Yellow("○ Not currently in a tmux session")
	}

	// Show active shared sessions for current user
	sessions, err := sessMgr.ListUserSessions(currentUser)
	if err != nil {
		color.Red("❌ Error reading sessions: %v", err)
		return
	}

	if len(sessions) == 0 {
		color.Yellow("○ No active shared sessions")
	} else {
		color.Green("✓ Active shared sessions (%d):", len(sessions))
		for _, session := range sessions {
			startTime := time.Unix(session.Started, 0)
			duration := time.Since(startTime).Round(time.Second)
			
			fmt.Printf("\n")
			fmt.Printf("  Session: %s\n", session.Name)
			fmt.Printf("  Port: %d\n", session.Port)
			fmt.Printf("  Started: %s (%s ago)\n", startTime.Format("15:04:05"), duration)
			
			if session.Private {
				color.Red("  Private session")
				if len(session.AllowedUsers) > 0 {
					fmt.Printf("  Allowed users: %v\n", session.AllowedUsers)
				}
			} else {
				color.Green("  Public session")
			}
		}
	}

	fmt.Println()
	color.Blue("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	color.Cyan("Commands:")
	fmt.Println("  jmux share     - Share current/new session")
	fmt.Println("  jmux sessions  - List all shared sessions")
	fmt.Println("  jmux stop      - Stop sharing sessions")
	fmt.Println("  jmux ls        - List tmux sessions")
}