package messaging

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"jmux/internal/config"
)

// MonitorManager handles centralized messaging monitor lifecycle
type MonitorManager struct {
	config *config.Config
}

// NewMonitorManager creates a new monitor manager
func NewMonitorManager(cfg *config.Config) *MonitorManager {
	return &MonitorManager{config: cfg}
}

// IsMonitorRunning checks if a monitor is already running
func (mm *MonitorManager) IsMonitorRunning() bool {
	pidFile := mm.config.MonitorPIDFile
	
	// Check if PID file exists
	if _, err := os.Stat(pidFile); os.IsNotExist(err) {
		return false
	}
	
	// Read PID from file
	pidBytes, err := os.ReadFile(pidFile)
	if err != nil {
		// If we can't read the PID file, assume no monitor is running
		os.Remove(pidFile) // Clean up invalid PID file
		return false
	}
	
	pid, err := strconv.Atoi(string(pidBytes))
	if err != nil {
		// Invalid PID in file
		os.Remove(pidFile)
		return false
	}
	
	// Check if process is actually running
	if err := syscall.Kill(pid, 0); err != nil {
		// Process is not running, clean up PID file
		os.Remove(pidFile)
		return false
	}
	
	return true
}

// StartMonitor starts the messaging monitor if not already running
func (mm *MonitorManager) StartMonitor() error {
	if mm.IsMonitorRunning() {
		if os.Getenv("DMUX_DEBUG") != "" {
			fmt.Printf("[DEBUG] Monitor already running, skipping start\n")
		}
		return nil
	}
	
	// Get current dmux binary path
	dmuxPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get dmux executable path: %v", err)
	}
	
	// Prepare environment for the monitor process
	env := os.Environ()
	// Ensure critical environment variables are passed
	env = append(env, 
		fmt.Sprintf("JMUX_SHARED_DIR=%s", mm.config.SharedDir),
		fmt.Sprintf("DMUX_MESSAGE_DISPLAY=%s", mm.config.MessageDisplayMethod),
		fmt.Sprintf("JMUX_NOTIFICATION_DURATION=%d", mm.config.NotificationDuration),
	)
	if os.Getenv("DMUX_DEBUG") != "" {
		env = append(env, "DMUX_DEBUG=1")
	}
	
	// Start the monitor process
	cmd := exec.Command(dmuxPath, "_internal_messaging_monitor")
	cmd.Env = env
	
	if os.Getenv("DMUX_DEBUG") != "" {
		fmt.Printf("[DEBUG] Starting messaging monitor: %s _internal_messaging_monitor\n", dmuxPath)
	}
	
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start monitor: %v", err)
	}
	
	// Write PID file
	pidFile := mm.config.MonitorPIDFile
	if err := os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", cmd.Process.Pid)), 0644); err != nil {
		// Kill the process if we can't write PID file
		cmd.Process.Kill()
		return fmt.Errorf("failed to write PID file: %v", err)
	}
	
	if os.Getenv("DMUX_DEBUG") != "" {
		fmt.Printf("[DEBUG] Monitor started with PID %d, PID file: %s\n", cmd.Process.Pid, pidFile)
	}
	
	return nil
}

// StopMonitor stops the messaging monitor
func (mm *MonitorManager) StopMonitor() error {
	pidFile := mm.config.MonitorPIDFile
	
	// Check if PID file exists
	if _, err := os.Stat(pidFile); os.IsNotExist(err) {
		return nil // No monitor running
	}
	
	// Read PID from file
	pidBytes, err := os.ReadFile(pidFile)
	if err != nil {
		return err
	}
	
	pid, err := strconv.Atoi(string(pidBytes))
	if err != nil {
		return fmt.Errorf("invalid PID in file: %v", err)
	}
	
	// Send SIGTERM to the process
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		// Process might already be dead
		if err != syscall.ESRCH {
			return fmt.Errorf("failed to stop monitor: %v", err)
		}
	}
	
	// Wait a bit for graceful shutdown
	time.Sleep(100 * time.Millisecond)
	
	// Check if process is still running, force kill if necessary
	if err := syscall.Kill(pid, 0); err == nil {
		if os.Getenv("DMUX_DEBUG") != "" {
			fmt.Printf("[DEBUG] Force killing monitor PID %d\n", pid)
		}
		syscall.Kill(pid, syscall.SIGKILL)
	}
	
	// Clean up PID file
	os.Remove(pidFile)
	
	if os.Getenv("DMUX_DEBUG") != "" {
		fmt.Printf("[DEBUG] Monitor stopped and PID file cleaned up\n")
	}
	
	return nil
}

// RestartMonitor stops and starts the monitor
func (mm *MonitorManager) RestartMonitor() error {
	if err := mm.StopMonitor(); err != nil {
		return fmt.Errorf("failed to stop monitor: %v", err)
	}
	
	// Small delay to ensure cleanup
	time.Sleep(200 * time.Millisecond)
	
	return mm.StartMonitor()
}

// GetMonitorStatus returns the status of the monitor
func (mm *MonitorManager) GetMonitorStatus() string {
	if mm.IsMonitorRunning() {
		pidBytes, _ := os.ReadFile(mm.config.MonitorPIDFile)
		return fmt.Sprintf("Running (PID: %s)", string(pidBytes))
	}
	return "Not running"
}