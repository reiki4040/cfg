package main

import (
	"fmt"
	"runtime"
)

var (
	// Version is the version of the application
	Version = "dev"
	// CommitHash is the git commit hash
	CommitHash = "unknown"
	// GoVersion is the version of Go used to build
	GoVersion = runtime.Version()
)

// VersionInfo holds all version-related information
type VersionInfo struct {
	Version    string `json:"version"`
	CommitHash string `json:"commit"`
	GoVersion  string `json:"go_version"`
}

// GetVersionInfo returns structured version information
func GetVersionInfo() VersionInfo {
	return VersionInfo{
		Version:    Version,
		CommitHash: CommitHash,
		GoVersion:  GoVersion,
	}
}

// GetVersionString returns a formatted version string
func GetVersionString() string {
	return fmt.Sprintf("cfgctl %s (%s) %s", Version, CommitHash, GoVersion)
}