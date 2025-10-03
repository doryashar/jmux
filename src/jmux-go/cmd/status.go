package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show dmux sharing status",
	Long:  `Show the current sharing status including active sessions and connection info.`,
	Run: func(cmd *cobra.Command, args []string) {
		showStatus()
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func showStatus() {
	// Clean up stale sessions first
	cleaned := performCleanup()
	if cleaned > 0 {
		color.Yellow("🧹 Cleaned up %d stale session(s)", cleaned)
		fmt.Println()
	}

	// Check for new messages
	checkMessages()

	currentUser := os.Getenv("USER")
	if currentUser == "" {
		color.Red("❌ Unable to determine current user")
		return
	}

	color.Blue("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	color.Green("DMUX Sharing Status")
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
	fmt.Println("  dmux share     - Share current/new session")
	fmt.Println("  dmux sessions  - List all shared sessions")
	fmt.Println("  dmux stop      - Stop sharing sessions")
	fmt.Println("  dmux ls        - List tmux sessions")
}

// checkMessages checks for new messages and displays them
func checkMessages() {
	if msgSystem == nil {
		return
	}

	currentUser := os.Getenv("USER")
	if currentUser == "" {
		return
	}

	// Check for message files
	pattern := currentUser + "_*.msg"
	matches, err := filepath.Glob(filepath.Join(cfg.MessagesDir, pattern))
	if err != nil {
		return
	}

	if len(matches) == 0 {
		return
	}

	color.Yellow("📨 You have %d new message(s):", len(matches))
	
	// Display brief summary of messages
	for _, msgFile := range matches {
		msg, err := readMessageFile(msgFile)
		if err != nil {
			continue
		}
		
		timeStr := time.Unix(msg.Timestamp, 0).Format("15:04")
		switch msg.Type {
		case "INVITE":
			color.Cyan("  %s - INVITE from %s: %s", timeStr, msg.From, msg.Data)
		case "URGENT":
			color.Red("  %s - URGENT from %s: %s", timeStr, msg.From, msg.Data)
		default:
			color.White("  %s - Message from %s: %s", timeStr, msg.From, msg.Data)
		}
	}
	
	color.Blue("💡 Run 'dmux messages' to read and clear messages")
	fmt.Println()
}

// readMessageFile reads a message from file (simplified version)
func readMessageFile(filePath string) (*messageData, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	msg := &messageData{}
	lines := strings.Split(string(content), "\n")
	
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key, value := parts[0], parts[1]
		switch key {
		case "FROM":
			msg.From = value
		case "TYPE":
			msg.Type = value
		case "TIMESTAMP":
			if ts, err := strconv.ParseInt(value, 10, 64); err == nil {
				msg.Timestamp = ts
			}
		case "DATA":
			msg.Data = value
		}
	}

	return msg, nil
}

// messageData represents basic message information for status
type messageData struct {
	From      string
	Type      string
	Timestamp int64
	Data      string
}