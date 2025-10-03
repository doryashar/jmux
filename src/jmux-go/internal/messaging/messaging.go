package messaging

import (
	"bufio"
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

// StartLiveMonitoring starts the live message monitoring
func (m *Messaging) StartLiveMonitoring() error {
	if !m.config.RealtimeEnabled {
		return nil
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	m.watcher = watcher

	// Watch the messages directory
	err = watcher.Add(m.config.MessagesDir)
	if err != nil {
		return err
	}

	// Log that monitoring started
	if m.logger != nil {
		m.logger.Info("Live monitoring started for: %s", m.config.MessagesDir)
	}

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Create == fsnotify.Create {
					// New message file created
					if strings.HasSuffix(event.Name, ".msg") {
						if m.logger != nil {
							m.logger.Debug("New message file detected: %s", event.Name)
						}
						m.handleNewMessage(event.Name)
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				if m.logger != nil {
					m.logger.Error("Watcher error: %v", err)
				}
			case <-m.done:
				return
			}
		}
	}()

	return nil
}

// StopLiveMonitoring stops the live message monitoring
func (m *Messaging) StopLiveMonitoring() {
	if m.logger != nil {
		m.logger.LogMonitorStop()
	}
	
	if m.watcher != nil {
		m.done <- true
		m.watcher.Close()
	}
	
	// Close logger
	if m.logger != nil {
		m.logger.Close()
	}
}

// handleNewMessage processes a new message file
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

// SendMessage sends a message to a user
func (m *Messaging) SendMessage(toUser string, msgType MessageType, data string) error {
	timestamp := time.Now().Unix()
	fileName := fmt.Sprintf("%s_%d.msg", toUser, timestamp)
	filePath := filepath.Join(m.config.MessagesDir, fileName)

	currentUser := os.Getenv("USER")
	if currentUser == "" {
		currentUser = "unknown"
	}

	content := fmt.Sprintf(`FROM=%s
TYPE=%s
TIMESTAMP=%d
DATA=%s
PRIORITY=normal
`, currentUser, msgType, timestamp, data)

	return os.WriteFile(filePath, []byte(content), 0644)
}

// ReadMessages reads and displays messages for current user
func (m *Messaging) ReadMessages() error {
	currentUser := os.Getenv("USER")
	if currentUser == "" {
		return fmt.Errorf("unable to determine current user")
	}

	pattern := currentUser + "_*.msg"
	matches, err := filepath.Glob(filepath.Join(m.config.MessagesDir, pattern))
	if err != nil {
		return err
	}

	if len(matches) == 0 {
		color.Yellow("No new messages")
		return nil
	}

	color.Blue("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	color.Green("New Messages")
	color.Blue("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for _, msgFile := range matches {
		msg, err := m.readMessageFile(msgFile)
		if err != nil {
			continue
		}

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

		// Remove the message after reading
		os.Remove(msgFile)
	}

	fmt.Println()
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

	// Display the dialog
	cmd := exec.Command("kdialog", dialogType, text, "--title", title, "--icon", "mail-message")
	
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