package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"jmux/internal/version"
)

const (
	githubAPI        = "https://api.github.com/repos/doryashar/jmux/releases/latest"
	githubRelease    = "https://github.com/doryashar/jmux/releases/download"
	timeout          = 30 * time.Second
	updateCheckFile  = "last_update_check"
	checkInterval    = 7 * 24 * time.Hour // 1 week
)

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// CheckAndUpdate checks for updates and updates if available
func CheckAndUpdate(force bool) error {
	// Skip update for dev versions unless forced
	if version.IsDevVersion() && !force {
		color.Yellow("⚠️  Development version detected. Use --force to update anyway.")
		return nil
	}

	color.Blue("🔍 Checking for updates...")
	
	// Get latest release info
	release, err := getLatestRelease()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %v", err)
	}

	currentVersion := version.GetVersion()
	latestVersion := release.TagName

	color.Cyan("Current version: %s", currentVersion)
	color.Cyan("Latest version:  %s", latestVersion)

	// Check if update is needed
	if !force && currentVersion == latestVersion {
		color.Green("✅ Already up to date!")
		return nil
	}

	if !force && isNewerVersion(currentVersion, latestVersion) {
		color.Green("✅ Current version is newer than latest release")
		return nil
	}

	// Find the appropriate binary
	binaryName := getBinaryName()
	downloadURL := ""
	
	for _, asset := range release.Assets {
		if asset.Name == binaryName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("no compatible binary found for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	color.Yellow("📥 Downloading %s...", latestVersion)
	
	// Download the new binary
	if err := downloadAndReplace(downloadURL); err != nil {
		return fmt.Errorf("failed to download update: %v", err)
	}

	color.Green("✅ Successfully updated to %s!", latestVersion)
	color.Blue("💡 Run 'dmux version' to verify the update")
	
	return nil
}

// getLatestRelease fetches the latest release info from GitHub
func getLatestRelease() (*GitHubRelease, error) {
	client := &http.Client{Timeout: timeout}
	
	resp, err := client.Get(githubAPI)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

// getBinaryName returns the expected binary name for this platform
func getBinaryName() string {
	// For now, assume Linux x64 static binary
	return "dmux"
}

// downloadAndReplace downloads the new binary and replaces the current one
func downloadAndReplace(url string) error {
	// Get current executable path
	currentPath, err := os.Executable()
	if err != nil {
		return err
	}

	// Create temporary file
	tempPath := currentPath + ".tmp"
	
	// Download to temporary file
	if err := downloadFile(url, tempPath); err != nil {
		os.Remove(tempPath)
		return err
	}

	// Make it executable
	if err := os.Chmod(tempPath, 0755); err != nil {
		os.Remove(tempPath)
		return err
	}

	// Replace the current binary
	if err := os.Rename(tempPath, currentPath); err != nil {
		os.Remove(tempPath)
		return err
	}

	return nil
}

// downloadFile downloads a file from URL to the specified path
func downloadFile(url, filepath string) error {
	client := &http.Client{Timeout: timeout}
	
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}

// isNewerVersion compares version strings (basic implementation)
func isNewerVersion(current, latest string) bool {
	// Remove 'v' prefix if present
	current = strings.TrimPrefix(current, "v")
	latest = strings.TrimPrefix(latest, "v")
	
	// Remove '-dev' suffix from current version
	current = strings.TrimSuffix(current, "-dev")
	
	// Simple string comparison for now - could be enhanced with proper semver
	return current > latest
}

// CheckForUpdatesIfNeeded checks for updates if enough time has passed
func CheckForUpdatesIfNeeded(configDir string) {
	// Skip for dev versions unless in debug mode
	if version.IsDevVersion() && os.Getenv("JMUX_DEBUG_UPDATES") == "" {
		return
	}

	// Check if it's time to check for updates
	if !shouldCheckForUpdates(configDir) {
		return
	}

	// Update the last check time
	updateLastCheckTime(configDir)

	// Check for updates in background to avoid blocking startup
	go func() {
		checkAndPromptUpdate()
	}()
}

// shouldCheckForUpdates returns true if it's time to check for updates
func shouldCheckForUpdates(configDir string) bool {
	checkFile := filepath.Join(configDir, updateCheckFile)
	
	// If file doesn't exist, it's time to check
	data, err := os.ReadFile(checkFile)
	if err != nil {
		return true
	}

	// Parse the last check time
	lastCheckStr := strings.TrimSpace(string(data))
	lastCheck, err := strconv.ParseInt(lastCheckStr, 10, 64)
	if err != nil {
		return true
	}

	lastCheckTime := time.Unix(lastCheck, 0)
	return time.Since(lastCheckTime) >= checkInterval
}

// updateLastCheckTime records the current time as the last check time
func updateLastCheckTime(configDir string) {
	checkFile := filepath.Join(configDir, updateCheckFile)
	currentTime := strconv.FormatInt(time.Now().Unix(), 10)
	os.WriteFile(checkFile, []byte(currentTime), 0644)
}

// checkAndPromptUpdate checks for updates and prompts user
func checkAndPromptUpdate() {
	// Quick check for latest release
	release, err := getLatestRelease()
	if err != nil {
		// Silently fail - don't bother user with network errors
		return
	}

	currentVersion := version.GetVersion()
	latestVersion := release.TagName

	// Check if update is available
	if currentVersion == latestVersion {
		return // Already up to date
	}

	if isNewerVersion(currentVersion, latestVersion) {
		return // Current version is newer
	}

	// Update is available - prompt user
	color.Yellow("\n🔔 Update Available!")
	color.Cyan("  Current version: %s", currentVersion)
	color.Cyan("  Latest version:  %s", latestVersion)
	color.Blue("  Run 'dmux update' to upgrade")
	
	// Ask if user wants to update now
	fmt.Print("\n" + color.YellowString("Would you like to update now? (y/N): "))
	
	var response string
	fmt.Scanln(&response)
	
	if strings.ToLower(strings.TrimSpace(response)) == "y" {
		color.Blue("\n📥 Updating...")
		err := CheckAndUpdate(false)
		if err != nil {
			color.Red("❌ Update failed: %v", err)
			color.Blue("💡 You can try again later with 'dmux update'")
		}
	} else {
		color.Blue("💡 You can update anytime with 'dmux update'")
	}
	fmt.Println()
}