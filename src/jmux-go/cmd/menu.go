package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// menuCmd represents the unified menu command
var menuCmd = &cobra.Command{
	Use:   "menu",
	Short: "Show dmux menu (role-based)",
	Long:  "Display menu options based on user role in current session",
	Hidden: true, // Hidden from help since it's triggered by key binding
	Run: showRoleBasedMenu,
}

func init() {
	rootCmd.AddCommand(menuCmd)
}

func showRoleBasedMenu(cmd *cobra.Command, args []string) {
	// Determine user role in current context
	role := determineUserRole()
	
	switch role {
	case "host":
		showHostMenu()
	case "client":
		showClientMenu()
	default:
		showGeneralMenu()
	}
}

func determineUserRole() string {
	// Check if we're in a tmux session
	if !tmuxMgr.IsInTmuxSession() {
		return "general"
	}
	
	// Get current session name
	currentSession, err := tmuxMgr.GetCurrentSession()
	if err != nil {
		return "general"
	}
	
	// Check if this session is being shared by current user
	currentUser := os.Getenv("USER")
	if currentUser == "" {
		return "general"
	}
	
	// Look for active sessions owned by current user with same name
	sessions, err := sessMgr.ListUserSessions(currentUser)
	if err != nil {
		return "general"
	}
	
	for _, session := range sessions {
		if session.Name == currentSession {
			return "host" // User is hosting this session
		}
	}
	
	// Check if we're connected as a client to someone else's session
	// This would require checking if we joined via dmux join command
	// For now, assume client if in tmux but not hosting
	return "client"
}

func showHostMenu() {
	clearScreen()
	color.Blue("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	color.Green("🔧 Host Management Menu")
	color.Blue("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	
	currentSession, _ := tmuxMgr.GetCurrentSession()
	fmt.Printf("Session: %s | Role: Host\n\n", currentSession)
	
	menu := []MenuOption{
		{Key: "1", Label: "👥 List Connected Users", Action: "list-users"},
		{Key: "2", Label: "🚫 Kick User", Action: "kick-user"},
		{Key: "3", Label: "⛔ Ban User", Action: "ban-user"},
		{Key: "4", Label: "🔄 Change Session Mode", Action: "change-mode"},
		{Key: "5", Label: "🔒 Session Settings", Action: "session-settings"},
		{Key: "6", Label: "👑 Transfer Ownership", Action: "transfer-ownership"},
		{Key: "7", Label: "📊 Session Statistics", Action: "session-stats"},
		{Key: "8", Label: "📝 Change Display Name", Action: "change-name"},
		{Key: "9", Label: "💬 Send Message", Action: "send-message"},
		{Key: "s", Label: "🛑 Stop Sharing", Action: "stop-sharing"},
		{Key: "h", Label: "❓ Help", Action: "help"},
		{Key: "q", Label: "❌ Close Menu", Action: "quit"},
	}
	
	handleMenuSelection(menu, "host")
}

func showClientMenu() {
	clearScreen()
	color.Blue("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	color.Cyan("👤 Client Menu")
	color.Blue("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	
	currentSession, _ := tmuxMgr.GetCurrentSession()
	fmt.Printf("Session: %s | Role: Client\n\n", currentSession)
	
	menu := []MenuOption{
		{Key: "1", Label: "📝 Change Display Name", Action: "change-name"},
		{Key: "2", Label: "👥 View Connected Users", Action: "list-users"},
		{Key: "3", Label: "📊 Session Info", Action: "session-info"},
		{Key: "4", Label: "💬 Send Message", Action: "send-message"},
		{Key: "5", Label: "🙋 Request Permission", Action: "request-permission"},
		{Key: "6", Label: "🚪 Leave Session", Action: "leave-session"},
		{Key: "7", Label: "⚠️  Report Issue", Action: "report-issue"},
		{Key: "h", Label: "❓ Help", Action: "help"},
		{Key: "q", Label: "❌ Close Menu", Action: "quit"},
	}
	
	handleMenuSelection(menu, "client")
}

func showGeneralMenu() {
	clearScreen()
	color.Blue("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	color.Yellow("📱 dmux Menu")
	color.Blue("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("General dmux operations\n\n")
	
	menu := []MenuOption{
		{Key: "1", Label: "🚀 Start Sharing Session", Action: "start-sharing"},
		{Key: "2", Label: "📋 Show Status", Action: "show-status"},
		{Key: "3", Label: "📁 List Sessions", Action: "list-sessions"},
		{Key: "4", Label: "🔗 Join Session", Action: "join-session"},
		{Key: "5", Label: "💬 Check Messages", Action: "check-messages"},
		{Key: "6", Label: "🧹 Cleanup", Action: "cleanup"},
		{Key: "h", Label: "❓ Help", Action: "help"},
		{Key: "q", Label: "❌ Close Menu", Action: "quit"},
	}
	
	handleMenuSelection(menu, "general")
}

type MenuOption struct {
	Key    string
	Label  string
	Action string
}

func handleMenuSelection(menu []MenuOption, userRole string) {
	// Display menu options
	for _, option := range menu {
		fmt.Printf("  [%s] %s\n", option.Key, option.Label)
	}
	
	fmt.Print("\nSelect option: ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	choice := strings.TrimSpace(input)
	
	// Find selected option
	var selectedAction string
	for _, option := range menu {
		if option.Key == choice {
			selectedAction = option.Action
			break
		}
	}
	
	if selectedAction == "" {
		color.Red("Invalid selection")
		pressAnyKey()
		return
	}
	
	// Execute action
	executeMenuAction(selectedAction, userRole)
}

func executeMenuAction(action string, userRole string) {
	clearScreen()
	
	switch action {
	case "quit":
		return
		
	case "list-users":
		showConnectedUsers()
		
	case "kick-user":
		if userRole != "host" {
			color.Red("❌ Host privileges required")
			pressAnyKey()
			return
		}
		kickUser()
		
	case "ban-user":
		if userRole != "host" {
			color.Red("❌ Host privileges required")
			pressAnyKey()
			return
		}
		banUser()
		
	case "change-mode":
		if userRole != "host" {
			color.Red("❌ Host privileges required")
			pressAnyKey()
			return
		}
		changeSessionMode()
		
	case "session-settings":
		if userRole != "host" {
			color.Red("❌ Host privileges required")
			pressAnyKey()
			return
		}
		showSessionSettings()
		
	case "transfer-ownership":
		if userRole != "host" {
			color.Red("❌ Host privileges required")
			pressAnyKey()
			return
		}
		transferOwnership()
		
	case "change-name":
		changeDisplayName()
		
	case "send-message":
		sendMessage()
		
	case "session-info":
		showSessionInfo()
		
	case "session-stats":
		if userRole != "host" {
			color.Red("❌ Host privileges required")
			pressAnyKey()
			return
		}
		showSessionStats()
		
	case "request-permission":
		if userRole == "host" {
			color.Yellow("You already have host privileges")
			pressAnyKey()
			return
		}
		requestPermission()
		
	case "leave-session":
		leaveSession()
		
	case "report-issue":
		reportIssue()
		
	case "stop-sharing":
		if userRole != "host" {
			color.Red("❌ Host privileges required")
			pressAnyKey()
			return
		}
		stopSharing()
		
	case "start-sharing":
		startSharing()
		
	case "show-status":
		showStatus()
		pressAnyKey()
		
	case "list-sessions":
		listSessions()
		
	case "join-session":
		joinSession()
		
	case "check-messages":
		checkMessages()
		pressAnyKey()
		
	case "cleanup":
		cleanupCmd.Run(nil, []string{})
		pressAnyKey()
		
	case "help":
		showRoleHelp(userRole)
		
	default:
		color.Red("Unknown action: %s", action)
		pressAnyKey()
	}
}

// Utility functions
func clearScreen() {
	fmt.Print("\033[2J\033[H")
}

func pressAnyKey() {
	fmt.Print("\nPress Enter to continue...")
	bufio.NewReader(os.Stdin).ReadLine()
}

// Menu action implementations (simplified for now)
func showConnectedUsers() {
	color.Green("👥 Connected Users")
	color.Blue("━━━━━━━━━━━━━━━━━")
	// Implementation would show actual connected users
	color.Yellow("Feature in development - use 'dmux status' for now")
	pressAnyKey()
}

func kickUser() {
	color.Green("🚫 Kick User")
	color.Blue("━━━━━━━━━━━━")
	color.Yellow("Feature in development")
	pressAnyKey()
}

func banUser() {
	color.Green("⛔ Ban User")
	color.Blue("━━━━━━━━━━━")
	color.Yellow("Feature in development")
	pressAnyKey()
}

func changeSessionMode() {
	color.Green("🔄 Change Session Mode")
	color.Blue("━━━━━━━━━━━━━━━━━━━━")
	color.Yellow("Feature in development")
	pressAnyKey()
}

func showSessionSettings() {
	color.Green("🔒 Session Settings")
	color.Blue("━━━━━━━━━━━━━━━━━━")
	color.Yellow("Feature in development")
	pressAnyKey()
}

func transferOwnership() {
	color.Green("👑 Transfer Ownership")
	color.Blue("━━━━━━━━━━━━━━━━━━━")
	color.Yellow("Feature in development")
	pressAnyKey()
}

func changeDisplayName() {
	color.Green("📝 Change Display Name")
	color.Blue("━━━━━━━━━━━━━━━━━━━━━")
	color.Yellow("Feature in development")
	pressAnyKey()
}

func sendMessage() {
	color.Green("💬 Send Message")
	color.Blue("━━━━━━━━━━━━━━━")
	color.Yellow("Feature in development")
	pressAnyKey()
}

func showSessionInfo() {
	color.Green("📊 Session Information")
	color.Blue("━━━━━━━━━━━━━━━━━━━━━")
	
	if tmuxMgr.IsInTmuxSession() {
		sessionName, _ := tmuxMgr.GetCurrentSession()
		fmt.Printf("Current session: %s\n", sessionName)
	}
	
	// Show dmux status
	showStatus()
	pressAnyKey()
}

func showSessionStats() {
	color.Green("📊 Session Statistics")
	color.Blue("━━━━━━━━━━━━━━━━━━━━")
	color.Yellow("Feature in development")
	pressAnyKey()
}

func requestPermission() {
	color.Green("🙋 Request Permission")
	color.Blue("━━━━━━━━━━━━━━━━━━━")
	color.Yellow("Feature in development")
	pressAnyKey()
}

func leaveSession() {
	color.Green("🚪 Leave Session")
	color.Blue("━━━━━━━━━━━━━━━")
	color.Yellow("Exiting tmux session...")
	// This would disconnect from the session
	os.Exit(0)
}

func reportIssue() {
	color.Green("⚠️  Report Issue")
	color.Blue("━━━━━━━━━━━━━━━")
	color.Yellow("Feature in development")
	pressAnyKey()
}

func stopSharing() {
	color.Green("🛑 Stop Sharing")
	color.Blue("━━━━━━━━━━━━━━")
	
	// Get current session and stop sharing
	currentSession, err := tmuxMgr.GetCurrentSession()
	if err != nil {
		color.Red("❌ Could not determine current session")
		pressAnyKey()
		return
	}
	
	// Find and stop the session
	currentUser := os.Getenv("USER")
	sessions, err := sessMgr.ListUserSessions(currentUser)
	if err != nil {
		color.Red("❌ Could not list sessions: %v", err)
		pressAnyKey()
		return
	}
	
	for _, session := range sessions {
		if session.Name == currentSession {
			err := sessMgr.StopShare([]string{session.Name})
			if err != nil {
				color.Red("❌ Failed to stop sharing: %v", err)
			} else {
				color.Green("✅ Sharing stopped for session '%s'", session.Name)
			}
			pressAnyKey()
			return
		}
	}
	
	color.Yellow("No active sharing found for current session")
	pressAnyKey()
}

func startSharing() {
	color.Green("🚀 Start Sharing")
	color.Blue("━━━━━━━━━━━━━━━")
	
	if tmuxMgr.IsInTmuxSession() {
		sessionName, _ := tmuxMgr.GetCurrentSession()
		color.Blue("Starting sharing for session: %s", sessionName)
		
		err := sessMgr.StartShare(sessionName, false, []string{}, "pair")
		if err != nil {
			color.Red("❌ Failed to start sharing: %v", err)
		} else {
			color.Green("✅ Session sharing started")
		}
	} else {
		color.Yellow("Not in a tmux session - use 'dmux share' to start sharing")
	}
	
	pressAnyKey()
}

func listSessions() {
	color.Green("📁 Sessions")
	color.Blue("━━━━━━━━━━━━")
	tmuxMgr.ListSessions()
	pressAnyKey()
}

func joinSession() {
	color.Green("🔗 Join Session")
	color.Blue("━━━━━━━━━━━━━━━")
	color.Yellow("Use 'dmux join <user> <session>' from command line")
	pressAnyKey()
}

func showRoleHelp(userRole string) {
	clearScreen()
	color.Blue("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	color.Green("📖 dmux Help - %s", strings.Title(userRole))
	color.Blue("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	
	switch userRole {
	case "host":
		fmt.Println("As the session host, you can:")
		fmt.Println("• Manage users (kick, ban)")
		fmt.Println("• Change session settings")
		fmt.Println("• Transfer ownership")
		fmt.Println("• View detailed statistics")
		fmt.Println("• Stop sharing")
		fmt.Println("• All client capabilities")
		
	case "client":
		fmt.Println("As a client, you can:")
		fmt.Println("• Change your display name")
		fmt.Println("• View session information")
		fmt.Println("• Send messages to other users")
		fmt.Println("• Request additional permissions")
		fmt.Println("• Leave the session")
		
	case "general":
		fmt.Println("General dmux operations:")
		fmt.Println("• Start sharing sessions")
		fmt.Println("• Join existing sessions")
		fmt.Println("• Check session status")
		fmt.Println("• Manage messages")
		fmt.Println("• System cleanup")
	}
	
	fmt.Println("\nKey Binding: Ctrl+A + M")
	fmt.Println("Commands: Use 'dmux --help' for full command list")
	
	pressAnyKey()
}