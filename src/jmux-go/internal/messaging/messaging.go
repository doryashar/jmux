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
}

// NewMessaging creates a new messaging instance
func NewMessaging(cfg *config.Config) *Messaging {
	return &Messaging{
		config: cfg,
		done:   make(chan bool),
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

	// Debug: Log that monitoring started
	if os.Getenv("DMUX_DEBUG") != "" {
		fmt.Printf("[DEBUG] Live monitoring started for: %s\n", m.config.MessagesDir)
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
						if os.Getenv("DMUX_DEBUG") != "" {
							fmt.Printf("[DEBUG] New message file detected: %s\n", event.Name)
						}
						m.handleNewMessage(event.Name)
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				fmt.Printf("Watcher error: %v\n", err)
			case <-m.done:
				return
			}
		}
	}()

	return nil
}

// StopLiveMonitoring stops the live message monitoring
func (m *Messaging) StopLiveMonitoring() {
	if m.watcher != nil {
		m.done <- true
		m.watcher.Close()
	}
}

// handleNewMessage processes a new message file
func (m *Messaging) handleNewMessage(msgFile string) {
	// Small delay to ensure file is fully written
	time.Sleep(100 * time.Millisecond)

	msg, err := m.readMessageFile(msgFile)
	if err != nil {
		if os.Getenv("DMUX_DEBUG") != "" {
			fmt.Printf("[DEBUG] Failed to read message file %s: %v\n", msgFile, err)
		}
		return
	}

	// Check if message is for current user
	currentUser := os.Getenv("USER")
	if currentUser == "" {
		if os.Getenv("DMUX_DEBUG") != "" {
			fmt.Printf("[DEBUG] No USER environment variable\n")
		}
		return
	}

	expectedPrefix := currentUser + "_"
	fileName := filepath.Base(msgFile)
	if !strings.HasPrefix(fileName, expectedPrefix) {
		if os.Getenv("DMUX_DEBUG") != "" {
			fmt.Printf("[DEBUG] Message file %s not for user %s\n", fileName, currentUser)
		}
		return
	}

	if os.Getenv("DMUX_DEBUG") != "" {
		fmt.Printf("[DEBUG] Processing message for user %s: %s\n", currentUser, msg.Data)
	}

	// Try multiple display methods
	m.displayRealtimeMessage(msg)
	
	// Also try tmux display if we're in tmux
	if os.Getenv("TMUX") != "" {
		m.displayTmuxMessage(msg)
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
		tmuxMsg = fmt.Sprintf("INVITE from %s: Join session '%s' | dmux join %s", msg.From, msg.Data, msg.From)
	case MessageTypeUrgent:
		tmuxMsg = fmt.Sprintf("URGENT from %s: %s", msg.From, msg.Data)
	default:
		tmuxMsg = fmt.Sprintf("Message from %s: %s", msg.From, msg.Data)
	}

	// Use tmux display-message to show the notification
	cmd := exec.Command("tmux", "display-message", "-d", "5000", tmuxMsg)
	if err := cmd.Run(); err != nil && os.Getenv("DMUX_DEBUG") != "" {
		fmt.Printf("[DEBUG] Failed to display tmux message: %v\n", err)
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