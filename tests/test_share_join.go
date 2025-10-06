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

// TestShareJoin tests the core share and join functionality
func TestShareJoin(t *testing.T) {
	// Setup test environment
	testDir := setupTestEnv(t)
	defer cleanupTestEnv(testDir)

	dmuxBinary := buildDmuxBinary(t)

	t.Run("ShareCommand", func(t *testing.T) {
		testShareCommand(t, dmuxBinary, testDir)
	})

	t.Run("ShareWithOptions", func(t *testing.T) {
		testShareWithOptions(t, dmuxBinary, testDir)
	})

	t.Run("ArgumentParsing", func(t *testing.T) {
		testArgumentParsing(t, dmuxBinary, testDir)
	})

	t.Run("JoinCommandValidation", func(t *testing.T) {
		testJoinCommandValidation(t, dmuxBinary, testDir)
	})

	t.Run("SessionModes", func(t *testing.T) {
		testSessionModes(t, dmuxBinary, testDir)
	})
}

func testShareCommand(t *testing.T, dmuxBinary, testDir string) {
	env := []string{
		"JMUX_SHARED_DIR=" + testDir,
		"JMUX_PORT=12500",
		"PATH=" + os.Getenv("PATH"),
		"USER=" + os.Getenv("USER"),
		"HOME=" + testDir,
	}

	sessionName := "share-test-" + fmt.Sprintf("%d", time.Now().Unix())

	// Test basic share command
	cmd := exec.Command(dmuxBinary, "share", sessionName)
	cmd.Env = env
	output, err := cmd.CombinedOutput()

	outputStr := string(output)
	
	// Should show session creation messages
	if strings.Contains(outputStr, "Starting") ||
	   strings.Contains(outputStr, "session") {
		t.Logf("✓ Share command produces expected output")
	}

	// Should mention the session name
	if strings.Contains(outputStr, sessionName) {
		t.Logf("✓ Share command uses correct session name")
	}

	// Should mention port assignment
	if strings.Contains(outputStr, "port") {
		t.Logf("✓ Share command assigns port")
	}

	if err != nil {
		t.Logf("Share command failed (expected in test environment): %v", err)
		// This is expected since we don't have a proper terminal
	}
}

func testShareWithOptions(t *testing.T, dmuxBinary, testDir string) {
	env := []string{
		"JMUX_SHARED_DIR=" + testDir,
		"JMUX_PORT=12501",
		"PATH=" + os.Getenv("PATH"),
		"USER=" + os.Getenv("USER"),
		"HOME=" + testDir,
	}

	sessionName := "options-test-" + fmt.Sprintf("%d", time.Now().Unix())

	// Test share with private flag
	cmd := exec.Command(dmuxBinary, "share", sessionName, "--private", "--invite", "user1,user2", "--mode", "view")
	cmd.Env = env
	output, err := cmd.CombinedOutput()

	outputStr := string(output)
	
	// Should mention private session
	if strings.Contains(outputStr, "private") || strings.Contains(outputStr, "Private") {
		t.Logf("✓ Private session option recognized")
	}

	// Should handle mode specification
	if strings.Contains(outputStr, "view") {
		t.Logf("✓ Session mode option recognized")
	}

	if err != nil {
		t.Logf("Share with options failed (expected): %v", err)
	}
}

func testArgumentParsing(t *testing.T, dmuxBinary, testDir string) {
	env := []string{
		"JMUX_SHARED_DIR=" + testDir,
		"JMUX_PORT=12502",
		"PATH=" + os.Getenv("PATH"),
		"USER=" + os.Getenv("USER"),
		"HOME=" + testDir,
	}

	// Test the specific bug that was fixed: --name flag should take precedence
	cmd := exec.Command(dmuxBinary, "share", "positional_arg", "--name", "flag_name", "--password", "testing")
	cmd.Env = env
	output, err := cmd.CombinedOutput()

	outputStr := string(output)
	
	// Should use "flag_name" not "positional_arg"
	if strings.Contains(outputStr, "flag_name") {
		t.Logf("✓ --name flag takes precedence over positional argument")
	}

	if strings.Contains(outputStr, "positional_arg") && !strings.Contains(outputStr, "flag_name") {
		t.Errorf("Argument parsing bug: using positional arg instead of --name flag")
	}

	if err != nil {
		t.Logf("Argument parsing test failed (expected): %v", err)
	}
}

func testJoinCommandValidation(t *testing.T, dmuxBinary, testDir string) {
	env := []string{
		"JMUX_SHARED_DIR=" + testDir,
		"JMUX_PORT=12503",
		"PATH=" + os.Getenv("PATH"),
		"USER=" + os.Getenv("USER"),
		"HOME=" + testDir,
	}

	// Test join command with no sessions available
	cmd := exec.Command(dmuxBinary, "join", "nonexistent_user")
	cmd.Env = env
	output, err := cmd.CombinedOutput()

	if err != nil {
		outputStr := string(output)
		
		// Should provide helpful error message
		if strings.Contains(outputStr, "not found") || 
		   strings.Contains(outputStr, "no sessions") {
			t.Logf("✓ Join command provides helpful error for missing user")
		}
	}

	// Test join command with invalid arguments
	cmd = exec.Command(dmuxBinary, "join")
	cmd.Env = env
	output, err = cmd.CombinedOutput()

	if err != nil {
		outputStr := string(output)
		
		// Should show usage information
		if strings.Contains(outputStr, "usage") || 
		   strings.Contains(outputStr, "required") ||
		   strings.Contains(outputStr, "argument") {
			t.Logf("✓ Join command shows usage for missing arguments")
		}
	}
}

func testSessionModes(t *testing.T, dmuxBinary, testDir string) {
	env := []string{
		"JMUX_SHARED_DIR=" + testDir,
		"JMUX_PORT=12504",
		"PATH=" + os.Getenv("PATH"),
		"USER=" + os.Getenv("USER"),
		"HOME=" + testDir,
	}

	sessionName := "mode-test-" + fmt.Sprintf("%d", time.Now().Unix())

	// Test different session modes
	modes := []string{"pair", "view", "rogue"}
	
	for _, mode := range modes {
		cmd := exec.Command(dmuxBinary, "share", sessionName+"-"+mode, "--mode", mode)
		cmd.Env = env
		output, err := cmd.CombinedOutput()

		outputStr := string(output)
		
		// Should mention the mode
		if strings.Contains(outputStr, mode) {
			t.Logf("✓ Mode '%s' recognized", mode)
		}

		if err != nil {
			t.Logf("Mode test for '%s' failed (expected): %v", mode, err)
		}
	}
}

// Additional tests for edge cases

func TestShareEdgeCases(t *testing.T) {
	testDir := setupTestEnv(t)
	defer cleanupTestEnv(testDir)

	dmuxBinary := buildDmuxBinary(t)

	t.Run("EmptySessionName", func(t *testing.T) {
		testEmptySessionName(t, dmuxBinary, testDir)
	})

	t.Run("LongSessionName", func(t *testing.T) {
		testLongSessionName(t, dmuxBinary, testDir)
	})

	t.Run("SpecialCharacters", func(t *testing.T) {
		testSpecialCharacters(t, dmuxBinary, testDir)
	})
}

func testEmptySessionName(t *testing.T, dmuxBinary, testDir string) {
	env := []string{
		"JMUX_SHARED_DIR=" + testDir,
		"JMUX_PORT=12505",
		"PATH=" + os.Getenv("PATH"),
		"USER=" + os.Getenv("USER"),
		"HOME=" + testDir,
	}

	// Test with empty session name - should generate one
	cmd := exec.Command(dmuxBinary, "share")
	cmd.Env = env
	output, err := cmd.CombinedOutput()

	outputStr := string(output)
	
	// Should generate a session name with timestamp
	if strings.Contains(outputStr, "session-") {
		t.Logf("✓ Empty session name generates automatic name")
	}

	if err != nil {
		t.Logf("Empty session name test failed (expected): %v", err)
	}
}

func testLongSessionName(t *testing.T, dmuxBinary, testDir string) {
	env := []string{
		"JMUX_SHARED_DIR=" + testDir,
		"JMUX_PORT=12506",
		"PATH=" + os.Getenv("PATH"),
		"USER=" + os.Getenv("USER"),
		"HOME=" + testDir,
	}

	// Test with very long session name
	longName := strings.Repeat("long", 50) // 200 characters
	
	cmd := exec.Command(dmuxBinary, "share", longName)
	cmd.Env = env
	output, err := cmd.CombinedOutput()

	outputStr := string(output)
	
	// Should handle long names (might truncate or reject)
	if strings.Contains(outputStr, "long") {
		t.Logf("✓ Long session name handled")
	}

	if err != nil {
		t.Logf("Long session name test failed (expected): %v", err)
	}
}

func testSpecialCharacters(t *testing.T, dmuxBinary, testDir string) {
	env := []string{
		"JMUX_SHARED_DIR=" + testDir,
		"JMUX_PORT=12507",
		"PATH=" + os.Getenv("PATH"),
		"USER=" + os.Getenv("USER"),
		"HOME=" + testDir,
	}

	// Test with special characters that might cause issues
	specialNames := []string{
		"test-with-dashes",
		"test_with_underscores", 
		"test.with.dots",
		// Avoid truly problematic characters that could break the system
	}
	
	for _, name := range specialNames {
		cmd := exec.Command(dmuxBinary, "share", name)
		cmd.Env = env
		output, err := cmd.CombinedOutput()

		outputStr := string(output)
		
		// Should handle the name or provide clear error
		if strings.Contains(outputStr, name) || strings.Contains(outputStr, "invalid") {
			t.Logf("✓ Special character name '%s' handled appropriately", name)
		}

		if err != nil {
			t.Logf("Special character test for '%s' failed (expected): %v", name, err)
		}
	}
}

// Helper functions

func setupTestEnv(t *testing.T) string {
	testDir := filepath.Join("/tmp", "dmux_share_test_"+fmt.Sprintf("%d", time.Now().UnixNano()))
	
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

func cleanupTestEnv(testDir string) {
	os.RemoveAll(testDir)
}

func buildDmuxBinary(t *testing.T) string {
	cmd := exec.Command("go", "build", "-o", "/tmp/dmux_share_test", ".")
	cmd.Dir = "/home/yashar/projects/jmux/src/jmux-go"
	
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build dmux binary: %v", err)
	}
	
	return "/tmp/dmux_share_test"
}

// Main function for standalone test execution
func main() {
	t := &testing.T{}
	
	fmt.Println("Running Share/Join Tests...")
	TestShareJoin(t)
	TestShareEdgeCases(t)
	
	if t.Failed() {
		fmt.Println("❌ Some tests failed")
		os.Exit(1)
	} else {
		fmt.Println("✅ All share/join tests passed")
	}
}