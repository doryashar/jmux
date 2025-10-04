package cmd

import (
	"fmt"
	"os"

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
	return rootCmd.Execute()
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
		if err := monitorMgr.StartMonitor(); err != nil {
			color.Yellow("Warning: Could not start messaging monitor: %v", err)
		}
	}

	// Initialize managers
	sessMgr = session.NewManager(cfg, msgSystem)
	tmuxMgr = tmux.NewManager()

	// Register user in database
	registerCurrentUser()

	// Check for updates if needed (runs in background)
	updater.CheckForUpdatesIfNeeded(cfg.ConfigDir)
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

// registerCurrentUser registers the current user in the database
func registerCurrentUser() {
	currentUser := os.Getenv("USER")
	if currentUser == "" {
		return
	}

	// Get current IP (simplified - could be enhanced)
	hostname, _ := os.Hostname()
	
	// Write to users file
	userEntry := fmt.Sprintf("%s:%s\n", currentUser, hostname)
	
	file, err := os.OpenFile(cfg.UsersFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer file.Close()
	
	file.WriteString(userEntry)
}