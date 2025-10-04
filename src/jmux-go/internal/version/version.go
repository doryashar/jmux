package version

import (
	"fmt"
	"runtime"
)

// Version information - will be set at build time
var (
	Version   = "v1.1.5"
	GitCommit = "cc7087b739188480ea223ebf711aaa4b1a395a4a"
	BuildTime = "2025-10-04 18:48:20 UTC"
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
