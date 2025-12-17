//go:build !windows
// +build !windows

package main

import (
	"os"
)

// checkHighIntegrity determines if the current process is running with high integrity (e.g., root on Linux/macOS).
func checkHighIntegrity() bool {
	return os.Geteuid() == 0 // Check if running as root
}
