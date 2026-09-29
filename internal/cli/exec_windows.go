//go:build windows

package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/Arthur031221/installwall/internal/shim"
)

// execReal runs the real tool as a child process. Windows has no
// exec-replace syscall usable here, so this proxies stdio and the exit
// code instead.
func execReal(tool string, toolArgs []string, stdout, stderr io.Writer) int {
	realPath, err := shim.FindReal(tool)
	if err != nil {
		fmt.Fprintf(stderr, "installwall: could not find the real %s on PATH: %v\n", tool, err)
		return 127
	}
	cmd := exec.Command(realPath, toolArgs...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, stdout, stderr
	if runErr := cmd.Run(); runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(stderr, "installwall: %v\n", runErr)
		return 1
	}
	return 0
}
