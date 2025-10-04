package version

import (
	"fmt"
	"runtime"
)

// Version information - will be set at build time
var (
	Version   = "v1.1.3"
	GitCommit = "e90e0097f8e3ff91af7e0c64496cdfe1d7f7d59f"
	BuildTime = "2025-10-04 18:29:27 UTC"
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
