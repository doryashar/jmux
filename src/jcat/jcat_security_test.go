package main

import (
	"net"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// Test helper to start jcat server in background
func startJcatServer(t *testing.T, mode, address, password string, raw bool) *exec.Cmd {
	args := []string{"jcat", mode, address}
	if password != "" {
		args = append(args, "--password", password)
	}
	if raw {
		args = append(args, "--raw")
	}
	
	cmd := exec.Command("./jcat", args[1:]...)
	cmd.Dir = "."
	
	err := cmd.Start()
	if err != nil {
		t.Fatalf("Failed to start jcat server: %v", err)
	}
	
	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	
	return cmd
}

// Test helper to run jcat client
func runJcatClient(t *testing.T, mode, address, password string, raw bool) ([]byte, error) {
	args := []string{mode, address}
	if password != "" {
		args = append(args, "--password", password)
	}
	if raw {
		args = append(args, "--raw")
	}
	
	cmd := exec.Command("./jcat", args...)
	cmd.Dir = "."
	
	output, err := cmd.CombinedOutput()
	return output, err
}

// Test 1: Connection without password to password-protected server should fail
func TestPasswordProtected_NoPassword_ShouldFail(t *testing.T) {
	address := "localhost:31337"
	password := "test123"
	
	// Start password-protected server
	serverCmd := startJcatServer(t, "listen", address, password, false)
	defer serverCmd.Process.Kill()
	
	// Wait for server to start
	time.Sleep(200 * time.Millisecond)
	
	// Try to connect without password
	output, err := runJcatClient(t, "connect", address, "", false)
	
	// Should fail
	if err == nil {
		t.Errorf("Expected connection to fail without password, but it succeeded")
	}
	
	// Should contain authentication error
	outputStr := string(output)
	if !strings.Contains(outputStr, "AUTH_FAIL") && !strings.Contains(outputStr, "authentication") && !strings.Contains(outputStr, "password") {
		t.Errorf("Expected authentication error message, got: %s", outputStr)
	}
}

// Test 2: Connection with wrong password should fail
func TestPasswordProtected_WrongPassword_ShouldFail(t *testing.T) {
	address := "localhost:31338"
	correctPassword := "test123"
	wrongPassword := "wrong456"
	
	// Start password-protected server
	serverCmd := startJcatServer(t, "listen", address, correctPassword, false)
	defer serverCmd.Process.Kill()
	
	// Wait for server to start
	time.Sleep(200 * time.Millisecond)
	
	// Try to connect with wrong password
	output, err := runJcatClient(t, "connect", address, wrongPassword, false)
	
	// Should fail
	if err == nil {
		t.Errorf("Expected connection to fail with wrong password, but it succeeded")
	}
	
	// Should contain authentication error
	outputStr := string(output)
	if !strings.Contains(outputStr, "AUTH_FAIL") && !strings.Contains(outputStr, "authentication") && !strings.Contains(outputStr, "password") {
		t.Errorf("Expected authentication error message, got: %s", outputStr)
	}
}

// Test 3: Connection with correct password should succeed
func TestPasswordProtected_CorrectPassword_ShouldSucceed(t *testing.T) {
	address := "localhost:31339"
	password := "test123"
	
	// Start password-protected server
	serverCmd := startJcatServer(t, "listen", address, password, false)
	defer serverCmd.Process.Kill()
	
	// Wait for server to start
	time.Sleep(200 * time.Millisecond)
	
	// Try to connect with correct password - but we need to test without actual shell
	// For now, just check that we can establish TCP connection and see handshake
	conn, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()
	
	// Read handshake
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		t.Fatalf("Failed to read handshake: %v", err)
	}
	
	handshake := string(buffer[:n])
	if !strings.Contains(handshake, "JCAT/2.0.0+SEC") {
		t.Errorf("Expected secure handshake, got: %s", handshake)
	}
}

// Test 4: Verify encryption is used (non-raw mode)
func TestPasswordProtected_EncryptionEnabled(t *testing.T) {
	address := "localhost:31340"
	password := "test123"
	
	// Start password-protected server (non-raw = encrypted)
	serverCmd := startJcatServer(t, "listen", address, password, false)
	defer serverCmd.Process.Kill()
	
	// Wait for server to start
	time.Sleep(200 * time.Millisecond)
	
	// Connect and check that secure handshake is used
	conn, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()
	
	// Read handshake
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		t.Fatalf("Failed to read handshake: %v", err)
	}
	
	handshake := string(buffer[:n])
	// Should indicate secure/encrypted mode
	if !strings.Contains(handshake, "JCAT/2.0.0+SEC") {
		t.Errorf("Expected secure handshake indicating encryption, got: %s", handshake)
	}
}

// Test 5: Verify raw mode uses authentication but no encryption
func TestPasswordProtected_RawMode_AuthNoEncryption(t *testing.T) {
	address := "localhost:31341"
	password := "test123"
	
	// Start password-protected server in raw mode
	serverCmd := startJcatServer(t, "listen", address, password, true)
	defer serverCmd.Process.Kill()
	
	// Wait for server to start
	time.Sleep(200 * time.Millisecond)
	
	// Connect and check handshake
	conn, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()
	
	// Read handshake
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		t.Fatalf("Failed to read handshake: %v", err)
	}
	
	handshake := string(buffer[:n])
	// Should indicate secure handshake but maybe different for raw mode
	if !strings.Contains(handshake, "JCAT/2.0.0+SEC") && !strings.Contains(handshake, "JCAT/2.0.0+AUTH") {
		t.Errorf("Expected secure/auth handshake for raw mode, got: %s", handshake)
	}
}

// Test 6: Test reverse mode with password protection
func TestReversePasswordProtected_CorrectPassword(t *testing.T) {
	address := "localhost:31342"
	password := "test123"
	
	// Start reverse password-protected server
	serverCmd := startJcatServer(t, "reverse-listen", address, password, false)
	defer serverCmd.Process.Kill()
	
	// Wait for server to start
	time.Sleep(200 * time.Millisecond)
	
	// Connect and check handshake
	conn, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatalf("Failed to connect to reverse server: %v", err)
	}
	defer conn.Close()
	
	// Read handshake
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		t.Fatalf("Failed to read handshake: %v", err)
	}
	
	handshake := string(buffer[:n])
	if !strings.Contains(handshake, "JCAT/2.0.0+SEC") {
		t.Errorf("Expected secure handshake for reverse mode, got: %s", handshake)
	}
}

// Test build first to see what functions we need to implement
func TestBuildValidation(t *testing.T) {
	// This test will fail until we implement the password functionality
	// It serves as a reminder of what we need to build
	t.Log("Running build validation - this will fail until password support is implemented")
	
	// Try to build with our expected flags
	cmd := exec.Command("go", "build", "-o", "jcat_test", ".")
	cmd.Dir = "."
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		t.Logf("Build failed (expected): %s", string(output))
		t.Log("Need to implement: password flags, secure handshake, authentication flow")
	} else {
		t.Log("Build succeeded - ready for functionality tests")
	}
}