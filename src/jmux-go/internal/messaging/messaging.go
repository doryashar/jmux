package messaging

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/fsnotify/fsnotify"
	"jmux/internal/config"
)

// MessageType represents different types of messages
type MessageType string

const (
	MessageTypeInvite  MessageType = "INVITE"
	MessageTypeUrgent  MessageType = "URGENT"
	MessageTypeMessage MessageType = "MESSAGE"
)

// Message represents a jmux message
type Message struct {
	From      string
	Type      MessageType
	Timestamp int64
	Data      string
	Priority  string
}

// Messaging handles the messaging system
type Messaging struct {
	config  *config.Config
	watcher *fsnotify.Watcher
	done    chan bool
	logger  *Logger
}

// NewMessaging creates a new messaging instance
func NewMessaging(cfg *config.Config) *Messaging {
	// Try to create logger, but don't fail if we can't
	logger, err := NewLogger(cfg.MonitorLogFile)
	if err != nil {
		fmt.Printf("Warning: Could not create monitor log file: %v\n", err)
		logger = nil
	}

	return &Messaging{
		config: cfg,
		done:   make(chan bool),
		logger: logger,
	}
}

// StartLiveMonitoring starts the live message monitoring using tail-based approach
func (m *Messaging) StartLiveMonitoring() error {
	if !m.config.RealtimeEnabled {
		return nil
	}

	currentUser := os.Getenv("USER")
	if currentUser == "" {
		if m.logger != nil {
			m.logger.Error("No USER environment variable")
		}
		return fmt.Errorf("unable to determine current user")
	}

	// Create user-specific message file path
	userMessageFile := filepath.Join(m.config.MessagesDir, currentUser+".messages")
	
	// Ensure the message file exists with proper permissions
	if _, err := os.Stat(userMessageFile); os.IsNotExist(err) {
		// Create file with 666 permissions to allow shared access
		file, err := os.OpenFile(userMessageFile, os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			if m.logger != nil {
				m.logger.Error("Failed to create user message file: %v", err)
			}
			return fmt.Errorf("failed to create user message file: %v", err)
		}
		file.Close()
		
		// Explicitly set permissions for shared directory scenarios
		if err := os.Chmod(userMessageFile, 0666); err != nil {
			if m.logger != nil {
				m.logger.Debug("Could not set file permissions for %s: %v", userMessageFile, err)
			}
		}
		
		if m.logger != nil {
			m.logger.Info("Created user message file with shared permissions: %s", userMessageFile)
		}
	}

	// Log that monitoring started
	if m.logger != nil {
		m.logger.Info("Live monitoring started for user %s, file: %s", currentUser, userMessageFile)
	}

	// Start tail-based monitoring for main messages file
	go m.tailUserMessages(userMessageFile)

	// Also start monitoring for user invitations file
	userInvitesFile := filepath.Join(m.config.MessagesDir, currentUser+".invites")
	go m.tailUserInvites(userInvitesFile)

	return nil
}

// tailUserMessages monitors a user's message file using tail-like approach
func (m *Messaging) tailUserMessages(userMessageFile string) {
	if m.logger != nil {
		m.logger.Debug("Starting tail monitoring for: %s", userMessageFile)
	}

	// Get initial file position
	lastSize := int64(0)
	if stat, err := os.Stat(userMessageFile); err == nil {
		lastSize = stat.Size()
	}

	ticker := time.NewTicker(500 * time.Millisecond) // Poll every 500ms
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := m.checkForNewMessages(userMessageFile, &lastSize); err != nil {
				if m.logger != nil {
					m.logger.Debug("Error checking for new messages: %v", err)
				}
			}
		case <-m.done:
			if m.logger != nil {
				m.logger.Debug("Tail monitoring stopped")
			}
			return
		}
	}
}

// tailUserInvites monitors a user's invites file using tail-like approach  
func (m *Messaging) tailUserInvites(userInvitesFile string) {
	if m.logger != nil {
		m.logger.Info("Starting tail monitoring for invites: %s", userInvitesFile)
	}

	// Ensure the invites file exists with proper permissions
	if _, err := os.Stat(userInvitesFile); os.IsNotExist(err) {
		// Create file with 666 permissions to allow shared access
		file, err := os.OpenFile(userInvitesFile, os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			if m.logger != nil {
				m.logger.Error("Failed to create user invites file: %v", err)
			}
			return
		}
		file.Close()
		
		// Explicitly set permissions for shared directory scenarios
		if err := os.Chmod(userInvitesFile, 0666); err != nil {
			if m.logger != nil {
				m.logger.Debug("Could not set file permissions for %s: %v", userInvitesFile, err)
			}
		}
		
		if m.logger != nil {
			m.logger.Info("Created user invites file with shared permissions: %s", userInvitesFile)
		}
	}

	// Get initial file position
	lastSize := int64(0)
	if stat, err := os.Stat(userInvitesFile); err == nil {
		lastSize = stat.Size()
	}

	ticker := time.NewTicker(500 * time.Millisecond) // Poll every 500ms like messages
	defer ticker.Stop()
	
	// Cleanup timer - run every 10 minutes to remove expired invitations
	cleanupTicker := time.NewTicker(10 * time.Minute)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := m.checkForNewInvites(userInvitesFile, &lastSize); err != nil {
				if m.logger != nil {
					m.logger.Debug("Error checking for new invites: %v", err)
				}
			}
		case <-cleanupTicker.C:
			if err := m.cleanupExpiredInvitations(userInvitesFile); err != nil {
				if m.logger != nil {
					m.logger.Debug("Error cleaning up expired invitations: %v", err)
				}
			} else if m.logger != nil {
				m.logger.Debug("Performed invitation cleanup check")
			}
		case <-m.done:
			if m.logger != nil {
				m.logger.Debug("Invites monitoring stopped")
			}
			return
		}
	}
}

// checkForNewInvites checks if the invites file has new invitations and processes them
func (m *Messaging) checkForNewInvites(userInvitesFile string, lastSize *int64) error {
	stat, err := os.Stat(userInvitesFile)
	if err != nil {
		return err
	}

	currentSize := stat.Size()
	if currentSize <= *lastSize {
		return nil // No new content
	}

	// Read all invitations from the file
	file, err := os.Open(userInvitesFile)
	if err != nil {
		return err
	}
	defer file.Close()

	var invitations []Message
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			var invite Message
			if err := json.Unmarshal([]byte(line), &invite); err != nil {
				if m.logger != nil {
					m.logger.Debug("Error parsing invitation JSON: %v", err)
				}
				continue // Skip malformed lines
			}
			invitations = append(invitations, invite)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// Display all invitations found
	for _, invite := range invitations {
		if err := m.handleNewMessageLine(invite); err != nil {
			if m.logger != nil {
				m.logger.Debug("Error processing invitation: %v", err)
			}
		}
	}

	// For monitor mode, don't clear the file - invitations should persist until accepted/expired
	// The monitor just displays them, but they remain available for manual processing
	if len(invitations) > 0 {
		if m.logger != nil {
			m.logger.Info("Processed %d invitations (kept in file for manual handling)", len(invitations))
		}
	}

	return nil
}

// cleanupExpiredInvitations removes invitations older than 24 hours
func (m *Messaging) cleanupExpiredInvitations(userInvitesFile string) error {
	if _, err := os.Stat(userInvitesFile); os.IsNotExist(err) {
		return nil // File doesn't exist, nothing to clean
	}

	file, err := os.Open(userInvitesFile)
	if err != nil {
		return err
	}
	defer file.Close()

	var validInvitations []Message
	currentTime := time.Now().Unix()
	expirationTime := int64(24 * 60 * 60) // 24 hours in seconds

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			var invite Message
			if err := json.Unmarshal([]byte(line), &invite); err != nil {
				// Skip malformed invitations
				continue
			}
			
			// Keep invitations that are less than 24 hours old
			if currentTime-invite.Timestamp < expirationTime {
				validInvitations = append(validInvitations, invite)
			} else if m.logger != nil {
				m.logger.Info("Expired invitation removed: from=%s timestamp=%d", invite.From, invite.Timestamp)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// Rewrite the file with only valid invitations
	return m.rewriteInvitationsFile(userInvitesFile, validInvitations)
}

// rewriteInvitationsFile rewrites the invitations file with the given invitations
func (m *Messaging) rewriteInvitationsFile(userInvitesFile string, invitations []Message) error {
	// Create temp file for atomic write
	tempFile := userInvitesFile + ".tmp"
	
	file, err := os.OpenFile(tempFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write all valid invitations
	for _, invite := range invitations {
		inviteJSON, err := json.Marshal(invite)
		if err != nil {
			continue // Skip malformed invitations
		}
		file.WriteString(string(inviteJSON) + "\n")
	}

	// Atomic move
	if err := os.Rename(tempFile, userInvitesFile); err != nil {
		os.Remove(tempFile) // Clean up temp file on error
		return err
	}

	return nil
}

// RemoveInvitationFromUser removes invitations from a specific user (public method)
func (m *Messaging) RemoveInvitationFromUser(fromUser string) error {
	currentUser := os.Getenv("USER")
	if currentUser == "" {
		return fmt.Errorf("unable to determine current user")
	}

	userInvitesFile := filepath.Join(m.config.MessagesDir, currentUser+".invites")
	
	if _, err := os.Stat(userInvitesFile); os.IsNotExist(err) {
		return nil // File doesn't exist, nothing to remove
	}

	file, err := os.Open(userInvitesFile)
	if err != nil {
		return err
	}
	defer file.Close()

	var remainingInvitations []Message
	removedCount := 0

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			var invite Message
			if err := json.Unmarshal([]byte(line), &invite); err != nil {
				// Keep malformed invitations to avoid data loss
				continue
			}
			
			// Keep invitations that are NOT from the specified user
			if invite.From != fromUser {
				remainingInvitations = append(remainingInvitations, invite)
			} else {
				removedCount++
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	if removedCount > 0 {
		if m.logger != nil {
			m.logger.Info("Removed %d invitations from %s", removedCount, fromUser)
		}
		return m.rewriteInvitationsFile(userInvitesFile, remainingInvitations)
	}

	return nil
}

// findDmuxExecutable finds the dmux executable with full path
func findDmuxExecutable() string {
	// First try to find dmux in PATH
	if dmuxPath, err := exec.LookPath("dmux"); err == nil {
		return dmuxPath
	}
	
	// Try current executable's directory (if we're running from dmux)
	if currentExe, err := os.Executable(); err == nil {
		if strings.Contains(currentExe, "dmux") {
			return currentExe
		}
		// Try dmux in the same directory as current executable
		dmuxPath := filepath.Join(filepath.Dir(currentExe), "dmux")
		if _, err := os.Stat(dmuxPath); err == nil {
			return dmuxPath
		}
	}
	
	// Try common installation locations
	commonPaths := []string{
		"/usr/local/bin/dmux",
		"/usr/bin/dmux", 
		filepath.Join(os.Getenv("HOME"), ".local/bin/dmux"),
		filepath.Join(os.Getenv("HOME"), "bin/dmux"),
	}
	
	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	
	// Fallback to just "dmux" and hope it's in PATH when terminal starts
	return "dmux"
}

// findTerminalEmulator finds an available terminal emulator
func findTerminalEmulator() (string, []string) {
	// Define terminal emulators with their execute argument patterns
	terminals := []struct {
		cmd  string
		args []string
	}{
		{"konsole", []string{"-e"}},
		{"gnome-terminal", []string{"--"}},
		{"xfce4-terminal", []string{"-e"}},
		{"mate-terminal", []string{"-e"}},
		{"xterm", []string{"-e"}},
		{"urxvt", []string{"-e"}},
		{"terminator", []string{"-e"}},
		{"tilix", []string{"-e"}},
		{"alacritty", []string{"-e"}},
		{"kitty", []string{"-e"}},
		{"x-terminal-emulator", []string{"-e"}},
	}
	
	// Check which terminals are available
	for _, term := range terminals {
		if _, err := exec.LookPath(term.cmd); err == nil {
			return term.cmd, term.args
		}
	}
	
	// Fallback to x-terminal-emulator with -e
	return "x-terminal-emulator", []string{"-e"}
}

// launchTerminalForJoin launches a terminal to join a session
func (m *Messaging) launchTerminalForJoin(fromUser string) error {
	dmuxPath := findDmuxExecutable()
	termCmd, termArgs := findTerminalEmulator()
	
	if os.Getenv("DMUX_DEBUG") != "" {
		fmt.Printf("[DEBUG] Using dmux path: %s\n", dmuxPath)
		fmt.Printf("[DEBUG] Using terminal: %s %v\n", termCmd, termArgs)
	}
	
	// Build the command to run in the terminal
	// Use absolute path to dmux and add common paths to environment
	joinCommand := fmt.Sprintf("export PATH=\"$PATH:/usr/local/bin:/usr/bin:$HOME/.local/bin:$HOME/bin\"; %s join %s; exec bash", dmuxPath, fromUser)
	
	// Build the full terminal command
	var args []string
	args = append(args, termArgs...)
	args = append(args, "bash", "-c", joinCommand)
	
	if os.Getenv("DMUX_DEBUG") != "" {
		fmt.Printf("[DEBUG] Terminal command: %s %v\n", termCmd, args)
	}
	
	// Launch the terminal
	cmd := exec.Command(termCmd, args...)
	
	// Set environment to include common paths
	env := os.Environ()
	currentPath := os.Getenv("PATH")
	enhancedPath := currentPath + ":/usr/local/bin:/usr/bin:" + 
		filepath.Join(os.Getenv("HOME"), ".local/bin") + ":" + 
		filepath.Join(os.Getenv("HOME"), "bin")
	
	// Update PATH in environment
	var newEnv []string
	pathSet := false
	for _, envVar := range env {
		if strings.HasPrefix(envVar, "PATH=") {
			newEnv = append(newEnv, "PATH="+enhancedPath)
			pathSet = true
		} else {
			newEnv = append(newEnv, envVar)
		}
	}
	if !pathSet {
		newEnv = append(newEnv, "PATH="+enhancedPath)
	}
	
	cmd.Env = newEnv
	
	if err := cmd.Start(); err != nil {
		if m.logger != nil {
			m.logger.Debug("Failed to launch terminal for joining %s: %v", fromUser, err)
		}
		return err
	}
	
	if m.logger != nil {
		m.logger.Info("Launched terminal for joining %s session", fromUser)
	}
	
	return nil
}

// checkForNewMessages checks if the file has new messages and processes them
func (m *Messaging) checkForNewMessages(userMessageFile string, lastSize *int64) error {
	stat, err := os.Stat(userMessageFile)
	if err != nil {
		return err
	}

	currentSize := stat.Size()
	if currentSize <= *lastSize {
		return nil // No new content
	}

	// Read all messages from the file
	file, err := os.Open(userMessageFile)
	if err != nil {
		return err
	}
	defer file.Close()

	var messages []Message
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			var msg Message
			if err := json.Unmarshal([]byte(line), &msg); err != nil {
				if m.logger != nil {
					m.logger.Debug("Error parsing message JSON: %v", err)
				}
				continue // Skip malformed lines
			}
			messages = append(messages, msg)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// Display all messages found
	for _, msg := range messages {
		if err := m.handleNewMessageLine(msg); err != nil {
			if m.logger != nil {
				m.logger.Debug("Error processing message: %v", err)
			}
		}
	}

	// Clear the file after processing all messages (monitor consumes them)
	if len(messages) > 0 {
		if err := os.Truncate(userMessageFile, 0); err != nil {
			if m.logger != nil {
				m.logger.Debug("Could not clear message file after processing: %v", err)
			}
		} else {
			if m.logger != nil {
				m.logger.Info("Cleared %d processed messages from file", len(messages))
			}
			*lastSize = 0 // Reset size tracking after clearing
		}
	}

	return nil
}

// StopLiveMonitoring stops the live message monitoring
func (m *Messaging) StopLiveMonitoring() {
	if m.logger != nil {
		m.logger.LogMonitorStop()
	}
	
	if m.done != nil {
		select {
		case m.done <- true:
		default:
		}
	}
	
	// Close logger
	if m.logger != nil {
		m.logger.Close()
	}
}

// handleNewMessageLine processes a message struct
func (m *Messaging) handleNewMessageLine(msg Message) error {
	if m.logger != nil {
		m.logger.LogMessageProcessed(msg.From, string(msg.Type), msg.Data)
	}

	// Display message using configured method
	switch m.config.MessageDisplayMethod {
	case "kdialog":
		if m.logger != nil {
			m.logger.LogDisplayMethod("kdialog")
		}
		m.displayKDialogMessage(&msg)
	case "notify":
		if m.logger != nil {
			m.logger.LogDisplayMethod("notify-send")
		}
		m.displayNotifyMessage(&msg)
	case "tmux":
		if os.Getenv("TMUX") != "" || m.hasTmuxSessions() {
			if m.logger != nil {
				m.logger.LogDisplayMethod("tmux")
			}
			m.displayTmuxMessage(&msg)
		} else {
			// Fallback to auto-detect
			m.displayAutoMessage(&msg)
		}
	case "terminal":
		if m.logger != nil {
			m.logger.LogDisplayMethod("terminal")
		}
		m.displayRealtimeMessage(&msg)
	case "auto":
		m.displayAutoMessage(&msg)
	default:
		// Default to auto-detect
		m.displayAutoMessage(&msg)
	}

	return nil
}

// handleNewMessage processes a new message file (legacy method for compatibility)
func (m *Messaging) handleNewMessage(msgFile string) {
	// Small delay to ensure file is fully written
	time.Sleep(100 * time.Millisecond)

	msg, err := m.readMessageFile(msgFile)
	if err != nil {
		if m.logger != nil {
			m.logger.Debug("Failed to read message file %s: %v", msgFile, err)
		}
		return
	}

	// Check if message is for current user
	currentUser := os.Getenv("USER")
	if currentUser == "" {
		if m.logger != nil {
			m.logger.Debug("No USER environment variable")
		}
		return
	}

	expectedPrefix := currentUser + "_"
	fileName := filepath.Base(msgFile)
	if !strings.HasPrefix(fileName, expectedPrefix) {
		if m.logger != nil {
			m.logger.Debug("Message file %s not for user %s", fileName, currentUser)
		}
		return
	}

	if m.logger != nil {
		m.logger.LogMessageProcessed(msg.From, string(msg.Type), msg.Data)
	}

	// Display message using configured method
	switch m.config.MessageDisplayMethod {
	case "zenity":
		if m.logger != nil {
			m.logger.LogDisplayMethod("zenity")
		}
		m.displayZenityMessage(msg)
	case "kdialog":
		if m.logger != nil {
			m.logger.LogDisplayMethod("kdialog")
		}
		m.displayKDialogMessage(msg)
	case "notify":
		if m.logger != nil {
			m.logger.LogDisplayMethod("notify-send")
		}
		m.displayNotifyMessage(msg)
	case "tmux":
		if os.Getenv("TMUX") != "" || m.hasTmuxSessions() {
			if m.logger != nil {
				m.logger.LogDisplayMethod("tmux")
			}
			m.displayTmuxMessage(msg)
		} else {
			// Fallback to auto-detect
			m.displayAutoMessage(msg)
		}
	case "terminal":
		if m.logger != nil {
			m.logger.LogDisplayMethod("terminal")
		}
		m.displayRealtimeMessage(msg)
	case "auto":
		m.displayAutoMessage(msg)
	default:
		// Default to auto-detect
		m.displayAutoMessage(msg)
	}

	// Auto-remove old message after some time
	go func() {
		time.Sleep(time.Duration(m.config.NotificationDuration) * time.Second)
		os.Remove(msgFile)
	}()
}

// displayRealtimeMessage shows the message as a terminal overlay
func (m *Messaging) displayRealtimeMessage(msg *Message) {
	// Check if we're in an interactive terminal or tmux
	if !isInteractiveTerminal() && os.Getenv("TMUX") == "" {
		if os.Getenv("DMUX_DEBUG") != "" {
			fmt.Printf("[DEBUG] Not displaying message - not interactive terminal and not in TMUX\n")
		}
		return
	}

	// Save cursor position and move to bottom
	fmt.Print("\033[s") // Save cursor
	fmt.Print("\033[999;1H") // Move to bottom
	fmt.Print("\033[1A") // Move up one line

	// Clear line and display message with background
	fmt.Print("\033[K\033[7m") // Clear line and reverse video

	switch msg.Type {
	case MessageTypeInvite:
		fmt.Printf(" 📨 INVITE from %s: Join session '%s' | dmux join %s ", msg.From, msg.Data, msg.From)
	case MessageTypeUrgent:
		fmt.Printf(" 🚨 URGENT from %s: %s ", msg.From, msg.Data)
	default:
		fmt.Printf(" 💬 %s: %s ", msg.From, msg.Data)
	}

	fmt.Print("\033[0m") // Reset formatting

	// Auto-hide after duration
	go func() {
		time.Sleep(time.Duration(m.config.NotificationDuration) * time.Second)
		fmt.Print("\033[u") // Restore cursor
		fmt.Print("\033[K") // Clear line
	}()
}

// SendMessage sends a message to a user by appending to their message file
func (m *Messaging) SendMessage(toUser string, msgType MessageType, data string) error {
	timestamp := time.Now().Unix()
	userMessageFile := filepath.Join(m.config.MessagesDir, toUser+".messages")

	currentUser := os.Getenv("USER")
	if currentUser == "" {
		currentUser = "unknown"
	}

	// Create JSON-like message format for easier parsing
	messageLine := fmt.Sprintf("{\"from\":\"%s\",\"type\":\"%s\",\"timestamp\":%d,\"data\":\"%s\",\"priority\":\"normal\"}\n", 
		currentUser, msgType, timestamp, strings.ReplaceAll(data, "\"", "\\\""))

	// Ensure message file exists with proper permissions before writing
	if _, err := os.Stat(userMessageFile); os.IsNotExist(err) {
		// Create file with 666 permissions for shared access
		if file, err := os.OpenFile(userMessageFile, os.O_CREATE|os.O_WRONLY, 0666); err != nil {
			return fmt.Errorf("failed to create user message file: %v", err)
		} else {
			file.Close()
			// Explicitly set permissions
			os.Chmod(userMessageFile, 0666)
			if m.logger != nil {
				m.logger.Info("Created user message file for %s with shared permissions", toUser)
			}
		}
	}

	// Append message to user's message file
	file, err := os.OpenFile(userMessageFile, os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return fmt.Errorf("failed to open user message file: %v", err)
	}
	defer file.Close()

	// Ensure permissions are correct (in case file already existed)
	if err := os.Chmod(userMessageFile, 0666); err != nil {
		if m.logger != nil {
			m.logger.Debug("Could not set file permissions for %s: %v", userMessageFile, err)
		}
	}

	// Write the message
	if _, err := file.WriteString(messageLine); err != nil {
		return fmt.Errorf("failed to write message: %v", err)
	}

	if m.logger != nil {
		m.logger.Info("Message sent to %s: %s", toUser, data)
	}

	return nil
}

// SendInvitation sends an invitation to a user by appending to their invites file
func (m *Messaging) SendInvitation(toUser string, inviteData string) error {
	timestamp := time.Now().Unix()
	userInvitesFile := filepath.Join(m.config.MessagesDir, toUser+".invites")

	currentUser := os.Getenv("USER")
	if currentUser == "" {
		currentUser = "unknown"
	}

	// Create JSON invitation format for easier parsing
	invitationLine := fmt.Sprintf("{\"from\":\"%s\",\"type\":\"%s\",\"timestamp\":%d,\"data\":\"%s\",\"priority\":\"high\"}\n", 
		currentUser, MessageTypeInvite, timestamp, strings.ReplaceAll(inviteData, "\"", "\\\""))

	// Ensure invites file exists with proper permissions before writing
	if _, err := os.Stat(userInvitesFile); os.IsNotExist(err) {
		// Create file with 666 permissions for shared access
		if file, err := os.OpenFile(userInvitesFile, os.O_CREATE|os.O_WRONLY, 0666); err != nil {
			return fmt.Errorf("failed to create user invites file: %v", err)
		} else {
			file.Close()
			// Explicitly set permissions
			os.Chmod(userInvitesFile, 0666)
			if m.logger != nil {
				m.logger.Info("Created user invites file for %s with shared permissions", toUser)
			}
		}
	}

	// Append invitation to user's invites file
	file, err := os.OpenFile(userInvitesFile, os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return fmt.Errorf("failed to open user invites file: %v", err)
	}
	defer file.Close()

	// Ensure permissions are correct (in case file already existed)
	if err := os.Chmod(userInvitesFile, 0666); err != nil {
		if m.logger != nil {
			m.logger.Debug("Could not set file permissions for %s: %v", userInvitesFile, err)
		}
	}

	// Write the invitation
	if _, err := file.WriteString(invitationLine); err != nil {
		return fmt.Errorf("failed to write invitation: %v", err)
	}

	if m.logger != nil {
		m.logger.Info("Invitation sent to %s: %s", toUser, inviteData)
	}

	return nil
}

// ReadMessages reads and displays messages for current user from their message and invites files
func (m *Messaging) ReadMessages() error {
	currentUser := os.Getenv("USER")
	if currentUser == "" {
		return fmt.Errorf("unable to determine current user")
	}

	userMessageFile := filepath.Join(m.config.MessagesDir, currentUser+".messages")
	userInvitesFile := filepath.Join(m.config.MessagesDir, currentUser+".invites")
	
	var allMessages []Message
	
	// Read from main messages file
	if _, err := os.Stat(userMessageFile); err == nil {
		messages, err := m.readMessagesFromFile(userMessageFile)
		if err == nil {
			allMessages = append(allMessages, messages...)
		}
	}
	
	// Read from invites file  
	if _, err := os.Stat(userInvitesFile); err == nil {
		invites, err := m.readMessagesFromFile(userInvitesFile)
		if err == nil {
			allMessages = append(allMessages, invites...)
		}
	}

	if len(allMessages) == 0 {
		color.Yellow("No new messages")
		return nil
	}

	color.Blue("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	color.Green("New Messages")
	color.Blue("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for _, msg := range allMessages {
		fmt.Printf("\n")
		switch msg.Type {
		case MessageTypeInvite:
			color.Cyan("From: %s", msg.From)
			color.Yellow("  Invitation to join session")
			fmt.Printf("  Session: %s\n", msg.Data)
			color.Green("  To join: dmux join %s", msg.From)
		case MessageTypeUrgent:
			color.Red("From: %s (URGENT)", msg.From)
			fmt.Printf("  %s\n", msg.Data)
		default:
			color.Cyan("From: %s", msg.From)
			fmt.Printf("  %s\n", msg.Data)
		}
	}

	fmt.Println()

	// Clear messages file but keep invitations (they should persist until accepted)
	if _, err := os.Stat(userMessageFile); err == nil {
		if err := os.Truncate(userMessageFile, 0); err != nil {
			return fmt.Errorf("failed to clear messages: %v", err)
		}
	}
	
	// Don't clear invitations - they should persist until accepted or expired
	// Users can manually clear them by joining sessions or they'll auto-expire after 24h

	return nil
}

// readMessagesFromFile reads messages from a specific file
func (m *Messaging) readMessagesFromFile(filePath string) ([]Message, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var messages []Message
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			var msg Message
			if err := json.Unmarshal([]byte(line), &msg); err != nil {
				continue // Skip malformed lines
			}
			messages = append(messages, msg)
		}
	}

	return messages, scanner.Err()
}

// readMessageFile reads a message file
func (m *Messaging) readMessageFile(filePath string) (*Message, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	msg := &Message{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key, value := parts[0], parts[1]
		switch key {
		case "FROM":
			msg.From = value
		case "TYPE":
			msg.Type = MessageType(value)
		case "TIMESTAMP":
			if ts, err := strconv.ParseInt(value, 10, 64); err == nil {
				msg.Timestamp = ts
			}
		case "DATA":
			msg.Data = value
		case "PRIORITY":
			msg.Priority = value
		}
	}

	return msg, scanner.Err()
}

// displayTmuxMessage shows the message using tmux display-message
func (m *Messaging) displayTmuxMessage(msg *Message) {
	var tmuxMsg string
	switch msg.Type {
	case MessageTypeInvite:
		tmuxMsg = fmt.Sprintf("📨 INVITE from %s: Join session '%s' | dmux join %s", msg.From, msg.Data, msg.From)
	case MessageTypeUrgent:
		tmuxMsg = fmt.Sprintf("🚨 URGENT from %s: %s", msg.From, msg.Data)
	default:
		tmuxMsg = fmt.Sprintf("💬 Message from %s: %s", msg.From, msg.Data)
	}

	if os.Getenv("DMUX_DEBUG") != "" {
		fmt.Printf("[DEBUG] Displaying tmux message: %s\n", tmuxMsg)
	}

	// Try multiple tmux session targets to ensure message displays
	sessionTargets := []string{"dmux-main", ""} // Try specific session first, then current
	
	for _, target := range sessionTargets {
		var cmd *exec.Cmd
		if target != "" {
			cmd = exec.Command("tmux", "display-message", "-t", target, "-d", "5000", tmuxMsg)
		} else {
			cmd = exec.Command("tmux", "display-message", "-d", "5000", tmuxMsg)
		}
		
		if err := cmd.Run(); err == nil {
			if os.Getenv("DMUX_DEBUG") != "" {
				fmt.Printf("[DEBUG] Successfully displayed message to target: %s\n", target)
			}
			return // Success
		} else if os.Getenv("DMUX_DEBUG") != "" {
			fmt.Printf("[DEBUG] Failed to display tmux message to target '%s': %v\n", target, err)
		}
	}
}

// isInteractiveTerminal checks if we're in an interactive terminal
func isInteractiveTerminal() bool {
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & fs.ModeCharDevice) != 0
}

// hasTmuxSessions checks if there are any active tmux sessions
func (m *Messaging) hasTmuxSessions() bool {
	cmd := exec.Command("tmux", "list-sessions")
	err := cmd.Run()
	return err == nil
}

// displayKDialogMessage shows the message using kdialog
func (m *Messaging) displayKDialogMessage(msg *Message) {
	// Check if kdialog is available
	if _, err := exec.LookPath("kdialog"); err != nil {
		if os.Getenv("DMUX_DEBUG") != "" {
			fmt.Printf("[DEBUG] kdialog not found, falling back to terminal display\n")
		}
		m.displayRealtimeMessage(msg)
		return
	}

	var title, text string
	var dialogType string = "--msgbox"
	
	switch msg.Type {
	case MessageTypeInvite:
		title = "dmux - Session Invitation"
		text = fmt.Sprintf("📨 Invitation from %s\n\nJoin session: %s\n\nTo join, run:\ndmux join %s", msg.From, msg.Data, msg.From)
		dialogType = "--msgbox"
	case MessageTypeUrgent:
		title = "dmux - Urgent Message"
		text = fmt.Sprintf("🚨 URGENT from %s\n\n%s", msg.From, msg.Data)
		dialogType = "--msgbox"
	default:
		title = "dmux - Message"
		text = fmt.Sprintf("💬 Message from %s\n\n%s", msg.From, msg.Data)
		dialogType = "--msgbox"
	}

	if os.Getenv("DMUX_DEBUG") != "" {
		fmt.Printf("[DEBUG] Displaying kdialog message: %s\n", text)
	}

	// Build kdialog command with focus and attention options
	args := []string{
		dialogType, text,
		"--title", title,
		"--icon", "mail-message",
	}
	
	// Add window attachment for focus - use WINDOWID if available, otherwise attach to root
	if windowID := os.Getenv("WINDOWID"); windowID != "" {
		args = append(args, "--attach", windowID)
	} else {
		args = append(args, "--attach", "0") // Attach to root window
	}
	
	// Add urgency-specific options
	switch msg.Type {
	case MessageTypeUrgent:
		// For urgent messages, use error dialog type which is more attention-grabbing
		args[0] = "--error"
		args = append(args, 
			"--geometry", "400x200+100+100", // Position prominently
			"--dontagain", "dmux-urgent-msg") // Prevent spam for urgent messages
	case MessageTypeInvite:
		// For invites, use question dialog with buttons for better interaction
		args[0] = "--yesno"
		args[1] = text + "\n\nOpen terminal to join?"
		args = append(args, "--geometry", "450x250+100+100")
	default:
		// Regular messages get standard positioning
		args = append(args, "--geometry", "400x150+100+100")
	}
	
	if os.Getenv("DMUX_DEBUG") != "" {
		fmt.Printf("[DEBUG] Displaying kdialog with args: %v\n", args)
	}

	// Display the dialog
	cmd := exec.Command("kdialog", args...)
	
	// Run kdialog in background so it doesn't block
	go func() {
		if err := cmd.Run(); err == nil {
			// If it was an invite and user clicked yes, open terminal
			if msg.Type == MessageTypeInvite {
				if os.Getenv("DMUX_DEBUG") != "" {
					fmt.Printf("[DEBUG] User clicked yes\n")
				}
				if err := m.launchTerminalForJoin(msg.From); err != nil && os.Getenv("DMUX_DEBUG") != "" {
					fmt.Printf("[DEBUG] Failed to open terminal for joining: %v\n", err)
				}
			}
		} else if msg.Type != MessageTypeInvite && os.Getenv("DMUX_DEBUG") != "" {
			fmt.Printf("[DEBUG] Failed to display kdialog message: %v\n", err)
		}
	}()
}

// displayZenityMessage shows the message using zenity
func (m *Messaging) displayZenityMessage(msg *Message) {
	// Check if zenity is available
	if _, err := exec.LookPath("zenity"); err != nil {
		if os.Getenv("DMUX_DEBUG") != "" {
			fmt.Printf("[DEBUG] zenity not found, falling back to terminal display\n")
		}
		m.displayRealtimeMessage(msg)
		return
	}

	var title, text string
	var dialogType string = "--info"
	
	switch msg.Type {
	case MessageTypeInvite:
		title = "dmux - Session Invitation"
		text = fmt.Sprintf("📨 Invitation from %s\n\nJoin session: %s\n\nTo join, run: dmux join %s", msg.From, msg.Data, msg.From)
		dialogType = "--question"
	case MessageTypeUrgent:
		title = "dmux - Urgent Message"
		text = fmt.Sprintf("🚨 URGENT from %s\n\n%s", msg.From, msg.Data)
		dialogType = "--warning"
	default:
		title = "dmux - Message"
		text = fmt.Sprintf("💬 Message from %s\n\n%s", msg.From, msg.Data)
		dialogType = "--info"
	}

	if os.Getenv("DMUX_DEBUG") != "" {
		fmt.Printf("[DEBUG] Displaying zenity message: %s\n", text)
	}

	// Build zenity command
	args := []string{
		dialogType,
		"--title", title,
		"--text", text,
		"--width", "400",
		"--height", "200",
	}
	
	// Add message type specific options
	switch msg.Type {
	case MessageTypeUrgent:
		args = append(args, 
			"--icon-name", "dialog-warning",
			"--timeout", "10") // Auto-close after 10 seconds for urgent
	case MessageTypeInvite:
		args = append(args, 
			"--icon-name", "mail-message",
			"--ok-label", "Join Session",
			"--cancel-label", "Dismiss")
	default:
		args = append(args, 
			"--icon-name", "dialog-information",
			"--timeout", "8") // Auto-close after 8 seconds for regular messages
	}

	if os.Getenv("DMUX_DEBUG") != "" {
		fmt.Printf("[DEBUG] Displaying zenity with args: %v\n", args)
	}

	// Display the dialog
	cmd := exec.Command("zenity", args...)
	
	// Run zenity in background so it doesn't block
	go func() {
		err := cmd.Run()

		if m.logger != nil {
			m.logger.Debug("User response from zenity: %v", err)
		}

		if err == nil {
			// If it was an invite and user clicked OK, open terminal
			if msg.Type == MessageTypeInvite {
				if os.Getenv("DMUX_DEBUG") != "" {
					fmt.Printf("[DEBUG] User clicked Join Session\n")
				}
				if err := m.launchTerminalForJoin(msg.From); err != nil && os.Getenv("DMUX_DEBUG") != "" {
					fmt.Printf("[DEBUG] Failed to open terminal for joining: %v\n", err)
				}
			}
		} else if msg.Type != MessageTypeInvite && os.Getenv("DMUX_DEBUG") != "" {
			fmt.Printf("[DEBUG] Failed to display zenity message: %v\n", err)
		}
	}()
}

// displayNotifyMessage shows the message using notify-send
func (m *Messaging) displayNotifyMessage(msg *Message) {
	// Check if notify-send is available
	if _, err := exec.LookPath("notify-send"); err != nil {
		if os.Getenv("DMUX_DEBUG") != "" {
			fmt.Printf("[DEBUG] notify-send not found, falling back to terminal display\n")
		}
		m.displayRealtimeMessage(msg)
		return
	}

	var title, text, urgency string
	
	switch msg.Type {
	case MessageTypeInvite:
		title = "dmux - Session Invitation"
		text = fmt.Sprintf("📨 Invitation from %s\nJoin session: %s\nRun: dmux join %s", msg.From, msg.Data, msg.From)
		urgency = "normal"
	case MessageTypeUrgent:
		title = "dmux - Urgent Message"
		text = fmt.Sprintf("🚨 URGENT from %s\n%s", msg.From, msg.Data)
		urgency = "critical"
	default:
		title = "dmux - Message"
		text = fmt.Sprintf("💬 Message from %s\n%s", msg.From, msg.Data)
		urgency = "normal"
	}

	if os.Getenv("DMUX_DEBUG") != "" {
		fmt.Printf("[DEBUG] Displaying notify-send message: %s\n", text)
	}

	// Display the notification
	cmd := exec.Command("notify-send", "-u", urgency, "-t", "5000", "-i", "mail-message", title, text)
	
	// Run notify-send in background so it doesn't block
	go func() {
		if err := cmd.Run(); err != nil && os.Getenv("DMUX_DEBUG") != "" {
			fmt.Printf("[DEBUG] Failed to display notify-send message: %v\n", err)
		}
	}()
}

// displayAutoMessage automatically chooses the best display method
func (m *Messaging) displayAutoMessage(msg *Message) {
	if m.logger != nil {
		m.logger.Debug("Auto-detecting best display method")
	}

	// Priority order: zenity -> kdialog -> notify-send -> tmux -> terminal
	if _, err := exec.LookPath("zenity"); err == nil {
		if m.logger != nil {
			m.logger.LogDisplayMethod("zenity (auto-detected)")
		}
		m.displayZenityMessage(msg)
		return
	}
	
	if _, err := exec.LookPath("kdialog"); err == nil {
		if m.logger != nil {
			m.logger.LogDisplayMethod("kdialog (auto-detected)")
		}
		m.displayKDialogMessage(msg)
		return
	}
	
	if _, err := exec.LookPath("notify-send"); err == nil {
		if m.logger != nil {
			m.logger.LogDisplayMethod("notify-send (auto-detected)")
		}
		m.displayNotifyMessage(msg)
		return
	}
	
	// Check for tmux sessions
	if os.Getenv("TMUX") != "" || m.hasTmuxSessions() {
		if m.logger != nil {
			m.logger.LogDisplayMethod("tmux (auto-detected)")
		}
		m.displayTmuxMessage(msg)
		return
	}
	
	// Fallback to terminal
	if m.logger != nil {
		m.logger.LogDisplayMethod("terminal (auto-detected fallback)")
	}
	m.displayRealtimeMessage(msg)
}

// CleanupOldMessages removes old message files
func (m *Messaging) CleanupOldMessages() error {
	pattern := "*.msg"
	matches, err := filepath.Glob(filepath.Join(m.config.MessagesDir, pattern))
	if err != nil {
		return err
	}

	cutoff := time.Now().Add(-24 * time.Hour).Unix()

	for _, msgFile := range matches {
		msg, err := m.readMessageFile(msgFile)
		if err != nil {
			continue
		}

		if msg.Timestamp < cutoff {
			os.Remove(msgFile)
		}
	}

	return nil
}