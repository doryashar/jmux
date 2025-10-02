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
)

var (
	cfg       *config.Config
	msgSystem *messaging.Messaging
	sessMgr   *session.Manager
	tmuxMgr   *tmux.Manager
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "jmux",
	Short: "Tmux Session Sharing Made Easy",
	Long: `jmux is an enhanced tmux session sharing tool with real-time messaging, 
live monitoring, and built-in networking capabilities.

Features:
- Share tmux sessions with simple commands
- Real-time messaging and notifications
- Private sessions with access control
- Built-in jcat networking (no socat dependency)
- Live session monitoring`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Skip initialization for help and completion commands
		if cmd.Name() == "help" || cmd.Name() == "completion" {
			return
		}
		initializeSystem()
	},
	Run: func(cmd *cobra.Command, args []string) {
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
	// Add global flags here if needed
}

// initializeSystem initializes the jmux system
func initializeSystem() {
	cfg = config.DefaultConfig()
	
	// Ensure directories exist
	if err := cfg.EnsureDirectories(); err != nil {
		color.Red("Error creating directories: %v", err)
		os.Exit(1)
	}

	// Initialize messaging system
	msgSystem = messaging.NewMessaging(cfg)
	
	// Start live monitoring in background
	if err := msgSystem.StartLiveMonitoring(); err != nil {
		color.Yellow("Warning: Could not start live monitoring: %v", err)
	}

	// Initialize managers
	sessMgr = session.NewManager(cfg, msgSystem)
	tmuxMgr = tmux.NewManager()

	// Register user in database
	registerCurrentUser()
}

// startRegularSession starts a regular tmux session
func startRegularSession() {
	if err := tmuxMgr.StartRegularSession(); err != nil {
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