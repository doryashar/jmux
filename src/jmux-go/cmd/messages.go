package cmd

import (
	"github.com/spf13/cobra"
	"jmux/internal/messaging"
)

// messagesCmd represents the messages command
var messagesCmd = &cobra.Command{
	Use:   "messages",
	Short: "Read new messages",
	Long:  `Display and clear new messages from other users.`,
	Run: func(cmd *cobra.Command, args []string) {
		err := msgSystem.ReadMessages()
		if err != nil {
			cmd.Printf("Error reading messages: %v\n", err)
			return
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