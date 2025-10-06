package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	cleanupTerminal   bool
	cleanupProcesses  bool
	cleanupSessions   bool
)

// cleanupCmd represents the cleanup command
var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Clean up stale session files and fix terminal issues",
	Long: `Clean up stale session files, orphaned processes, and fix terminal issues.

Examples:
  dmux cleanup                    # Clean up everything (default)
  dmux cleanup --terminal         # Fix terminal settings only
  dmux cleanup --processes        # Kill orphaned processes only  
  dmux cleanup --sessions         # Clean session files only`,
	Run: func(cmd *cobra.Command, args []string) {
		// If no specific flags, do everything
		if !cleanupTerminal && !cleanupProcesses && !cleanupSessions {
			cleanupTerminal = true
			cleanupProcesses = true
			cleanupSessions = true
		}

		totalCleaned := 0
		
		if cleanupTerminal {
			color.Blue("🔧 Fixing terminal settings...")
			performTerminalCleanup()
		}
		
		if cleanupProcesses {
			color.Blue("🧹 Cleaning up orphaned processes...")
			killed := performProcessCleanup()
			if killed > 0 {
				color.Green("✓ Killed %d orphaned process(es)", killed)
				totalCleaned += killed
			} else {
				color.Blue("✓ No orphaned processes found")
			}
		}
		
		if cleanupSessions {
			color.Blue("📁 Cleaning up stale session files...")
			cleaned := performSessionCleanup()
			if cleaned > 0 {
				color.Green("✓ Cleaned up %d stale session(s)", cleaned)
				totalCleaned += cleaned
			} else {
				color.Blue("✓ No stale sessions found")
			}
			
			// Also clean up stale port mappings
			color.Blue("🔌 Cleaning up stale port mappings...")
			portsCleaned := performPortMappingCleanup()
			if portsCleaned > 0 {
				color.Green("✓ Cleaned up %d stale port mapping(s)", portsCleaned)
				totalCleaned += portsCleaned
			} else {
				color.Blue("✓ No stale port mappings found")
			}
		}

		if totalCleaned > 0 {
			color.Green("\n🎉 Cleanup complete: %d items cleaned", totalCleaned)
		} else {
			color.Green("\n✨ System is clean - no cleanup needed")
		}
	},
}

func init() {
	rootCmd.AddCommand(cleanupCmd)
	
	cleanupCmd.Flags().BoolVar(&cleanupTerminal, "terminal", false, "Fix terminal settings only")
	cleanupCmd.Flags().BoolVar(&cleanupProcesses, "processes", false, "Kill orphaned processes only")
	cleanupCmd.Flags().BoolVar(&cleanupSessions, "sessions", false, "Clean session files only")
}

// performTerminalCleanup fixes terminal settings
func performTerminalCleanup() {
	// Save current terminal settings first
	cmd := exec.Command("stty", "-g")
	savedSettings, err := cmd.Output()
	if err == nil && len(savedSettings) > 0 {
		color.Yellow("💾 Saving current terminal settings...")
	}
	
	// Reset terminal settings
	color.Yellow("🔄 Resetting terminal...")
	
	// Run stty sane to fix terminal settings
	cmd = exec.Command("stty", "sane")
	if err := cmd.Run(); err != nil {
		color.Yellow("⚠️  Warning: stty sane failed: %v", err)
	} else {
		color.Green("✓ Applied sane terminal settings")
	}
	
	// Run reset to completely reset terminal
	cmd = exec.Command("reset")
	if err := cmd.Run(); err != nil {
		color.Yellow("⚠️  Warning: reset failed: %v", err)
	} else {
		color.Green("✓ Terminal reset complete")
	}
}

// performProcessCleanup kills orphaned dmux-related processes
func performProcessCleanup() int {
	killed := 0
	
	// Kill orphaned jcat server processes
	cmd := exec.Command("pgrep", "-f", "_internal_jcat_server")
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		pids := strings.Fields(strings.TrimSpace(string(output)))
		for _, pid := range pids {
			if pid != "" {
				color.Yellow("🔫 Killing orphaned jcat server process (PID: %s)", pid)
				killCmd := exec.Command("kill", pid)
				if err := killCmd.Run(); err != nil {
					color.Yellow("⚠️  Warning: failed to kill PID %s: %v", pid, err)
				} else {
					killed++
				}
			}
		}
	}
	
	// Kill orphaned jcat client processes
	cmd = exec.Command("pgrep", "-f", "jcat.*client")
	output, err = cmd.Output()
	if err == nil && len(output) > 0 {
		pids := strings.Fields(strings.TrimSpace(string(output)))
		for _, pid := range pids {
			if pid != "" {
				color.Yellow("🔫 Killing orphaned jcat client process (PID: %s)", pid)
				killCmd := exec.Command("kill", pid)
				if err := killCmd.Run(); err != nil {
					color.Yellow("⚠️  Warning: failed to kill PID %s: %v", pid, err)
				} else {
					killed++
				}
			}
		}
	}
	
	// Kill orphaned socat processes related to dmux
	cmd = exec.Command("pgrep", "-f", "socat.*dmux")
	output, err = cmd.Output()
	if err == nil && len(output) > 0 {
		pids := strings.Fields(strings.TrimSpace(string(output)))
		for _, pid := range pids {
			if pid != "" {
				color.Yellow("🔫 Killing orphaned socat process (PID: %s)", pid)
				killCmd := exec.Command("kill", pid)
				if err := killCmd.Run(); err != nil {
					color.Yellow("⚠️  Warning: failed to kill PID %s: %v", pid, err)
				} else {
					killed++
				}
			}
		}
	}
	
	return killed
}

// performSessionCleanup removes stale session files and returns count of cleaned sessions
func performSessionCleanup() int {
	if cfg == nil {
		return 0
	}

	pattern := "*.session"
	matches, err := filepath.Glob(filepath.Join(cfg.SessionsDir, pattern))
	if err != nil {
		return 0
	}

	cleaned := 0
	for _, sessionFile := range matches {
		if isStaleSession(sessionFile) {
			os.Remove(sessionFile)
			cleaned++
		}
	}

	return cleaned
}

// isStaleSession checks if a session file represents a dead/stale session
func isStaleSession(sessionFile string) bool {
	// Read session data
	session, err := readSessionFromFile(sessionFile)
	if err != nil {
		return true // If we can't read it, consider it stale
	}

	// Check if port is still in use
	if !isPortInUse(session.Port) {
		return true
	}

	// Check if the jcat server process is still running
	cmd := exec.Command("pgrep", "-f", fmt.Sprintf("_internal_jcat_server %d", session.Port))
	err = cmd.Run()
	return err != nil // If pgrep fails, process is not running
}

// isPortInUse checks if a port is currently in use
func isPortInUse(port int) bool {
	cmd := exec.Command("lsof", "-i", fmt.Sprintf(":%d", port))
	err := cmd.Run()
	return err == nil // If lsof succeeds, port is in use
}

// readSessionFromFile reads session data from a file
func readSessionFromFile(filePath string) (*sessionData, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	session := &sessionData{}
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key, value := parts[0], parts[1]
		switch key {
		case "USER":
			session.User = value
		case "SESSION":
			session.Name = value
		case "PORT":
			if port, err := strconv.Atoi(value); err == nil {
				session.Port = port
			}
		case "PID":
			if pid, err := strconv.Atoi(value); err == nil {
				session.PID = pid
			}
		}
	}

	return session, nil
}

// sessionData represents basic session information for cleanup
type sessionData struct {
	User string
	Name string
	Port int
	PID  int
}

// performPortMappingCleanup removes stale port mappings and returns count of cleaned entries
func performPortMappingCleanup() int {
	if cfg == nil {
		return 0
	}

	portMapFile := cfg.PortMapFile
	if portMapFile == "" {
		portMapFile = filepath.Join(cfg.ConfigDir, "port_sessions.db")
	}

	// Read current port mappings
	content, err := os.ReadFile(portMapFile)
	if err != nil {
		return 0 // File doesn't exist or can't be read
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	var validMappings []string
	cleaned := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse port mapping: format is "port:user:session"
		parts := strings.Split(line, ":")
		if len(parts) < 3 {
			continue // Invalid format, skip
		}

		port, err := strconv.Atoi(parts[0])
		if err != nil {
			continue // Invalid port, skip
		}

		// Check if this port mapping is still valid
		if isPortMappingValid(port, parts[1], parts[2]) {
			validMappings = append(validMappings, line)
		} else {
			color.Yellow("🗑️  Removing stale port mapping: %s", line)
			cleaned++
		}
	}

	// Write back only valid mappings
	newContent := strings.Join(validMappings, "\n")
	if len(validMappings) > 0 {
		newContent += "\n"
	}

	if err := os.WriteFile(portMapFile, []byte(newContent), 0644); err != nil {
		color.Yellow("⚠️  Warning: failed to update port mappings file: %v", err)
		return 0
	}

	return cleaned
}

// isPortMappingValid checks if a port mapping is still valid (port in use by jcat server)
func isPortMappingValid(port int, user, session string) bool {
	// Check if port is still in use
	if !isPortInUse(port) {
		return false
	}

	// Check if the specific jcat server process is running on this port
	cmd := exec.Command("pgrep", "-f", fmt.Sprintf("_internal_jcat_server %d", port))
	err := cmd.Run()
	return err == nil // If pgrep succeeds, the specific jcat server is running
}