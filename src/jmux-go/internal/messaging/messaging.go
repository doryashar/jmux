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

	// Start tail-based monitoring
	go m.tailUserMessages(userMessageFile)

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

// ReadMessages reads and displays messages for current user from their message file
func (m *Messaging) ReadMessages() error {
	currentUser := os.Getenv("USER")
	if currentUser == "" {
		return fmt.Errorf("unable to determine current user")
	}

	userMessageFile := filepath.Join(m.config.MessagesDir, currentUser+".messages")
	
	// Check if user message file exists
	if _, err := os.Stat(userMessageFile); os.IsNotExist(err) {
		color.Yellow("No new messages")
		return nil
	}

	// Read all messages from the file
	file, err := os.Open(userMessageFile)
	if err != nil {
		return fmt.Errorf("failed to open user message file: %v", err)
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

	if len(messages) == 0 {
		color.Yellow("No new messages")
		return nil
	}

	color.Blue("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	color.Green("New Messages")
	color.Blue("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for _, msg := range messages {
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

	// Clear messages by truncating the file
	if err := os.Truncate(userMessageFile, 0); err != nil {
		return fmt.Errorf("failed to clear messages: %v", err)
	}

	return nil
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
		if err := cmd.Run(); err != nil && os.Getenv("DMUX_DEBUG") != "" {
			fmt.Printf("[DEBUG] Failed to display kdialog message: %v\n", err)
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

	// Priority order: kdialog -> notify-send -> tmux -> terminal
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