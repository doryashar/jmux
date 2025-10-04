package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config holds all jmux configuration
type Config struct {
	Port                    int
	SharedDir              string
	ConfigDir              string
	SetSizeScript          string
	MessagesDir            string
	UsersFile              string
	SessionsDir            string
	PortMapFile            string
	RealtimeEnabled        bool
	NotificationDuration   int
	WatcherPIDFile         string
	MonitorPIDFile         string
	MonitorLogFile         string
	MessageDisplayMethod   string // "kdialog", "terminal", "tmux"
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	// defaultSharedDir := filepath.Join(homeDir, ".jmux", "shared")
	sharedDir := getEnvOrDefault("JMUX_SHARED_DIR", "/projects/common/work/dory/jmux")
	configDir := filepath.Join(homeDir, ".config", "jmux")

	return &Config{
		Port:                    getEnvOrDefaultInt("JMUX_PORT", 12345),
		SharedDir:              sharedDir,
		ConfigDir:              configDir,
		SetSizeScript:          filepath.Join(configDir, "setsize.sh"),
		MessagesDir:            filepath.Join(sharedDir, "messages"),
		UsersFile:              filepath.Join(sharedDir, "users.db"),
		SessionsDir:            filepath.Join(sharedDir, "sessions"),
		PortMapFile:            filepath.Join(sharedDir, "port_sessions.db"),
		RealtimeEnabled:        getEnvOrDefaultBool("JMUX_REALTIME", true),
		NotificationDuration:   getEnvOrDefaultInt("JMUX_NOTIFICATION_DURATION", 5),
		WatcherPIDFile:         filepath.Join(configDir, "watcher.pid"),
		MonitorPIDFile:         filepath.Join("/tmp", "dmux-monitor-"+os.Getenv("USER")+".pid"),
		MonitorLogFile:         filepath.Join(configDir, "monitor.log"),
		MessageDisplayMethod:   getEnvOrDefault("DMUX_MESSAGE_DISPLAY", "auto"),
	}
}

// EnsureDirectories creates necessary directories
func (c *Config) EnsureDirectories() error {
	dirs := []string{
		c.ConfigDir,
		c.MessagesDir,
		c.SessionsDir,
		filepath.Dir(c.UsersFile),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Ensure setsize script exists and is current
	if err := c.EnsureSetSizeScript(); err != nil {
		return err
	}

	return nil
}

// EnsureSetSizeScript creates or updates the setsize script if needed
func (c *Config) EnsureSetSizeScript() error {
	needsUpdate := false

	// Check if script doesn't exist
	if _, err := os.Stat(c.SetSizeScript); os.IsNotExist(err) {
		needsUpdate = true
	} else if err == nil {
		// Check if script is outdated (doesn't contain the profile source line)
		needsUpdate = !c.isSetSizeScriptCurrent()
	} else {
		return err
	}

	if needsUpdate {
		if err := c.createSetSizeScript(); err != nil {
			return err
		}
		if err := c.createProfileScript(); err != nil {
			return err
		}
	}

	return nil
}

// isSetSizeScriptCurrent checks if the setsize script contains the profile sourcing
func (c *Config) isSetSizeScriptCurrent() bool {
	file, err := os.Open(c.SetSizeScript)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "Source jmux profile") {
			return true
		}
	}
	return false
}

// createSetSizeScript creates the setsize.sh script
func (c *Config) createSetSizeScript() error {
	content := `#!/bin/bash
#stty rows 50 cols 254

# Source jmux profile to ensure PATH and jmux availability
if [[ -f "$HOME/.config/jmux/profile.sh" ]]; then
    source "$HOME/.config/jmux/profile.sh"
fi

# Determine session name from port mapping or fallback to hostname
SESSION_NAME=""
if [[ -n "${SOCAT_SOCKPORT:-}" ]]; then
    # Get session name from port mapping
    PORT_MAP_FILE="${JMUX_SHARED_DIR:-/projects/common/work/dory/jmux}/port_sessions.db"
    if [[ -f "$PORT_MAP_FILE" ]]; then
        SESSION_NAME=$(grep "^${SOCAT_SOCKPORT}:" "$PORT_MAP_FILE" 2>/dev/null | head -1 | cut -d: -f3)
    fi
fi

# Fallback to hostname if no session name found
if [[ -z "$SESSION_NAME" ]]; then
    SESSION_NAME="${HOSTNAME:-$(hostname 2>/dev/null || echo "jmux-session")}"
fi

# Final safety check - ensure session name is never empty
if [[ -z "$SESSION_NAME" ]]; then
    SESSION_NAME="jmux-fallback-session"
fi

# Try multiple common tmux locations
if command -v tmux &> /dev/null; then
    exec tmux new -A -s "$SESSION_NAME"
elif [[ -x "/bin/tmux" ]]; then
    exec /bin/tmux new -A -s "$SESSION_NAME"
elif [[ -x "/usr/bin/tmux" ]]; then
    exec /usr/bin/tmux new -A -s "$SESSION_NAME"
elif [[ -x "$HOME/.local/bin/tmux" ]]; then
    exec $HOME/.local/bin/tmux new -A -s "$SESSION_NAME"
else
    echo "Error: tmux not found in any common location"
    echo "Available paths:"
    echo "  PATH: $PATH"
    echo "Tried:"
    echo "  - tmux (in PATH)"
    echo "  - /bin/tmux"
    echo "  - /usr/bin/tmux"
    echo "  - $HOME/.local/bin/tmux"
    exit 1
fi
`

	if err := os.WriteFile(c.SetSizeScript, []byte(content), 0755); err != nil {
		return err
	}

	return nil
}

// createProfileScript creates the profile.sh script
func (c *Config) createProfileScript() error {
	profileScript := filepath.Join(c.ConfigDir, "profile.sh")
	content := `#!/bin/bash
# jmux profile script - sourced in tmux sessions

# Ensure ~/.local/bin is in PATH
if [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
    export PATH="$HOME/.local/bin:$PATH"
fi

# Ensure jmux is available if it exists in ~/.local/bin
if [[ -x "$HOME/.local/bin/jmux" && ! -x "$(command -v jmux 2>/dev/null)" ]]; then
    # Create a function to make jmux available
    jmux() {
        "$HOME/.local/bin/jmux" "$@"
    }
    export -f jmux
fi
`

	if err := os.WriteFile(profileScript, []byte(content), 0755); err != nil {
		return err
	}

	return nil
}

// Helper functions
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvOrDefaultInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvOrDefaultBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}