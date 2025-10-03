package version

import (
	"fmt"
	"runtime"
)

// Version information - will be set at build time
var (
	Version   = "v1.0.0"
	GitCommit = "ec08522ac469acae162aca2263fe015a7491a7ec"
	BuildTime = "2025-10-03 13:03:14 UTC"
	GoVersion = runtime.Version()
)

// GetVersion returns the current version
func GetVersion() string {
	return Version
}

// GetFullVersion returns detailed version information
func GetFullVersion() string {
	return fmt.Sprintf("dmux %s\\nGit commit: %s\\nBuild time: %s\\nGo version: %s",
		Version, GitCommit, BuildTime, GoVersion)
}

// IsDevVersion returns true if this is a development version
func IsDevVersion() bool {
	return len(Version) > 4 && Version[len(Version)-4:] == "-dev"
}
