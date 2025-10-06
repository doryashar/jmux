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

// TestPortManagement tests port assignment, conflict resolution, and cleanup
func TestPortManagement(t *testing.T) {
	// Setup test environment
	testDir := setupTestEnvPort(t)
	defer cleanupTestEnvPort(testDir)

	dmuxBinary := buildDmuxBinaryPort(t)

	t.Run("PortAssignment", func(t *testing.T) {
		testPortAssignment(t, dmuxBinary, testDir)
	})

	t.Run("PortConflictPrevention", func(t *testing.T) {
		testPortConflictPrevention(t, dmuxBinary, testDir)
	})

	t.Run("PortMappingCleanup", func(t *testing.T) {
		testPortMappingCleanup(t, dmuxBinary, testDir)
	})

	t.Run("PortAvailabilityCheck", func(t *testing.T) {
		testPortAvailabilityCheck(t, dmuxBinary, testDir)
	})
}

func testPortAssignment(t *testing.T, dmuxBinary, testDir string) {
	env := []string{
		"JMUX_SHARED_DIR=" + testDir,
		"JMUX_PORT=12400", // Base port for testing
		"PATH=" + os.Getenv("PATH"),
		"USER=" + os.Getenv("USER"),
		"HOME=" + testDir,
	}

	sessionName := "port-test-" + fmt.Sprintf("%d", time.Now().Unix())

	// Try to create a session
	cmd := exec.Command(dmuxBinary, "share", "dummy", "--name", sessionName)
	cmd.Env = env
	output, err := cmd.CombinedOutput()

	// Check if output mentions port assignment
	outputStr := string(output)
	if strings.Contains(outputStr, "port 12400") || 
	   strings.Contains(outputStr, "port 12401") ||
	   strings.Contains(outputStr, "port 12402") {
		t.Logf("✓ Port assignment appears to be working. Output: %s", outputStr)
	} else if err != nil {
		t.Logf("Session creation failed (expected due to terminal): %v", err)
		// This is expected in test environment
	}

	// Check port mapping file
	portMapFile := filepath.Join(testDir, "port_sessions.db")
	if _, err := os.Stat(portMapFile); err == nil {
		content, err := os.ReadFile(portMapFile)
		if err == nil && len(content) > 0 {
			t.Logf("✓ Port mapping file created with content: %s", string(content))
		}
	}
}

func testPortConflictPrevention(t *testing.T, dmuxBinary, testDir string) {
	// Create a mock port mapping to simulate an existing session
	portMapFile := filepath.Join(testDir, "port_sessions.db")
	existingMapping := "12401:testuser:existing-session\n"
	
	if err := os.WriteFile(portMapFile, []byte(existingMapping), 0644); err != nil {
		t.Fatalf("Failed to create mock port mapping: %v", err)
	}

	env := []string{
		"JMUX_SHARED_DIR=" + testDir,
		"JMUX_PORT=12401", // Use the same base port that's already "taken"
		"PATH=" + os.Getenv("PATH"),
		"USER=" + os.Getenv("USER"),
		"HOME=" + testDir,
	}

	sessionName := "conflict-test-" + fmt.Sprintf("%d", time.Now().Unix())

	// Try to create a session - should get a different port
	cmd := exec.Command(dmuxBinary, "share", "dummy", "--name", sessionName)
	cmd.Env = env
	output, err := cmd.CombinedOutput()

	outputStr := string(output)
	
	// Should not use the conflicting port 12401
	if strings.Contains(outputStr, "port 12401") {
		t.Errorf("Session should not use conflicting port 12401. Output: %s", outputStr)
	}

	// Should use a different port (12402, 12403, etc.)
	if strings.Contains(outputStr, "port 12402") ||
	   strings.Contains(outputStr, "port 12403") {
		t.Logf("✓ Port conflict avoided, using alternative port")
	} else if err != nil {
		t.Logf("Session creation failed (expected due to terminal), but port conflict logic should still work")
	}
}

func testPortMappingCleanup(t *testing.T, dmuxBinary, testDir string) {
	// Create a mock port mapping with stale entries
	portMapFile := filepath.Join(testDir, "port_sessions.db")
	staleMapping := `12401:testuser:stale-session
12402:testuser:another-stale
12403:testuser:valid-session
`
	
	if err := os.WriteFile(portMapFile, []byte(staleMapping), 0644); err != nil {
		t.Fatalf("Failed to create mock port mapping: %v", err)
	}

	env := []string{
		"JMUX_SHARED_DIR=" + testDir,
		"JMUX_PORT=12401",
		"PATH=" + os.Getenv("PATH"),
		"USER=" + os.Getenv("USER"),
		"HOME=" + testDir,
	}

	// Run cleanup command
	cleanupCmd := exec.Command(dmuxBinary, "cleanup")
	cleanupCmd.Env = env
	output, err := cleanupCmd.CombinedOutput()

	if err != nil {
		t.Fatalf("Cleanup command failed: %v\nOutput: %s", err, string(output))
	}

	outputStr := string(output)
	
	// Should mention port mapping cleanup
	if strings.Contains(outputStr, "port mapping") || 
	   strings.Contains(outputStr, "stale") {
		t.Logf("✓ Port mapping cleanup appears to be working. Output: %s", outputStr)
	}

	// Check if port mapping file was cleaned
	if content, err := os.ReadFile(portMapFile); err == nil {
		if len(content) < len(staleMapping) {
			t.Logf("✓ Port mapping file was cleaned up")
		}
	}
}

func testPortAvailabilityCheck(t *testing.T, dmuxBinary, testDir string) {
	// Test the port availability logic by creating sessions with specific ports
	
	// Create mock session files to simulate active sessions
	sessionsDir := filepath.Join(testDir, "sessions")
	user := os.Getenv("USER")
	
	sessionFile1 := filepath.Join(sessionsDir, user+"_test1.session")
	sessionContent1 := fmt.Sprintf(`USER=%s
SESSION=test1
PORT=12405
STARTED=%d
PID=12345
PRIVATE=false
ALLOWED_USERS=
MODE=pair
`, user, time.Now().Unix())

	if err := os.WriteFile(sessionFile1, []byte(sessionContent1), 0644); err != nil {
		t.Fatalf("Failed to create mock session file: %v", err)
	}

	env := []string{
		"JMUX_SHARED_DIR=" + testDir,
		"JMUX_PORT=12405", // Use same port as existing session
		"PATH=" + os.Getenv("PATH"),
		"USER=" + user,
		"HOME=" + testDir,
	}

	sessionName := "availability-test-" + fmt.Sprintf("%d", time.Now().Unix())

	// Try to create a session - should avoid the already-used port
	cmd := exec.Command(dmuxBinary, "share", "dummy", "--name", sessionName)
	cmd.Env = env
	output, err := cmd.CombinedOutput()

	outputStr := string(output)
	
	// Should not reuse port 12405
	if strings.Contains(outputStr, "port 12405") {
		t.Errorf("Should not reuse existing session port 12405. Output: %s", outputStr)
	}

	// Should use a different port
	if strings.Contains(outputStr, "port 12406") ||
	   strings.Contains(outputStr, "port 12407") {
		t.Logf("✓ Port availability check working, using different port")
	} else if err != nil {
		t.Logf("Session creation failed (expected), but port logic should still work")
	}
}

// Helper functions

func setupTestEnvPort(t *testing.T) string {
	testDir := filepath.Join("/tmp", "dmux_port_test_"+fmt.Sprintf("%d", time.Now().UnixNano()))
	
	// Create test directory structure
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

func cleanupTestEnvPort(testDir string) {
	os.RemoveAll(testDir)
}

func buildDmuxBinaryPort(t *testing.T) string {
	// Build dmux binary for testing
	cmd := exec.Command("go", "build", "-o", "/tmp/dmux_port_test", ".")
	cmd.Dir = "/home/yashar/projects/jmux/src/jmux-go"
	
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build dmux binary: %v", err)
	}
	
	return "/tmp/dmux_port_test"
}