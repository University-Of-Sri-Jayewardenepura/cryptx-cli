//go:build !windows

package upgrade

import (
	"os"
	"syscall"
)

// execReplace replaces the current process image with the binary at path,
// using the POSIX exec(2) syscall so the PID is retained and the process
// transition is atomic from the OS's point of view.
func execReplace(path string, args []string) error {
	return syscall.Exec(path, args, os.Environ())
}
