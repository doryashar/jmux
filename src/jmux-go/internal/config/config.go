package config

import (
	"os"
	"path/filepath"
	"strconv"
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
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
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