//go:build !windows

package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"

	"github.com/Arthur031221/installwall/internal/shim"
)

// execReal replaces the current process with the real tool, which
// preserves its exit code and stdio exactly, and matches what running
// the tool directly (without installwall on PATH) would do.
func execReal(tool string, toolArgs []string, stdout, stderr io.Writer) int {
	realPath, err := shim.FindReal(tool)
	if err != nil {
		fmt.Fprintf(stderr, "installwall: could not find the real %s on PATH: %v\n", tool, err)
		return 127
	}
	full := append([]string{tool}, toolArgs...)
	if err := syscall.Exec(realPath, full, os.Environ()); err != nil {
		// syscall.Exec only returns on error. Fall back to a child
		// process rather than leave the shim silently doing nothing.
		cmd := exec.Command(realPath, toolArgs...)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, stdout, stderr
		if runErr := cmd.Run(); runErr != nil {
			if exitErr, ok := runErr.(*exec.ExitError); ok {
				return exitErr.ExitCode()
			}
			fmt.Fprintf(stderr, "installwall: %v\n", runErr)
			return 1
		}
	}
	return 0
}
