//go:build darwin || linux

package cli

import (
	"os"
	"syscall"
	"unsafe"
)

// terminalFile tests terminal attributes rather than the device file type.
func terminalFile(f *os.File) bool {
	var attributes syscall.Termios
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), terminalReadRequest, uintptr(unsafe.Pointer(&attributes)))
	return errno == 0
}
