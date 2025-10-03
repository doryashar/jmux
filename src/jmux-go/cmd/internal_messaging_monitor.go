package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"jmux/internal/config"
	"jmux/internal/messaging"
)

// internalMessagingMonitorCmd is a hidden command to run messaging monitor inside tmux
var internalMessagingMonitorCmd = &cobra.Command{
	Use:    "_internal_messaging_monitor",
	Short:  "Internal command to run messaging monitor",
	Long:   `Internal command used to run messaging monitor inside tmux sessions. Not for direct use.`,
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		// This is the centralized messaging monitor daemon
		if os.Getenv("DMUX_DEBUG") != "" {
			color.Blue("Starting centralized messaging monitor daemon...")
		}

		// Initialize configuration
		cfg := config.DefaultConfig()
		
		// Ensure directories exist
		if err := cfg.EnsureDirectories(); err != nil {
			color.Red("Error creating directories: %v", err)
			os.Exit(1)
		}

		// Initialize messaging system
		msgSystem := messaging.NewMessaging(cfg)
		
		// Start live monitoring
		if err := msgSystem.StartLiveMonitoring(); err != nil {
			color.Red("Error starting live monitoring: %v", err)
			os.Exit(1)
		}

		if os.Getenv("DMUX_DEBUG") != "" {
			color.Green("Messaging monitor started successfully")
		}

		// Set up signal handling for graceful shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		// Wait for termination signal
		<-sigChan
		
		if os.Getenv("DMUX_DEBUG") != "" {
			color.Yellow("Shutting down messaging monitor...")
		}
		
		// Stop monitoring
		msgSystem.StopLiveMonitoring()
		
		if os.Getenv("DMUX_DEBUG") != "" {
			color.Green("Messaging monitor stopped")
		}
	},
}

func init() {
	rootCmd.AddCommand(internalMessagingMonitorCmd)
}