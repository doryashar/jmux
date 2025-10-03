package version

import (
	"fmt"
	"runtime"
)

// Version information - will be set at build time
var (
	Version   = "v1.1.1"
	GitCommit = "9106ca4ebd25b48f6b2a9b20a270d6942814afb1"
	BuildTime = "2025-10-03 18:40:25 UTC"
	GoVersion = runtime.Version()
)

// GetVersion returns the current version
func GetVersion() string {
	return Version
}

// GetFullVersion returns detailed version information
func GetFullVersion() string {
	return fmt.Sprintf("dmux %s\nGit commit: %s\nBuild time: %s\nGo version: %s",
		Version, GitCommit, BuildTime, GoVersion)
}

// IsDevVersion returns true if this is a development version
func IsDevVersion() bool {
	return len(Version) > 4 && Version[len(Version)-4:] == "-dev"
}
