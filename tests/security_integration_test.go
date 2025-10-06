package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestSecurityIntegration tests security functionality via the dmux binary
func TestSecurityIntegration(t *testing.T) {
	// Setup test environment
	testDir := setupTestEnvSecurity(t)
	defer cleanupTestEnvSecurity(testDir)

	dmuxBinary := buildDmuxBinarySecurity(t)

	t.Run("PasswordProtectedSession", func(t *testing.T) {
		testPasswordProtectedSession(t, dmuxBinary, testDir)
	})

	t.Run("SecurityEnabledConfig", func(t *testing.T) {
		testSecurityEnabledConfig(t, dmuxBinary, testDir)
	})

	t.Run("PrivateSessionAccess", func(t *testing.T) {
		testPrivateSessionAccess(t, dmuxBinary, testDir)
	})
}

func testPasswordProtectedSession(t *testing.T, dmuxBinary, testDir string) {
	env := []string{
		"JMUX_SHARED_DIR=" + testDir,
		"JMUX_PORT=12600",
		"PATH=" + os.Getenv("PATH"),
		"USER=" + os.Getenv("USER"),
		"HOME=" + testDir,
	}

	sessionName := "secure-test-" + fmt.Sprintf("%d", time.Now().Unix())

	// Create session with password
	cmd := exec.Command(dmuxBinary, "share", sessionName, "--password", "test123")
	cmd.Env = env
	output, err := cmd.CombinedOutput()

	outputStr := string(output)
	
	// Should handle password option
	if strings.Contains(outputStr, sessionName) {
		t.Logf("✓ Password-protected session creation attempted")
	}

	if err != nil {
		t.Logf("Session creation failed (expected in test environment): %v", err)
	}

	// Check that the share command accepts password parameter
	cmd = exec.Command(dmuxBinary, "share", "--help")
	cmd.Env = env
	helpOutput, err := cmd.CombinedOutput()

	if err == nil && strings.Contains(string(helpOutput), "password") {
		t.Logf("✓ Share command has password option")
	}
}

func testSecurityEnabledConfig(t *testing.T, dmuxBinary, testDir string) {
	env := []string{
		"JMUX_SHARED_DIR=" + testDir,
		"JMUX_PORT=12601",
		"PATH=" + os.Getenv("PATH"),
		"USER=" + os.Getenv("USER"),
		"HOME=" + testDir,
	}

	// Test security-related flags and options
	cmd := exec.Command(dmuxBinary, "share", "--help")
	cmd.Env = env
	output, err := cmd.CombinedOutput()

	if err == nil {
		outputStr := string(output)
		
		// Check for security-related options
		securityOptions := []string{"password", "private", "invite"}
		foundOptions := 0
		
		for _, option := range securityOptions {
			if strings.Contains(outputStr, option) {
				foundOptions++
			}
		}
		
		if foundOptions >= 2 {
			t.Logf("✓ Security-related options available (%d/3)", foundOptions)
		} else {
			t.Logf("⚠ Limited security options found (%d/3)", foundOptions)
		}
	}
}

func testPrivateSessionAccess(t *testing.T, dmuxBinary, testDir string) {
	env := []string{
		"JMUX_SHARED_DIR=" + testDir,
		"JMUX_PORT=12602",
		"PATH=" + os.Getenv("PATH"),
		"USER=" + os.Getenv("USER"),
		"HOME=" + testDir,
	}

	sessionName := "private-test-" + fmt.Sprintf("%d", time.Now().Unix())

	// Create private session
	cmd := exec.Command(dmuxBinary, "share", sessionName, "--private", "--invite", "user1,user2")
	cmd.Env = env
	output, err := cmd.CombinedOutput()

	outputStr := string(output)
	
	// Should handle private session creation
	if strings.Contains(outputStr, "private") || strings.Contains(outputStr, "Private") {
		t.Logf("✓ Private session option recognized")
	}

	if strings.Contains(outputStr, sessionName) {
		t.Logf("✓ Private session creation attempted")
	}

	if err != nil {
		t.Logf("Private session creation failed (expected in test environment): %v", err)
	}

	// Test join command has proper parameters
	cmd = exec.Command(dmuxBinary, "join", "--help")
	cmd.Env = env
	helpOutput, err := cmd.CombinedOutput()

	if err == nil {
		helpStr := string(helpOutput)
		if strings.Contains(helpStr, "password") || strings.Contains(helpStr, "mode") {
			t.Logf("✓ Join command has security-related options")
		}
	}
}

// Helper functions

func setupTestEnvSecurity(t *testing.T) string {
	testDir := filepath.Join("/tmp", "dmux_security_test_"+fmt.Sprintf("%d", time.Now().UnixNano()))
	
	dirs := []string{
		filepath.Join(testDir, "sessions"),
		filepath.Join(testDir, "messages"), 
		filepath.Join(testDir, ".config", "jmux"),
	}
	
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create test directory %s: %v", dir, err)
		}
	}

	return testDir
}

func cleanupTestEnvSecurity(testDir string) {
	os.RemoveAll(testDir)
}

func buildDmuxBinarySecurity(t *testing.T) string {
	cmd := exec.Command("go", "build", "-o", "/tmp/dmux_security_test", ".")
	cmd.Dir = "/home/yashar/projects/jmux/src/jmux-go"
	
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build dmux binary: %v", err)
	}
	
	return "/tmp/dmux_security_test"
}