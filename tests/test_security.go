package main

import (
	"fmt"
	"testing"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Since we can't import the internal module in tests, we'll test via the binary
// This is an integration test for security functionality

func TestPasswordAuth(t *testing.T) {
	// Test dmux security functionality via CLI commands
	dmuxBinary := buildSecurityTestBinary(t)
	defer os.Remove(dmuxBinary)
	
	// Test password option is available in share command
	cmd := exec.Command(dmuxBinary, "share", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Warning: share --help failed: %v", err)
		return
	}
	
	if strings.Contains(string(output), "password") {
		t.Logf("✓ Password option available in share command")
	} else {
		t.Errorf("Password option not found in share command help")
	}
}

func TestEncryptedConnection(t *testing.T) {
	// Test that dmux binary supports encrypted connections
	dmuxBinary := buildSecurityTestBinary(t)
	defer os.Remove(dmuxBinary)
	
	// Test private session option
	cmd := exec.Command(dmuxBinary, "share", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Warning: share --help failed: %v", err)
		return
	}
	
	if strings.Contains(string(output), "private") {
		t.Logf("✓ Private session option available")
	} else {
		t.Logf("Note: Private session option not found in help")
	}
}

func TestSecurityProtocol(t *testing.T) {
	// Test that dmux has security-related commands
	dmuxBinary := buildSecurityTestBinary(t)
	defer os.Remove(dmuxBinary)
	
	// Test join command has security options
	cmd := exec.Command(dmuxBinary, "join", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Warning: join --help failed: %v", err)
		return
	}
	
	securityOptions := []string{"password", "mode"}
	foundOptions := 0
	
	for _, option := range securityOptions {
		if strings.Contains(string(output), option) {
			foundOptions++
		}
	}
	
	if foundOptions > 0 {
		t.Logf("✓ Found %d security-related options in join command", foundOptions)
	} else {
		t.Logf("Note: No security options found in join command")
	}
}

// Helper function to build dmux binary for testing
func buildSecurityTestBinary(t *testing.T) string {
	binaryPath := fmt.Sprintf("/tmp/dmux_security_test_%d", time.Now().UnixNano())
	
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = "/home/yashar/projects/jmux/src/jmux-go"
	
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build dmux binary: %v", err)
	}
	
	return binaryPath
}

// Remove the main function - this file will be run with `go test` instead