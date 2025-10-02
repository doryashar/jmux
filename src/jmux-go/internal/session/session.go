package session

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"jmux/internal/config"
	"jmux/internal/jcat"
	"jmux/internal/messaging"
)

// Session represents a jmux session
type Session struct {
	User      string
	Name      string
	Port      int
	Started   int64
	PID       int
	Private   bool
	AllowedUsers []string
}

// Manager handles session management
type Manager struct {
	config    *config.Config
	messaging *messaging.Messaging
}

// NewManager creates a new session manager
func NewManager(cfg *config.Config, msg *messaging.Messaging) *Manager {
	return &Manager{
		config:    cfg,
		messaging: msg,
	}
}

// StartShare starts sharing a tmux session
func (m *Manager) StartShare(sessionName string, private bool, inviteUsers []string) error {
	currentUser := os.Getenv("USER")
	if currentUser == "" {
		return fmt.Errorf("unable to determine current user")
	}

	// Find available port
	port, err := m.findAvailablePort()
	if err != nil {
		return err
	}

	// Generate session name if not provided
	if sessionName == "" {
		sessionName = fmt.Sprintf("session-%d", time.Now().Unix())
	}

	// Register the session
	session := &Session{
		User:         currentUser,
		Name:         sessionName,
		Port:         port,
		Started:      time.Now().Unix(),
		PID:          os.Getpid(),
		Private:      private,
		AllowedUsers: inviteUsers,
	}

	if err := m.registerSession(session); err != nil {
		return err
	}

	// Send invitations
	for _, user := range inviteUsers {
		err := m.messaging.SendMessage(user, messaging.MessageTypeInvite, sessionName)
		if err != nil {
			color.Yellow("Failed to send invitation to %s: %v", user, err)
		}
	}

	color.Green("✓ Session '%s' shared on port %d", sessionName, port)
	if len(inviteUsers) > 0 {
		color.Cyan("📧 Invitations sent to: %s", strings.Join(inviteUsers, ", "))
	}

	// Start the jcat server
	server := jcat.NewServer(fmt.Sprintf(":%d", port), m.config.SetSizeScript)
	return server.Start()
}

// JoinSession joins an existing session
func (m *Manager) JoinSession(hostUser, sessionName string) error {
	// Find the session
	session, err := m.findUserSession(hostUser, sessionName)
	if err != nil {
		return err
	}

	currentUser := os.Getenv("USER")
	if currentUser == "" {
		return fmt.Errorf("unable to determine current user")
	}

	// Check permissions for private sessions
	if session.Private && !m.isUserAllowed(currentUser, session.AllowedUsers) {
		return fmt.Errorf("access denied: private session")
	}

	// Get host IP (for now, use localhost or try to resolve)
	hostIP, err := m.resolveHostIP(hostUser)
	if err != nil {
		hostIP = "localhost" // fallback
	}

	color.Cyan("Connecting to %s's session (%s) at %s:%d...", hostUser, session.Name, hostIP, session.Port)
	color.Yellow("Press Ctrl+C to disconnect")

	// Connect with jcat client
	client := jcat.NewClient(fmt.Sprintf("%s:%d", hostIP, session.Port))
	return client.Connect()
}

// StopShare stops sharing sessions
func (m *Manager) StopShare(sessionNames []string) error {
	currentUser := os.Getenv("USER")
	if currentUser == "" {
		return fmt.Errorf("unable to determine current user")
	}

	sessions, err := m.listUserSessions(currentUser)
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		color.Yellow("No active shared sessions to stop")
		return nil
	}

	// If no specific sessions provided, stop all
	if len(sessionNames) == 0 {
		for _, session := range sessions {
			m.stopSession(session)
		}
		return nil
	}

	// Stop specific sessions
	for _, sessionName := range sessionNames {
		found := false
		for _, session := range sessions {
			if session.Name == sessionName {
				m.stopSession(session)
				found = true
				break
			}
		}
		if !found {
			color.Yellow("Session '%s' not found", sessionName)
		}
	}

	return nil
}

// ListSessions lists all active sessions
func (m *Manager) ListSessions() error {
	sessions, err := m.getAllSessions()
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		color.Yellow("No active shared sessions")
		return nil
	}

	color.Blue("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	color.Green("Active Shared Sessions")
	color.Blue("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for _, session := range sessions {
		startTime := time.Unix(session.Started, 0)
		duration := time.Since(startTime).Round(time.Second)

		fmt.Printf("\n")
		color.Cyan("User: %s", session.User)
		fmt.Printf("  Session: %s\n", session.Name)
		fmt.Printf("  Port: %d\n", session.Port)
		fmt.Printf("  Started: %s (%s ago)\n", startTime.Format("15:04:05"), duration)

		if session.Private {
			color.Red("  Private session")
			if len(session.AllowedUsers) > 0 {
				fmt.Printf("  Allowed users: %s\n", strings.Join(session.AllowedUsers, ", "))
			}
		} else {
			color.Green("  Public session")
		}

		color.Yellow("  To join: jmux join %s", session.User)
	}

	fmt.Println()
	return nil
}

// Helper functions

func (m *Manager) findAvailablePort() (int, error) {
	// Start from the configured port and find the next available
	for port := m.config.Port; port < m.config.Port+100; port++ {
		if m.isPortAvailable(port) {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available ports found")
}

func (m *Manager) isPortAvailable(port int) bool {
	// Simple check by trying to bind to the port
	cmd := exec.Command("sh", "-c", fmt.Sprintf("! lsof -i :%d", port))
	return cmd.Run() == nil
}

func (m *Manager) registerSession(session *Session) error {
	fileName := fmt.Sprintf("%s_%s.session", session.User, session.Name)
	filePath := filepath.Join(m.config.SessionsDir, fileName)

	content := fmt.Sprintf(`USER=%s
SESSION=%s
PORT=%d
STARTED=%d
PID=%d
PRIVATE=%t
ALLOWED_USERS=%s
`, session.User, session.Name, session.Port, session.Started, session.PID, session.Private, strings.Join(session.AllowedUsers, ","))

	return os.WriteFile(filePath, []byte(content), 0644)
}

func (m *Manager) findUserSession(user, sessionName string) (*Session, error) {
	sessions, err := m.listUserSessions(user)
	if err != nil {
		return nil, err
	}

	for _, session := range sessions {
		if sessionName == "" || session.Name == sessionName {
			return session, nil
		}
	}

	return nil, fmt.Errorf("session not found")
}

func (m *Manager) listUserSessions(user string) ([]*Session, error) {
	pattern := user + "_*.session"
	matches, err := filepath.Glob(filepath.Join(m.config.SessionsDir, pattern))
	if err != nil {
		return nil, err
	}

	var sessions []*Session
	for _, sessionFile := range matches {
		session, err := m.readSessionFile(sessionFile)
		if err != nil {
			continue
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

func (m *Manager) getAllSessions() ([]*Session, error) {
	pattern := "*.session"
	matches, err := filepath.Glob(filepath.Join(m.config.SessionsDir, pattern))
	if err != nil {
		return nil, err
	}

	var sessions []*Session
	for _, sessionFile := range matches {
		session, err := m.readSessionFile(sessionFile)
		if err != nil {
			continue
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

func (m *Manager) readSessionFile(filePath string) (*Session, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	session := &Session{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
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
		case "STARTED":
			if started, err := strconv.ParseInt(value, 10, 64); err == nil {
				session.Started = started
			}
		case "PID":
			if pid, err := strconv.Atoi(value); err == nil {
				session.PID = pid
			}
		case "PRIVATE":
			session.Private = value == "true"
		case "ALLOWED_USERS":
			if value != "" {
				session.AllowedUsers = strings.Split(value, ",")
			}
		}
	}

	return session, scanner.Err()
}

func (m *Manager) stopSession(session *Session) {
	color.Yellow("Stopping session '%s'...", session.Name)

	// Kill the process if it's still running
	if session.PID > 0 {
		process, err := os.FindProcess(session.PID)
		if err == nil {
			process.Kill()
		}
	}

	// Remove session file
	fileName := fmt.Sprintf("%s_%s.session", session.User, session.Name)
	filePath := filepath.Join(m.config.SessionsDir, fileName)
	os.Remove(filePath)

	color.Green("✓ Session '%s' stopped", session.Name)
}

func (m *Manager) isUserAllowed(user string, allowedUsers []string) bool {
	for _, allowed := range allowedUsers {
		if allowed == user {
			return true
		}
	}
	return false
}

func (m *Manager) resolveHostIP(hostUser string) (string, error) {
	// Try to read from users database
	usersFile := m.config.UsersFile
	file, err := os.Open(usersFile)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) >= 2 && parts[0] == hostUser {
			return parts[1], nil
		}
	}

	return "", fmt.Errorf("user %s not found", hostUser)
}