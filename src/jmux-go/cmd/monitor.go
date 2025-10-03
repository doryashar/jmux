package cmd

import (
	"github.com/fatih/color"
	"github.com/spf13/cobra"
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
		color.Blue("Restarting messaging monitor...")
		
		if err := monitorMgr.RestartMonitor(); err != nil {
			color.Red("Failed to restart monitor: %v", err)
			return
		}
		
		color.Green("✅ Messaging monitor restarted successfully")
	},
}

func init() {
	rootCmd.AddCommand(monitorCmd)
	monitorCmd.AddCommand(monitorStatusCmd)
	monitorCmd.AddCommand(monitorStartCmd)
	monitorCmd.AddCommand(monitorStopCmd)
	monitorCmd.AddCommand(monitorRestartCmd)
}