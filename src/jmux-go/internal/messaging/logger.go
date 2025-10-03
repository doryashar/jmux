package messaging

import (
	"fmt"
	"log"
	"os"
)

// Logger handles monitor logging
type Logger struct {
	file   *os.File
	logger *log.Logger
	debug  bool
}

// NewLogger creates a new logger for the monitor
func NewLogger(logFile string) (*Logger, error) {
	// Create log file
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}

	// Create logger with timestamp
	logger := log.New(file, "", log.LstdFlags)
	
	// Check if debug mode is enabled
	debug := os.Getenv("DMUX_DEBUG") != ""

	return &Logger{
		file:   file,
		logger: logger,
		debug:  debug,
	}, nil
}

// Close closes the log file
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	l.logger.Printf("[INFO] %s", msg)
	
	// Also print to stdout if debug is enabled
	if l.debug {
		fmt.Printf("[INFO] %s\n", msg)
	}
}

// Debug logs a debug message (only if DMUX_DEBUG is set)
func (l *Logger) Debug(format string, args ...interface{}) {
	if !l.debug {
		return
	}
	
	msg := fmt.Sprintf(format, args...)
	l.logger.Printf("[DEBUG] %s", msg)
	fmt.Printf("[DEBUG] %s\n", msg)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	l.logger.Printf("[ERROR] %s", msg)
	
	// Always print errors to stderr
	fmt.Fprintf(os.Stderr, "[ERROR] %s\n", msg)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	l.logger.Printf("[WARN] %s", msg)
	
	// Also print to stdout if debug is enabled
	if l.debug {
		fmt.Printf("[WARN] %s\n", msg)
	}
}

// LogMonitorStart logs monitor startup
func (l *Logger) LogMonitorStart(pid int) {
	l.Info("Messaging monitor started with PID %d", pid)
}

// LogMonitorStop logs monitor shutdown
func (l *Logger) LogMonitorStop() {
	l.Info("Messaging monitor stopping")
}

// LogMessageProcessed logs when a message is processed
func (l *Logger) LogMessageProcessed(from, msgType, data string) {
	l.Info("Processed message: from=%s type=%s data=%s", from, msgType, data)
}

// LogDisplayMethod logs which display method is being used
func (l *Logger) LogDisplayMethod(method string) {
	l.Debug("Using display method: %s", method)
}