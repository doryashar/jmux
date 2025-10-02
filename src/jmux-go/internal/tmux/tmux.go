package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/fatih/color"
)

// Manager handles tmux operations
type Manager struct{}

// NewManager creates a new tmux manager
func NewManager() *Manager {
	return &Manager{}
}

// StartRegularSession starts a regular tmux session
func (m *Manager) StartRegularSession() error {
	// Check if tmux is available
	if !m.IsTmuxAvailable() {
		return fmt.Errorf("tmux is not available. Please install tmux first")
	}

	// Check if we're already in a tmux session
	if m.IsInTmuxSession() {
		color.Yellow("Already in a tmux session")
		return nil
	}

	color.Blue("🔄 Starting regular tmux session...")
	color.Yellow("💡 Tip: Use 'jmux share' to make it shareable")

	// Start tmux session
	cmd := exec.Command("tmux", "new-session")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Replace current process with tmux
	return syscall.Exec("/usr/bin/tmux", []string{"tmux", "new-session"}, os.Environ())
}

// AttachToSession attaches to an existing tmux session
func (m *Manager) AttachToSession(sessionName string) error {
	if !m.IsTmuxAvailable() {
		return fmt.Errorf("tmux is not available")
	}

	var cmd *exec.Cmd
	if sessionName != "" {
		cmd = exec.Command("tmux", "attach-session", "-t", sessionName)
	} else {
		cmd = exec.Command("tmux", "attach-session")
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// CreateSession creates a new tmux session
func (m *Manager) CreateSession(sessionName string) error {
	if !m.IsTmuxAvailable() {
		return fmt.Errorf("tmux is not available")
	}

	var cmd *exec.Cmd
	if sessionName != "" {
		cmd = exec.Command("tmux", "new-session", "-s", sessionName)
	} else {
		cmd = exec.Command("tmux", "new-session")
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// ListSessions lists tmux sessions
func (m *Manager) ListSessions() error {
	if !m.IsTmuxAvailable() {
		return fmt.Errorf("tmux is not available")
	}

	color.Blue("📋 Tmux sessions (jmux-enhanced):")
	
	cmd := exec.Command("tmux", "list-sessions")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	err := cmd.Run()
	
	fmt.Println()
	color.Blue("💡 Tip: Use 'jmux sessions' to see shared sessions")
	
	return err
}

// KillSession kills a tmux session
func (m *Manager) KillSession(sessionName string) error {
	if !m.IsTmuxAvailable() {
		return fmt.Errorf("tmux is not available")
	}

	cmd := exec.Command("tmux", "kill-session", "-t", sessionName)
	return cmd.Run()
}

// GetCurrentSession gets the current tmux session name
func (m *Manager) GetCurrentSession() (string, error) {
	if !m.IsInTmuxSession() {
		return "", fmt.Errorf("not in a tmux session")
	}

	cmd := exec.Command("tmux", "display-message", "-p", "#S")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// IsTmuxAvailable checks if tmux is available
func (m *Manager) IsTmuxAvailable() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

// IsInTmuxSession checks if we're currently in a tmux session
func (m *Manager) IsInTmuxSession() bool {
	return os.Getenv("TMUX") != ""
}

// HasSession checks if a session exists
func (m *Manager) HasSession(sessionName string) bool {
	if !m.IsTmuxAvailable() {
		return false
	}

	cmd := exec.Command("tmux", "has-session", "-t", sessionName)
	return cmd.Run() == nil
}

// RunTmuxCommand runs a tmux command and replaces the current process
func (m *Manager) RunTmuxCommand(args []string) error {
	if !m.IsTmuxAvailable() {
		return fmt.Errorf("tmux is not available")
	}

	// Prepend "tmux" to the arguments
	tmuxArgs := append([]string{"tmux"}, args...)
	
	// Get the tmux path
	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		return err
	}

	// Replace current process with tmux
	return syscall.Exec(tmuxPath, tmuxArgs, os.Environ())
}

// SetupTmuxSession prepares a tmux session for sharing
func (m *Manager) SetupTmuxSession(sessionName string) error {
	if !m.IsTmuxAvailable() {
		return fmt.Errorf("tmux is not available")
	}

	// Set tmux status to show sharing info
	statusMsg := fmt.Sprintf("[SHARED] Session: %s | Use 'jmux stop' to stop sharing", sessionName)
	
	cmd := exec.Command("tmux", "set-option", "-g", "status-right", statusMsg)
	if err := cmd.Run(); err != nil {
		// Non-critical error
		color.Yellow("Warning: Could not update tmux status")
	}

	cmd = exec.Command("tmux", "set-option", "-g", "status-right-length", "100")
	cmd.Run() // Ignore errors

	return nil
}

// ClearTmuxStatus clears the tmux status when stopping sharing
func (m *Manager) ClearTmuxStatus() error {
	if !m.IsTmuxAvailable() {
		return nil
	}

	cmd := exec.Command("tmux", "set-option", "-g", "status-right", "")
	return cmd.Run()
}