//go:build windows

package upgrade

import (
	"os"
	"os/exec"
)

// execReplace on Windows cannot atomically replace the running process like
// POSIX exec(2), so we start the replacement binary as a child process and
// then exit, giving the child a moment to take over.
func execReplace(path string, args []string) error {
	cmd := exec.Command(path, args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	os.Exit(0)
	return nil // unreachable
}
