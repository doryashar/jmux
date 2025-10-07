package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"jmux/internal/messaging"
)

// messagesCmd represents the messages command
var messagesCmd = &cobra.Command{
	Use:   "messages",
	Short: "Read new messages",
	Long:  `Display and clear new messages from other users.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Read messages from both sources - consolidated .messages file and individual .msg files
		hasMessages := false
		
		// First, read from consolidated .messages file
		err := msgSystem.ReadMessages()
		if err != nil {
			// Don't fail completely, just note the error
			fmt.Printf("Warning: Error reading consolidated messages: %v\n", err)
		} else {
			// Check if there were messages in the consolidated file
			// (ReadMessages prints "No new messages" if empty)
		}
		
		// Second, read from individual .msg files (reverse invitations, etc.)
		currentUser := os.Getenv("USER")
		if currentUser != "" {
			msgFileMessages := readIndividualMessageFiles(currentUser)
			if len(msgFileMessages) > 0 {
				hasMessages = true
				fmt.Printf("📨 You have %d message(s) from individual files:\n", len(msgFileMessages))
				
				for _, msg := range msgFileMessages {
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
				
				// Clean up individual message files after reading
				cleanupIndividualMessageFiles(currentUser)
				color.Green("✅ Individual message files cleared")
			}
		}
		
		if !hasMessages {
			// Only show this if we didn't find messages in either source
			// (but avoid double "No new messages" if msgSystem.ReadMessages already printed it)
		}
	},
}

// msgCmd represents the msg command for sending messages
var msgCmd = &cobra.Command{
	Use:   "msg <user> <message>",
	Short: "Send a message to another user",
	Long: `Send a message to another user.

Examples:
  jmux msg alice "Hello there!"
  jmux msg bob "Meeting in 5 minutes"`,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		toUser := args[0]
		message := args[1]
		
		err := msgSystem.SendMessage(toUser, messaging.MessageTypeMessage, message)
		if err != nil {
			cmd.Printf("Error sending message: %v\n", err)
			return
		}
		
		cmd.Printf("Message sent to %s\n", toUser)
	},
}

func init() {
	rootCmd.AddCommand(messagesCmd)
	rootCmd.AddCommand(msgCmd)
}

// readIndividualMessageFiles reads individual .msg files (used by reverse sharing)
func readIndividualMessageFiles(currentUser string) []messageData {
	var messages []messageData
	
	// Check for message files
	pattern := currentUser + "_*.msg"
	matches, err := filepath.Glob(filepath.Join(cfg.MessagesDir, pattern))
	if err != nil {
		return messages
	}
	
	// Read and parse each message file
	for _, msgFile := range matches {
		msg, err := readMessageFile(msgFile)
		if err != nil {
			continue
		}
		messages = append(messages, *msg)
	}
	
	return messages
}

// cleanupIndividualMessageFiles removes individual .msg files after reading
func cleanupIndividualMessageFiles(currentUser string) {
	pattern := currentUser + "_*.msg"
	matches, err := filepath.Glob(filepath.Join(cfg.MessagesDir, pattern))
	if err != nil {
		return
	}
	
	for _, msgFile := range matches {
		os.Remove(msgFile)
	}
}

// Functions readMessageFile and messageData are defined in status.go