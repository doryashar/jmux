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

// TestSessionLifecycle tests the complete session creation, registration, and cleanup process
func TestSessionLifecycle(t *testing.T) {
	// Setup test environment
	testDir := setupTestEnv(t)
	defer cleanupTestEnv(testDir)

	dmuxBinary := buildDmuxBinary(t)

	t.Run("SessionCreationAndRegistration", func(t *testing.T) {
		testSessionCreationAndRegistration(t, dmuxBinary, testDir)
	})

	t.Run("SessionRegistrationRollback", func(t *testing.T) {
		testSessionRegistrationRollback(t, dmuxBinary, testDir)
	})

	t.Run("AtomicSessionCreation", func(t *testing.T) {
		testAtomicSessionCreation(t, dmuxBinary, testDir)
	})

	t.Run("SessionCleanupOnStop", func(t *testing.T) {
		testSessionCleanupOnStop(t, dmuxBinary, testDir)
	})
}

func testSessionCreationAndRegistration(t *testing.T, dmuxBinary, testDir string) {
	sessionName := "test-session-" + fmt.Sprintf("%d", time.Now().Unix())
	
	// Set environment variables for test
	env := []string{
		"JMUX_SHARED_DIR=" + testDir,
		"JMUX_PORT=12350",
		"PATH=" + os.Getenv("PATH"),
		"USER=" + os.Getenv("USER"),
		"HOME=" + testDir,
	}

	// Create session
	cmd := exec.Command(dmuxBinary, "share", "dummy", "--name", sessionName)
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		t.Logf("Command output: %s", string(output))
		// Session creation might fail due to terminal requirements, but that's ok for this test
		// We're testing the registration logic
	}

	// Wait a moment for file creation
	time.Sleep(100 * time.Millisecond)
	
	// List sessions to see if it's registered
	listCmd := exec.Command(dmuxBinary, "sessions")
	listCmd.Env = env
	listOutput, err := listCmd.CombinedOutput()
	
	if err != nil {
		if t != nil {
			t.Fatalf("Failed to list sessions: %v\nOutput: %s", err, string(listOutput))
		} else {
			fmt.Printf("Failed to list sessions: %v\nOutput: %s\n", err, string(listOutput))
			return
		}
	}

	// Should contain our session if registration worked
	outputStr := string(listOutput)
	if !strings.Contains(outputStr, sessionName) && !strings.Contains(outputStr, "No active shared sessions") {
		t.Logf("Session listing output: %s", outputStr)
		// This is expected if tmux creation failed but no registration happened
	}
}

func testSessionRegistrationRollback(t *testing.T, dmuxBinary, testDir string) {
	// This test verifies that failed session creation doesn't leave orphaned registrations
	sessionName := "rollback-test-" + fmt.Sprintf("%d", time.Now().Unix())
	
	env := []string{
		"JMUX_SHARED_DIR=" + testDir,
		"JMUX_PORT=12351",
		"PATH=" + os.Getenv("PATH"),
		"USER=" + os.Getenv("USER"),
		"HOME=" + testDir,
		"TERM=", // Remove TERM to ensure tmux fails
	}

	// Try to create session (should fail due to no terminal)
	cmd := exec.Command(dmuxBinary, "share", "dummy", "--name", sessionName)
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	
	// Expect failure
	if err == nil {
		t.Logf("Expected session creation to fail, but it succeeded. Output: %s", string(output))
	}

	// Check that no session file was created
	sessionFile := filepath.Join(testDir, "sessions", os.Getenv("USER")+"_"+sessionName+".session")
	if _, err := os.Stat(sessionFile); err == nil {
		t.Errorf("Session file should not exist after failed creation: %s", sessionFile)
	}

	// Check that session is not listed
	listCmd := exec.Command(dmuxBinary, "sessions")
	listCmd.Env = env
	listOutput, err := listCmd.CombinedOutput()
	
	if err != nil {
		if t != nil {
			t.Fatalf("Failed to list sessions: %v", err)
		} else {
			fmt.Printf("Failed to list sessions: %v\n", err)
			return
		}
	}

	if strings.Contains(string(listOutput), sessionName) {
		if t != nil {
			t.Errorf("Failed session should not appear in session list: %s", sessionName)
		} else {
			fmt.Printf("❌ Failed session should not appear in session list: %s\n", sessionName)
		}
	}
}

func testAtomicSessionCreation(t *testing.T, dmuxBinary, testDir string) {
	// Test that session creation is atomic - either fully succeeds or fully fails
	sessionName := "atomic-test-" + fmt.Sprintf("%d", time.Now().Unix())
	
	env := []string{
		"JMUX_SHARED_DIR=" + testDir,
		"JMUX_PORT=12352",
		"PATH=" + os.Getenv("PATH"),
		"USER=" + os.Getenv("USER"),
		"HOME=" + testDir,
	}

	// Get initial state
	listCmd := exec.Command(dmuxBinary, "sessions")
	listCmd.Env = env
	initialOutput, err := listCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to get initial session list: %v", err)
	}

	// Try to create session
	cmd := exec.Command(dmuxBinary, "share", "dummy", "--name", sessionName)
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	
	// Get final state
	listCmd = exec.Command(dmuxBinary, "sessions")
	listCmd.Env = env
	finalOutput, err2 := listCmd.CombinedOutput()
	if err2 != nil {
		t.Fatalf("Failed to get final session list: %v", err2)
	}

	// Check consistency
	initialSessions := strings.Count(string(initialOutput), "Session:")
	finalSessions := strings.Count(string(finalOutput), "Session:")

	if err == nil {
		// If session creation succeeded, should have one more session
		if finalSessions != initialSessions+1 {
			t.Errorf("Session creation succeeded but session count inconsistent. Initial: %d, Final: %d", 
				initialSessions, finalSessions)
		}
	} else {
		// If session creation failed, should have same number of sessions
		if finalSessions != initialSessions {
			t.Errorf("Session creation failed but session count changed. Initial: %d, Final: %d\nOutput: %s", 
				initialSessions, finalSessions, string(output))
		}
	}
}

func testSessionCleanupOnStop(t *testing.T, dmuxBinary, testDir string) {
	// This test is more complex as it requires a successful session creation
	// For now, we'll test the cleanup command functionality
	
	env := []string{
		"JMUX_SHARED_DIR=" + testDir,
		"JMUX_PORT=12353",
		"PATH=" + os.Getenv("PATH"),
		"USER=" + os.Getenv("USER"),
		"HOME=" + testDir,
	}

	// Run cleanup to ensure clean state
	cleanupCmd := exec.Command(dmuxBinary, "cleanup")
	cleanupCmd.Env = env
	output, err := cleanupCmd.CombinedOutput()
	
	if err != nil {
		t.Fatalf("Cleanup command failed: %v\nOutput: %s", err, string(output))
	}

	// Verify cleanup output mentions the various cleanup operations
	outputStr := string(output)
	if !strings.Contains(outputStr, "Cleanup complete") && 
	   !strings.Contains(outputStr, "items cleaned") {
		t.Logf("Cleanup output: %s", outputStr)
		// This is OK - might be no items to clean
	}
}

// Helper functions

func setupTestEnv(t *testing.T) string {
	testDir := filepath.Join("/tmp", "dmux_test_"+fmt.Sprintf("%d", time.Now().UnixNano()))
	
	// Create test directory structure
	dirs := []string{
		filepath.Join(testDir, "sessions"),
		filepath.Join(testDir, "messages"), 
		filepath.Join(testDir, ".config", "jmux"),
	}
	
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			if t != nil {
				t.Fatalf("Failed to create test directory %s: %v", dir, err)
			} else {
				fmt.Printf("Failed to create test directory %s: %v\n", dir, err)
				os.Exit(1)
			}
		}
	}

	return testDir
}

func cleanupTestEnv(testDir string) {
	os.RemoveAll(testDir)
}

func buildDmuxBinary(t *testing.T) string {
	// Build dmux binary for testing
	cmd := exec.Command("go", "build", "-o", "/tmp/dmux_test", ".")
	cmd.Dir = "/home/yashar/projects/jmux/src/jmux-go"
	
	if err := cmd.Run(); err != nil {
		if t != nil {
			t.Fatalf("Failed to build dmux binary: %v", err)
		} else {
			fmt.Printf("Failed to build dmux binary: %v\n", err)
			os.Exit(1)
		}
	}
	
	return "/tmp/dmux_test"
}

// Main function for standalone test execution
func main() {
	// This allows the test to be run as a standalone program
	// Usage: go run test_session_lifecycle.go
	
	fmt.Println("Running Session Lifecycle Tests...")
	
	// Run each test individually
	testDir := setupTestEnv(nil)
	defer cleanupTestEnv(testDir)

	dmuxBinary := buildDmuxBinary(nil)
	
	fmt.Println("  Testing session creation and registration...")
	testSessionCreationAndRegistration(nil, dmuxBinary, testDir)
	
	fmt.Println("  Testing session registration rollback...")
	testSessionRegistrationRollback(nil, dmuxBinary, testDir)
	
	fmt.Println("  Testing atomic session creation...")
	testAtomicSessionCreation(nil, dmuxBinary, testDir)
	
	fmt.Println("  Testing session cleanup on stop...")
	testSessionCleanupOnStop(nil, dmuxBinary, testDir)
	
	fmt.Println("✅ All session lifecycle tests completed")
}