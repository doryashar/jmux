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

// cleanupCmd represents the cleanup command
var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Clean up stale session files",
	Long:  `Remove session files for sessions that are no longer active (dead processes or closed ports).`,
	Run: func(cmd *cobra.Command, args []string) {
		cleaned := performCleanup()
		if cleaned > 0 {
			color.Green("✓ Cleaned up %d stale session(s)", cleaned)
		} else {
			color.Blue("✓ No stale sessions found")
		}
	},
}

func init() {
	rootCmd.AddCommand(cleanupCmd)
}

// performCleanup removes stale session files and returns count of cleaned sessions
func performCleanup() int {
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