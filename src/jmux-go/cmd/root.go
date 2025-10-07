package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"jmux/internal/config"
	"jmux/internal/messaging"
	"jmux/internal/session"
	"jmux/internal/tmux"
	"jmux/internal/updater"
	"jmux/internal/version"
)

var (
	cfg        *config.Config
	msgSystem  *messaging.Messaging
	monitorMgr *messaging.MonitorManager
	sessMgr    *session.Manager
	tmuxMgr    *tmux.Manager
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "dmux",
	Short: "Tmux Session Sharing Made Easy",
	Long: `dmux is an enhanced tmux session sharing tool with real-time messaging, 
live monitoring, and built-in networking capabilities.

Features:
- Share tmux sessions with simple commands
- Real-time messaging and notifications
- Private sessions with access control
- Built-in jcat networking (no socat dependency)
- Live session monitoring`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Skip initialization for commands that don't need the full system
		skipInit := cmd.Name() == "help" || 
				   cmd.Name() == "completion" || 
				   cmd.Name() == "version" ||
				   cmd.Name() == "monitor" ||  // Skip for monitor command itself
				   (cmd.Parent() != nil && cmd.Parent().Name() == "monitor") // Skip for monitor subcommands
		
		// Also skip if --version flag is set on root command
		if versionFlag, _ := cmd.Flags().GetBool("version"); versionFlag {
			skipInit = true
		}
		
		if skipInit {
			return
		}
		
		initializeSystem()
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Check for version flag first
		if versionFlag, _ := cmd.Flags().GetBool("version"); versionFlag {
			fmt.Println(version.GetVersion())
			return
		}
		
		// If no subcommand, start a regular tmux session
		startRegularSession()
	},
	// Handle unknown commands as tmux passthrough
	SilenceUsage: true,
	FParseErrWhitelist: cobra.FParseErrWhitelist{
		UnknownFlags: true,
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	// Set up unknown command handler for tmux passthrough
	rootCmd.SetArgs(os.Args[1:])
	
	// Temporarily disable error output for unknown commands
	originalSilenceErrors := rootCmd.SilenceErrors
	originalSilenceUsage := rootCmd.SilenceUsage
	
	// Try to execute the command with silenced errors
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true
	err := rootCmd.Execute()
	
	// Restore original settings
	rootCmd.SilenceErrors = originalSilenceErrors
	rootCmd.SilenceUsage = originalSilenceUsage
	
	// If we get a "unknown command" error, try tmux passthrough
	if err != nil && isUnknownCommandError(err) {
		// Initialize system for monitoring before passthrough
		if cfg == nil {
			initializeSystem()
		}
		
		// Pass through to tmux (no error message shown)
		return handleTmuxPassthrough(os.Args[1:])
	}
	
	// For other errors, show them normally
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
	
	return err
}

// isUnknownCommandError checks if the error is due to an unknown command
func isUnknownCommandError(err error) bool {
	return err != nil && 
		   (strings.Contains(err.Error(), "unknown command") ||
		    strings.Contains(err.Error(), "invalid command"))
}

// handleTmuxPassthrough passes unknown commands through to tmux
func handleTmuxPassthrough(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command provided")
	}
	
	// Check if tmux is available
	if tmuxMgr == nil {
		tmuxMgr = tmux.NewManager()
	}
	
	if !tmuxMgr.IsTmuxAvailable() {
		return fmt.Errorf("tmux is not available. Please install tmux first")
	}
	
	// Silently pass through to tmux without notification
	
	// Use the tmux manager to run the command
	return tmuxMgr.RunTmuxCommand(args)
}

func init() {
	// Add version flag to root command
	rootCmd.Flags().BoolP("version", "V", false, "Show version information")
}

// initializeSystem initializes the jmux system
func initializeSystem() {
	cfg = config.DefaultConfig()
	
	// Ensure directories exist
	if err := cfg.EnsureDirectories(); err != nil {
		color.Red("Error creating directories: %v", err)
		os.Exit(1)
	}
	
	// Ensure setsize script exists and is current
	if err := cfg.EnsureSetSizeScript(); err != nil {
		color.Red("Error creating setsize script: %v", err)
		os.Exit(1)
	}

	// Initialize messaging system
	msgSystem = messaging.NewMessaging(cfg)
	
	// Initialize monitor manager
	monitorMgr = messaging.NewMonitorManager(cfg)
	
	// Start centralized monitor if realtime is enabled and monitor not already running
	if cfg.RealtimeEnabled && !monitorMgr.IsMonitorRunning() {
		// Start monitor asynchronously to avoid blocking main execution
		go func() {
			if err := monitorMgr.StartMonitor(); err != nil {
				color.Yellow("Warning: Could not start messaging monitor: %v", err)
			}
		}()
	}

	// Initialize managers
	sessMgr = session.NewManager(cfg, msgSystem)
	tmuxMgr = tmux.NewManager()

	// Register user in database
	registerCurrentUser()

	// Check for updates if needed (skip in tmux to avoid hanging)
	if !tmuxMgr.IsInTmuxSession() {
		updater.CheckForUpdatesIfNeeded(cfg.ConfigDir, cfg.AutoUpdate)
	}
}

// startRegularSession starts a regular tmux session with messaging
func startRegularSession() {
	// If already in tmux, just show status and exit
	if tmuxMgr.IsInTmuxSession() {
		color.Yellow("Already in a tmux session with real-time messaging active")
		color.Blue("💡 Messages will appear automatically")
		
		// Show monitor status
		if monitorMgr != nil && monitorMgr.IsMonitorRunning() {
			color.Green("✅ Messaging monitor is running")
		} else {
			color.Yellow("⚠️  Messaging monitor is not running")
			color.Blue("   Start with: dmux monitor start")
		}
		return
	}
	
	if err := tmuxMgr.StartRegularSessionWithMessaging(); err != nil {
		color.Red("Error starting tmux session: %v", err)
		os.Exit(1)
	}
}

// registerCurrentUser registers the current user in the database (prevents duplicates)
func registerCurrentUser() {
	currentUser := os.Getenv("USER")
	if currentUser == "" {
		return
	}

	// Get current IP (simplified - could be enhanced)
	hostname, _ := os.Hostname()
	userEntry := fmt.Sprintf("%s:%s", currentUser, hostname)
	
	// Read existing entries to check for duplicates
	existingEntries := readUsersFile()
	
	// First, remove any existing entries for this user and check if exact entry exists
	var filteredEntries []string
	exactEntryExists := false
	
	for _, entry := range existingEntries {
		entryTrimmed := strings.TrimSpace(entry)
		parts := strings.Split(entryTrimmed, ":")
		
		if len(parts) >= 1 && parts[0] == currentUser {
			// This entry is for the current user
			if entryTrimmed == userEntry {
				// Exact entry already exists
				exactEntryExists = true
			}
			// Skip all entries for this user (we'll add the current one back)
			continue
		}
		filteredEntries = append(filteredEntries, entry)
	}
	
	// If exact entry already exists, no need to update
	if exactEntryExists && len(filteredEntries) == len(existingEntries)-1 {
		return
	}
	
	// Add the new/updated entry
	filteredEntries = append(filteredEntries, userEntry)
	
	// Write back all entries
	writeUsersFile(filteredEntries)
}

// readUsersFile reads all entries from users.db
func readUsersFile() []string {
	content, err := os.ReadFile(cfg.UsersFile)
	if err != nil {
		return []string{} // File doesn't exist yet
	}
	
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	var entries []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			entries = append(entries, line)
		}
	}
	return entries
}

// writeUsersFile writes all entries to users.db
func writeUsersFile(entries []string) {
	content := strings.Join(entries, "\n")
	if content != "" {
		content += "\n"
	}
	
	err := os.WriteFile(cfg.UsersFile, []byte(content), 0644)
	if err != nil {
		color.Yellow("Warning: Could not update users database: %v", err)
	}
}

