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
	Mode      string // "pair", "view", or "rogue"
	IsReverse bool   // true if this is reverse sharing (client listening)
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
func (m *Manager) StartShare(sessionName string, private bool, inviteUsers []string, mode string) error {
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

	// Use provided session name for registration, but get actual tmux session name for reference
	tmuxSessionName := sessionName
	actualTmuxSession := ""
	
	if m.isInTmuxSession() {
		// Get current tmux session name for reference, but keep user-provided name for sharing
		cmd := exec.Command("tmux", "display-message", "-p", "#S")
		output, err := cmd.Output()
		if err == nil {
			actualTmuxSession = strings.TrimSpace(string(output))
		}
		color.Blue("📋 Sharing current tmux session (%s) as '%s'", actualTmuxSession, tmuxSessionName)
	} else {
		color.Blue("🔄 Starting tmux session '%s'...", tmuxSessionName)
	}

	// Create session object but don't register yet
	session := &Session{
		User:         currentUser,
		Name:         tmuxSessionName,
		Port:         port,
		Started:      time.Now().Unix(),
		PID:          os.Getpid(),
		Private:      private,
		AllowedUsers: inviteUsers,
		Mode:         mode,
	}

	// If already in tmux, start the server in background
	if m.isInTmuxSession() {
		// Start server first, then register on success
		if err := m.startServerInBackground(port); err != nil {
			return err
		}
		
		// Register session only after successful server start
		if err := m.registerSession(session); err != nil {
			color.Yellow("Warning: Failed to register session: %v", err)
		}
		
		// Update port mapping
		if err := m.updatePortMapping(session); err != nil {
			color.Yellow("Warning: Failed to update port mapping: %v", err)
		}
		
		// Send invitations and display success
		m.sendInvitationsAndDisplaySuccess(session, inviteUsers, tmuxSessionName, port, mode)
		return nil
	}

	// When not in tmux, create new session and set up sharing within it
	color.Blue("🔗 Starting shared tmux session...")
	
	// Create the tmux session (or attach if it exists)
	// First check if session already exists
	checkCmd := exec.Command("tmux", "has-session", "-t", tmuxSessionName)
	sessionExists := checkCmd.Run() == nil
	
	if !sessionExists {
		createCmd := exec.Command("tmux", "new-session", "-d", "-s", tmuxSessionName)
		if err := createCmd.Run(); err != nil {
			return fmt.Errorf("failed to create tmux session: %v", err)
		}
	}
	
	// Get current executable path
	jmuxBinary, err := os.Executable()
	if err != nil {
		// Clean up tmux session on failure
		exec.Command("tmux", "kill-session", "-t", tmuxSessionName).Run()
		return fmt.Errorf("failed to get current executable path: %v", err)
	}

	// Build command arguments for internal server
	var args []string
	args = append(args, "_internal_jcat_server")
	args = append(args, fmt.Sprintf("%d", port))
	args = append(args, m.config.SetSizeScript)

	// Add security flag if enabled
	if m.config.Security.Enabled {
		args = append(args, "--secure")
	}

	// Create the command string for tmux
	cmdString := fmt.Sprintf("%s %s", jmuxBinary, strings.Join(args, " "))
	
	// Start the jcat server in a new tmux window within the session
	tmuxCmd := exec.Command("tmux", "new-window", "-t", tmuxSessionName, "-d", "-n", "jcat-server", cmdString)
	if err := tmuxCmd.Run(); err != nil {
		// If window creation fails, clean up the session
		exec.Command("tmux", "kill-session", "-t", tmuxSessionName).Run()
		return fmt.Errorf("failed to start jcat server in tmux window: %v", err)
	}

	// Set up the menu key binding for this session
	menuCommand := fmt.Sprintf("display-popup -E '%s menu'", jmuxBinary)
	bindCmd := exec.Command("tmux", "bind-key", "-t", tmuxSessionName, "M", menuCommand)
	bindCmd.Run() // Ignore errors for key binding

	// Register session only after successful setup
	if err := m.registerSession(session); err != nil {
		color.Yellow("Warning: Failed to register session: %v", err)
	}

	// Update port mapping
	if err := m.updatePortMapping(session); err != nil {
		color.Yellow("Warning: Failed to update port mapping: %v", err)
	}

	color.Green("🚀 jcat server started in tmux session (session-bound)")
	color.Blue("💡 Use 'dmux stop' to stop sharing or Ctrl+A + M for menu")

	// Send invitations and display success
	m.sendInvitationsAndDisplaySuccess(session, inviteUsers, tmuxSessionName, port, mode)

	// Attach to the session - this will block until tmux exits
	// Only attach if we have a terminal available
	if isTerminalAvailable() {
		attachCmd := exec.Command("tmux", "attach-session", "-t", tmuxSessionName)
		attachCmd.Stdin = os.Stdin
		attachCmd.Stdout = os.Stdout
		attachCmd.Stderr = os.Stderr
		return attachCmd.Run()
	} else {
		color.Green("📋 Session sharing started in background")
		color.Yellow("💡 Use 'tmux attach-session -t %s' to connect to the session", tmuxSessionName)
		return nil
	}
}

// JoinSession joins an existing session
func (m *Manager) JoinSession(hostUser, sessionName string, modeOverride string, password string) error {
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

	// Determine the actual mode to use (override takes precedence)
	actualMode := session.Mode
	if modeOverride != "" {
		actualMode = modeOverride
	}

	// If no mode is set in session (backward compatibility), default to pair
	if actualMode == "" {
		actualMode = "pair"
	}

	// Check if this is a local session (same user or local connection)
	if hostUser == currentUser {
		// Local session - use direct tmux connection
		return m.joinLocalSession(session, actualMode)
	}

	// Remote session - use jcat for now (network connection)
	// Get host IP (for now, use localhost or try to resolve)
	hostIP, err := m.resolveHostIP(hostUser)
	if err != nil {
		hostIP = "localhost" // fallback
	}

	// Display mode-specific connection message
	var modeDesc string
	switch actualMode {
	case "view":
		modeDesc = " in view-only mode"
	case "rogue":
		modeDesc = " in rogue mode (independent control)"
	default:
		modeDesc = " in pair mode (shared control)"
	}
	
	color.Cyan("Connecting to %s's session (%s) at %s:%d%s...", hostUser, session.Name, hostIP, session.Port, modeDesc)
	color.Yellow("Press Ctrl+C to disconnect")

	// Connect with jcat client using the specified mode
	if m.config.Security.Enabled {
		secureClient := jcat.NewSecureClientWithMode(fmt.Sprintf("%s:%d", hostIP, session.Port), actualMode, m.config.Security)
		return secureClient.Connect(sessionName, password)
	} else {
		client := jcat.NewClientWithMode(fmt.Sprintf("%s:%d", hostIP, session.Port), actualMode)
		return client.Connect()
	}
}

// StopShare stops sharing sessions
func (m *Manager) StopShare(sessionNames []string) error {
	currentUser := os.Getenv("USER")
	if currentUser == "" {
		return fmt.Errorf("unable to determine current user")
	}

	sessions, err := m.ListUserSessions(currentUser)
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		color.Yellow("No active shared sessions to stop")
		return nil
	}

	// If no specific sessions provided, determine what to stop
	if len(sessionNames) == 0 {
		// If we're in a tmux session, only stop the current session
		if os.Getenv("TMUX") != "" {
			currentTmuxSession, err := m.getCurrentTmuxSession()
			if err == nil {
				// Find the shared session that matches the current tmux session
				found := false
				for _, session := range sessions {
					if session.Name == currentTmuxSession {
						color.Blue("🛑 Stopping sharing for current session '%s'", currentTmuxSession)
						m.stopSession(session)
						found = true
						break
					}
				}
				if !found {
					color.Yellow("Current session '%s' is not being shared", currentTmuxSession)
				}
				return nil
			}
		}
		
		// If not in tmux or couldn't detect current session, stop all
		color.Blue("🛑 Stopping all shared sessions")
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

// sendInvitationsAndDisplaySuccess sends invitations and displays success message
func (m *Manager) sendInvitationsAndDisplaySuccess(session *Session, inviteUsers []string, sessionName string, port int, mode string) {
	// Send invitations to users
	if len(inviteUsers) > 0 {
		for _, user := range inviteUsers {
			inviteMessage := fmt.Sprintf("You're invited to join dmux session '%s' at port %d (mode: %s)", sessionName, port, mode)
			if m.messaging != nil {
				if err := m.messaging.SendMessage(user, "INVITE", inviteMessage); err != nil {
					color.Yellow("Warning: Failed to send invitation to %s: %v", user, err)
				} else {
					color.Green("📨 Invitation sent to %s", user)
				}
			}
		}
	}

	// Display success message with connection details
	color.Green("✅ Session '%s' is now being shared on port %d", sessionName, port)
	if session.Private {
		color.Yellow("🔒 Private session - only invited users can join")
	} else {
		color.Green("🌐 Public session - anyone can join")
	}
	color.Cyan("📞 Others can join with: dmux join %s %s", session.User, sessionName)
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
		
		// Display mode information
		mode := session.Mode
		if mode == "" {
			mode = "pair" // default for backward compatibility
		}
		var modeDesc string
		switch mode {
		case "view":
			modeDesc = "View-only (read-only)"
		case "rogue":
			modeDesc = "Rogue (independent control)"
		default:
			modeDesc = "Pair (shared control)"
		}
		fmt.Printf("  Mode: %s\n", modeDesc)

		color.Yellow("  To join: dmux join %s", session.User)
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
	// Check if port is in use by system
	cmd := exec.Command("sh", "-c", fmt.Sprintf("! lsof -i :%d", port))
	if cmd.Run() != nil {
		return false // Port is in use by system
	}
	
	// Check if port is registered in dmux port mappings
	if m.isPortInPortMappings(port) {
		return false // Port is registered in dmux
	}
	
	// Check if port is used by any active dmux sessions
	if m.isPortInActiveSessions(port) {
		return false // Port is used by active dmux session
	}
	
	return true
}

// isPortInPortMappings checks if port is registered in port_sessions.db
func (m *Manager) isPortInPortMappings(port int) bool {
	portMappings, err := m.readPortMappings()
	if err != nil {
		return false // If we can't read, assume not in use
	}
	
	for _, mapping := range portMappings {
		parts := strings.Split(mapping, ":")
		if len(parts) >= 1 {
			if mappedPort := parts[0]; mappedPort == fmt.Sprintf("%d", port) {
				return true
			}
		}
	}
	return false
}

// isPortInActiveSessions checks if port is used by any active session
func (m *Manager) isPortInActiveSessions(port int) bool {
	sessions, err := m.getAllSessions()
	if err != nil {
		return false // If we can't read, assume not in use
	}
	
	for _, session := range sessions {
		if session.Port == port {
			return true
		}
	}
	return false
}

// isTerminalAvailable checks if we have a terminal available for tmux attach
func isTerminalAvailable() bool {
	// Check if stdin is a terminal
	return os.Getenv("TERM") != "" && isatty(os.Stdin.Fd())
}

// isatty checks if the file descriptor is a terminal
func isatty(fd uintptr) bool {
	// Simple check - try to get terminal size
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.NewFile(fd, "stdin")
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
MODE=%s
`, session.User, session.Name, session.Port, session.Started, session.PID, session.Private, strings.Join(session.AllowedUsers, ","), session.Mode)

	return os.WriteFile(filePath, []byte(content), 0644)
}

func (m *Manager) findUserSession(user, sessionName string) (*Session, error) {
	sessions, err := m.ListUserSessions(user)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions for user %s: %v", user, err)
	}

	if len(sessions) == 0 {
		return nil, fmt.Errorf("no sessions found for user %s", user)
	}

	for _, session := range sessions {
		if sessionName == "" || session.Name == sessionName {
			return session, nil
		}
	}

	// List available sessions for better error message
	var availableSessions []string
	for _, session := range sessions {
		availableSessions = append(availableSessions, session.Name)
	}

	if sessionName == "" {
		return nil, fmt.Errorf("no default session found for user %s. Available sessions: %v", user, availableSessions)
	} else {
		return nil, fmt.Errorf("session '%s' not found for user %s. Available sessions: %v", sessionName, user, availableSessions)
	}
}

func (m *Manager) ListUserSessions(user string) ([]*Session, error) {
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
		case "MODE":
			session.Mode = value
		}
	}

	return session, scanner.Err()
}

func (m *Manager) stopSession(session *Session) {
	color.Yellow("Stopping sharing for session '%s'...", session.Name)

	// Find and kill only the jcat server process, not the tmux session
	cmd := exec.Command("pkill", "-f", fmt.Sprintf("_internal_jcat_server %d", session.Port))
	if err := cmd.Run(); err != nil {
		// Try alternative method using lsof and port
		cmd = exec.Command("sh", "-c", fmt.Sprintf("lsof -ti:%d | xargs -r kill", session.Port))
		cmd.Run() // Ignore errors - process might already be dead
	}

	// Remove session file
	fileName := fmt.Sprintf("%s_%s.session", session.User, session.Name)
	filePath := filepath.Join(m.config.SessionsDir, fileName)
	os.Remove(filePath)

	// Remove from port_sessions.db
	if err := m.removePortMapping(session.Port); err != nil {
		color.Yellow("Warning: Failed to update port mapping: %v", err)
	}

	color.Green("✓ Sharing stopped for session '%s' (tmux session remains active)", session.Name)
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

func (m *Manager) isInTmuxSession() bool {
	return os.Getenv("TMUX") != ""
}

// joinLocalSession joins a local session using direct tmux commands
func (m *Manager) joinLocalSession(session *Session, mode string) error {
	// For local sessions, we first try to find the session in the default tmux server
	// If that fails, we'll check if it exists and report an error
	
	// TODO: Dory
	//// Construct the tmux socket path based on the session port
	//// This assumes the session is using a socket file based on port
	//socketPath := fmt.Sprintf("/tmp/tmux-%d/default", session.Port)
	//for every command need to add "-S", socketPath

	// First, check if the session exists in the default tmux server
	checkCmd := exec.Command("tmux", "has-session", "-t", session.Name)
	sessionExists := checkCmd.Run() == nil
	
	if !sessionExists {
		// Session might be running in a custom socket, try to find it
		return fmt.Errorf("session '%s' not found in default tmux server. For local shared sessions, please ensure the session is accessible via the default tmux server", session.Name)
	}
	
	var cmd *exec.Cmd
	var modeDesc string
	
	switch mode {
	case "view":
		// View-only mode: attach with read-only flag
		cmd = exec.Command("tmux", "attach-session", "-t", session.Name, "-r")
		modeDesc = "view-only (read-only)"
		color.Cyan("Joining %s's session (%s) in %s mode...", session.User, session.Name, modeDesc)
		color.Yellow("You are in read-only mode. Press Ctrl+C to disconnect")
		
	case "rogue":
		// Rogue mode: create new session that shares the same server
		cmd = exec.Command("tmux", "new-session", "-t", session.Name)
		modeDesc = "rogue (independent control)"
		color.Cyan("Joining %s's session (%s) in %s mode...", session.User, session.Name, modeDesc)
		color.Yellow("You have independent control. Press Ctrl+C to disconnect")
		
	default: // pair mode
		// Pair mode: standard attach (shared control)
		cmd = exec.Command("tmux", "attach-session", "-t", session.Name)
		modeDesc = "pair (shared control)"
		color.Cyan("Joining %s's session (%s) in %s mode...", session.User, session.Name, modeDesc)
		color.Yellow("You have shared control. Press Ctrl+C to disconnect")
	}
	
	// Set up command to run interactively
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	return cmd.Run()
}

// updatePortMapping adds/updates entry in port_sessions.db
func (m *Manager) updatePortMapping(session *Session) error {
	// Read existing entries
	portMappings, err := m.readPortMappings()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Add/update the entry for this session
	entry := fmt.Sprintf("%d:%s:%s", session.Port, session.User, session.Name)
	
	// Remove any existing entry for this port
	var updatedMappings []string
	for _, mapping := range portMappings {
		parts := strings.Split(mapping, ":")
		if len(parts) >= 1 {
			if existingPort := parts[0]; existingPort != fmt.Sprintf("%d", session.Port) {
				updatedMappings = append(updatedMappings, mapping)
			}
		}
	}
	
	// Add the new entry
	updatedMappings = append(updatedMappings, entry)
	
	// Write back to file
	content := strings.Join(updatedMappings, "\n")
	if content != "" {
		content += "\n"
	}
	
	return os.WriteFile(m.config.PortMapFile, []byte(content), 0644)
}

// removePortMapping removes entry from port_sessions.db
func (m *Manager) removePortMapping(port int) error {
	portMappings, err := m.readPortMappings()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Remove the entry for this port
	var updatedMappings []string
	for _, mapping := range portMappings {
		parts := strings.Split(mapping, ":")
		if len(parts) >= 1 {
			if existingPort := parts[0]; existingPort != fmt.Sprintf("%d", port) {
				updatedMappings = append(updatedMappings, mapping)
			}
		}
	}
	
	// Write back to file
	content := strings.Join(updatedMappings, "\n")
	if content != "" {
		content += "\n"
	}
	
	return os.WriteFile(m.config.PortMapFile, []byte(content), 0644)
}

// readPortMappings reads all entries from port_sessions.db
func (m *Manager) readPortMappings() ([]string, error) {
	content, err := os.ReadFile(m.config.PortMapFile)
	if err != nil {
		return nil, err
	}
	
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	var mappings []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			mappings = append(mappings, line)
		}
	}
	
	return mappings, nil
}

// startServerInBackground starts the jcat server within the tmux session context
func (m *Manager) startServerInBackground(port int) error {
	// Get current tmux session name
	currentSession, err := m.getCurrentTmuxSession()
	if err != nil {
		return fmt.Errorf("failed to get current tmux session: %v", err)
	}

	// Get current executable path
	jmuxBinary, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %v", err)
	}

	// Build command arguments for internal server
	var args []string
	args = append(args, "_internal_jcat_server")
	args = append(args, fmt.Sprintf("%d", port))
	args = append(args, m.config.SetSizeScript)

	// Add security flag if enabled
	if m.config.Security.Enabled {
		args = append(args, "--secure")
	}

	// Create the command string for tmux
	cmdString := fmt.Sprintf("%s %s", jmuxBinary, strings.Join(args, " "))
	
	// Start the jcat server in a new tmux window within the current session
	// This ensures the server dies when the session is killed
	tmuxCmd := exec.Command("tmux", "new-window", "-t", currentSession, "-d", "-n", "jcat-server", cmdString)
	
	if err := tmuxCmd.Run(); err != nil {
		return fmt.Errorf("failed to start jcat server in tmux window: %v", err)
	}

	color.Green("🚀 jcat server started in tmux session (session-bound)")
	color.Blue("💡 Use 'dmux stop' to stop sharing or Ctrl+A + M for menu")
	
	// Set up the menu key binding
	if err := m.setupMenuKeyBinding(); err != nil {
		color.Yellow("Warning: Could not set up menu key binding: %v", err)
	}
	
	return nil
}

// getCurrentTmuxSession gets the current tmux session name
func (m *Manager) getCurrentTmuxSession() (string, error) {
	cmd := exec.Command("tmux", "display-message", "-p", "#S")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// setupMenuKeyBinding sets up the Ctrl+A + M key binding for the dmux menu
func (m *Manager) setupMenuKeyBinding() error {
	// Get current executable path
	jmuxBinary, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %v", err)
	}

	// Create the menu command
	menuCommand := fmt.Sprintf("display-popup -E '%s menu'", jmuxBinary)
	
	// Set up the key binding: Ctrl+A + M
	cmd := exec.Command("tmux", "bind-key", "M", menuCommand)
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set up menu key binding: %v", err)
	}
	
	return nil
}

// StartReverseShare starts reverse sharing - client listens for connections from hosts
func (m *Manager) StartReverseShare(inviteUsers []string, password string, private bool, mode string) error {
	currentUser := os.Getenv("USER")
	if currentUser == "" {
		return fmt.Errorf("unable to determine current user")
	}

	// Find an available port
	port, err := m.findAvailablePort()
	if err != nil {
		return fmt.Errorf("no available ports: %v", err)
	}

	// Create reverse share session
	session := &Session{
		User:         currentUser,
		Name:         fmt.Sprintf("reverse-%d", time.Now().Unix()),
		Port:         port,
		Started:      time.Now().Unix(),
		Private:      private,
		AllowedUsers: inviteUsers,
		Mode:         mode,
		IsReverse:    true, // Mark as reverse sharing
	}

	// Set default mode if empty
	if session.Mode == "" {
		session.Mode = "pair"
	}

	// Register the reverse session
	if err := m.registerSession(session); err != nil {
		return fmt.Errorf("failed to register reverse session: %v", err)
	}

	color.Green("🔄 Starting reverse sharing session '%s'", session.Name)
	color.Blue("🎯 Listening on port %d for incoming connections", port)

	// Start listening server in background
	go func() {
		if err := m.startReverseListener(session); err != nil {
			color.Red("❌ Reverse sharing server error: %v", err)
		}
	}()

	// Send invitation messages to users
	m.sendReverseInvitations(session, inviteUsers, password)

	color.Green("✅ Reverse sharing started - waiting for connections")
	color.Yellow("💡 Invited users can join with: dmux join-share %s", currentUser)
	color.Blue("🛑 Use 'dmux stop %s' to stop reverse sharing", session.Name)

	return nil
}

// startReverseListener starts the listening server for reverse sharing
func (m *Manager) startReverseListener(session *Session) error {
	// This would start a jcat server that waits for connections
	// When someone connects, instead of sharing the local session,
	// we request their session to be shared with us
	
	// For now, start a simple server that handles the reverse connection
	// Bind to all interfaces (0.0.0.0) so it's accessible from other machines
	server := jcat.NewServer(fmt.Sprintf("0.0.0.0:%d", session.Port), m.config.SetSizeScript)
	return server.Start()
}

// sendReverseInvitations sends invitation messages for reverse sharing
func (m *Manager) sendReverseInvitations(session *Session, users []string, password string) {
	currentUser := os.Getenv("USER")
	hostname, _ := os.Hostname()
	
	for _, user := range users {
		user = strings.TrimSpace(user)
		if user == "" {
			continue
		}

		// Create invitation message
		timestamp := time.Now().Unix()
		filename := fmt.Sprintf("%s_reverse_invite_%d.msg", user, timestamp)
		filepath := filepath.Join(m.config.SharedDir, "messages", filename)

		var content string
		if password != "" {
			content = fmt.Sprintf(`FROM=%s
TYPE=REVERSE_INVITE
TIMESTAMP=%d
SESSION=%s
PORT=%d
HOSTNAME=%s
PASSWORD_PROTECTED=true
PRIVATE=%t
MODE=%s
DATA=%s is requesting to share a session with you (password protected)
PRIORITY=high`, currentUser, timestamp, session.Name, session.Port, hostname, session.Private, session.Mode, currentUser)
		} else {
			content = fmt.Sprintf(`FROM=%s
TYPE=REVERSE_INVITE
TIMESTAMP=%d
SESSION=%s
PORT=%d
HOSTNAME=%s
PASSWORD_PROTECTED=false
PRIVATE=%t
MODE=%s
DATA=%s is requesting to share a session with you
PRIORITY=high`, currentUser, timestamp, session.Name, session.Port, hostname, session.Private, session.Mode, currentUser)
		}

		if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
			color.Yellow("⚠️  Could not send invitation to %s: %v", user, err)
		} else {
			color.Green("📨 Invitation sent to %s", user)
		}
	}
}

// JoinReverseShare joins a reverse sharing session by reading invitation messages
func (m *Manager) JoinReverseShare(hostUser, sessionName string, modeOverride string, password string) error {
	currentUser := os.Getenv("USER")
	if currentUser == "" {
		return fmt.Errorf("unable to determine current user")
	}

	// Find reverse invitation message
	invitation, err := m.findReverseInvitation(hostUser, sessionName)
	if err != nil {
		return fmt.Errorf("no reverse sharing invitation found from %s: %v", hostUser, err)
	}

	// Check permissions for private sessions
	if invitation.Private && !m.isUserAllowed(currentUser, invitation.AllowedUsers) {
		return fmt.Errorf("access denied: private session")
	}

	// Determine the actual mode to use (override takes precedence)
	actualMode := invitation.Mode
	if modeOverride != "" {
		actualMode = modeOverride
	}

	// If no mode is set in invitation (backward compatibility), default to pair
	if actualMode == "" {
		actualMode = "pair"
	}

	// Display mode-specific connection message
	var modeDesc string
	switch actualMode {
	case "view":
		modeDesc = " in view-only mode (read-only)"
	case "rogue":
		modeDesc = " in rogue mode (independent control)"
	default:
		modeDesc = " in pair mode (shared control)"
	}

	color.Blue("🔗 Connecting to %s's reverse sharing session (%s) at %s:%d%s...", 
		hostUser, invitation.SessionName, invitation.HostIP, invitation.Port, modeDesc)
	color.Yellow("Press Ctrl+C to disconnect")

	// Connect using jcat client
	client := jcat.NewClientWithMode(fmt.Sprintf("%s:%d", invitation.HostIP, invitation.Port), actualMode)
	return client.Connect()
}

// ReverseInvitation represents a reverse sharing invitation
type ReverseInvitation struct {
	From               string
	SessionName        string
	Port               int
	HostIP             string
	PasswordProtected  bool
	Private            bool
	Mode               string
	AllowedUsers       []string
	Timestamp          int64
}

// findReverseInvitation finds a reverse sharing invitation message
func (m *Manager) findReverseInvitation(hostUser, sessionName string) (*ReverseInvitation, error) {
	currentUser := os.Getenv("USER")
	messagesDir := filepath.Join(m.config.SharedDir, "messages")

	// Pattern to match reverse invitation messages for current user
	pattern := fmt.Sprintf("%s_reverse_invite_*.msg", currentUser)
	matches, err := filepath.Glob(filepath.Join(messagesDir, pattern))
	if err != nil {
		return nil, fmt.Errorf("failed to search for invitation messages: %v", err)
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no reverse sharing invitations found")
	}

	// Find the most recent invitation from the specified host user
	var latestInvitation *ReverseInvitation
	var latestTimestamp int64

	for _, msgFile := range matches {
		invitation, err := m.parseReverseInvitation(msgFile)
		if err != nil {
			continue // Skip invalid messages
		}

		// Check if it's from the right host user
		if invitation.From != hostUser {
			continue
		}

		// Check if session name matches (if specified)
		if sessionName != "" && invitation.SessionName != sessionName {
			continue
		}

		// Use the most recent invitation
		if invitation.Timestamp > latestTimestamp {
			latestInvitation = invitation
			latestTimestamp = invitation.Timestamp
		}
	}

	if latestInvitation == nil {
		if sessionName != "" {
			return nil, fmt.Errorf("no reverse sharing invitation found from %s for session %s", hostUser, sessionName)
		}
		return nil, fmt.Errorf("no reverse sharing invitation found from %s", hostUser)
	}

	return latestInvitation, nil
}

// parseReverseInvitation parses a reverse invitation message file
func (m *Manager) parseReverseInvitation(msgFile string) (*ReverseInvitation, error) {
	content, err := os.ReadFile(msgFile)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	invitation := &ReverseInvitation{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "FROM":
			invitation.From = value
		case "SESSION":
			invitation.SessionName = value
		case "PORT":
			if port, err := strconv.Atoi(value); err == nil {
				invitation.Port = port
			}
		case "HOSTNAME":
			invitation.HostIP = value
		case "PASSWORD_PROTECTED":
			invitation.PasswordProtected = value == "true"
		case "PRIVATE":
			invitation.Private = value == "true"
		case "MODE":
			invitation.Mode = value
		case "TIMESTAMP":
			if ts, err := strconv.ParseInt(value, 10, 64); err == nil {
				invitation.Timestamp = ts
			}
		}
	}

	// Validate required fields
	if invitation.From == "" || invitation.Port == 0 || invitation.HostIP == "" {
		return nil, fmt.Errorf("invalid invitation message format")
	}

	return invitation, nil
}