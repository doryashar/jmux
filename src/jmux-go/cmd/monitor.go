package cmd

import (
	"fmt"
	"os"
	"os/exec"
	
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"jmux/internal/config"
	"jmux/internal/messaging"
)

// monitorCmd represents the monitor command
var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Manage messaging monitor",
	Long:  `Manage the centralized messaging monitor daemon.`,
}

// monitorStatusCmd shows monitor status
var monitorStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show monitor status",
	Long:  `Show the status of the messaging monitor daemon.`,
	Run: func(cmd *cobra.Command, args []string) {
		ensureMonitorManager()
		status := monitorMgr.GetMonitorStatus()
		color.Blue("Messaging Monitor Status: %s", status)
		
		if monitorMgr.IsMonitorRunning() {
			color.Green("✅ Monitor is active and ready to receive messages")
		} else {
			color.Yellow("⚠️  Monitor is not running - messages will not be displayed in real-time")
			color.Blue("💡 Start with: dmux monitor start")
		}
	},
}

// monitorStartCmd starts the monitor
var monitorStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start monitor",
	Long:  `Start the messaging monitor daemon.`,
	Run: func(cmd *cobra.Command, args []string) {
		ensureMonitorManager()
		if monitorMgr.IsMonitorRunning() {
			color.Yellow("Monitor is already running")
			return
		}
		
		if err := monitorMgr.StartMonitor(); err != nil {
			color.Red("Failed to start monitor: %v", err)
			return
		}
		
		color.Green("✅ Messaging monitor started successfully")
	},
}

// monitorStopCmd stops the monitor
var monitorStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop monitor",
	Long:  `Stop the messaging monitor daemon.`,
	Run: func(cmd *cobra.Command, args []string) {
		ensureMonitorManager()
		if !monitorMgr.IsMonitorRunning() {
			color.Yellow("Monitor is not running")
			return
		}
		
		if err := monitorMgr.StopMonitor(); err != nil {
			color.Red("Failed to stop monitor: %v", err)
			return
		}
		
		color.Green("✅ Messaging monitor stopped successfully")
	},
}

// monitorRestartCmd restarts the monitor
var monitorRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart monitor",
	Long:  `Restart the messaging monitor daemon.`,
	Run: func(cmd *cobra.Command, args []string) {
		ensureMonitorManager()
		color.Blue("Restarting messaging monitor...")
		
		if err := monitorMgr.RestartMonitor(); err != nil {
			color.Red("Failed to restart monitor: %v", err)
			return
		}
		
		color.Green("✅ Messaging monitor restarted successfully")
	},
}

// ensureMonitorManager creates a minimal monitor manager for monitor commands
func ensureMonitorManager() {
	if monitorMgr == nil {
		cfg = config.DefaultConfig()
		monitorMgr = messaging.NewMonitorManager(cfg)
	}
}

// monitorLogsCmd shows monitor logs
var monitorLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Show monitor logs",
	Long:  `Display the messaging monitor log file.`,
	Run: func(cmd *cobra.Command, args []string) {
		ensureMonitorManager()
		logFile := cfg.MonitorLogFile
		
		// Check if log file exists
		if _, err := os.Stat(logFile); os.IsNotExist(err) {
			color.Yellow("No log file found at: %s", logFile)
			color.Blue("💡 Start the monitor to begin logging: dmux monitor start")
			return
		}
		
		// Get number of lines to show
		lines, _ := cmd.Flags().GetInt("lines")
		follow, _ := cmd.Flags().GetBool("follow")
		
		if follow {
			color.Blue("Following monitor logs (Ctrl+C to exit):")
			color.Blue("Log file: %s", logFile)
			fmt.Println()
			
			// Use tail -f equivalent
			execCmd := exec.Command("tail", "-f", logFile)
			execCmd.Stdout = os.Stdout
			execCmd.Stderr = os.Stderr
			execCmd.Run()
		} else {
			color.Blue("Monitor logs (%d lines):", lines)
			color.Blue("Log file: %s", logFile)
			fmt.Println()
			
			// Use tail to show last N lines
			execCmd := exec.Command("tail", "-n", fmt.Sprintf("%d", lines), logFile)
			execCmd.Stdout = os.Stdout
			execCmd.Stderr = os.Stderr
			if err := execCmd.Run(); err != nil {
				color.Red("Failed to read log file: %v", err)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(monitorCmd)
	monitorCmd.AddCommand(monitorStatusCmd)
	monitorCmd.AddCommand(monitorStartCmd)
	monitorCmd.AddCommand(monitorStopCmd)
	monitorCmd.AddCommand(monitorRestartCmd)
	monitorCmd.AddCommand(monitorLogsCmd)
	
	// Add flags for logs command
	monitorLogsCmd.Flags().IntP("lines", "n", 50, "Number of lines to show")
	monitorLogsCmd.Flags().BoolP("follow", "f", false, "Follow log file (like tail -f)")
}