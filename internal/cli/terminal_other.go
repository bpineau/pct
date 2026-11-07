//go:build !darwin && !linux

package cli

import "os"

// terminalFile conservatively uses non-interactive mode on other platforms.
func terminalFile(_ *os.File) bool { return false }
