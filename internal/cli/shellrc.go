package cli

import (
	"os"
	"path/filepath"
	"strings"
)

// appendToShellRC appends line to the user's shell rc file if it is not
// already present. Returns the rc path, whether it wrote anything, and
// any error.
func appendToShellRC(line string) (string, bool, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false, err
	}
	rc := filepath.Join(home, ".zshrc")
	if shell := os.Getenv("SHELL"); strings.Contains(shell, "bash") {
		rc = filepath.Join(home, ".bashrc")
	}

	existing, err := os.ReadFile(rc)
	if err != nil && !os.IsNotExist(err) {
		return rc, false, err
	}
	if strings.Contains(string(existing), line) {
		return rc, false, nil
	}

	f, err := os.OpenFile(rc, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return rc, false, err
	}
	defer f.Close()
	if _, err := f.WriteString("\n# added by installwall\n" + line + "\n"); err != nil {
		return rc, false, err
	}
	return rc, true, nil
}
