package shim

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Tools is the fixed set of commands installwall shims.
var Tools = []string{"npm", "pip", "pip3", "gem", "cargo"}

// scriptTemplate is a POSIX sh shim. It calls back into installwall's own
// "exec" subcommand, passing the shimmed tool name so one binary can shim
// every tool. $0 is not used for the tool name because some shells resolve
// it to the shim's absolute path rather than its basename.
const scriptTemplate = `#!/bin/sh
# Installed by installwall. Do not edit by hand, run "installwall uninstall"
# to remove it and "installwall install" to regenerate it.
exec %q exec %s -- "$@"
`

// BinDir returns the directory installwall writes shim scripts into.
func BinDir() (string, error) {
	home, err := homeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "bin"), nil
}

func homeDir() (string, error) {
	if v := os.Getenv("INSTALLWALL_HOME"); v != "" {
		return v, nil
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, ".installwall"), nil
}

// Install writes a shim script for every tool in Tools, pointing at the
// given installwall binary path. It returns the shim directory.
func Install(installwallPath string) (string, error) {
	dir, err := BinDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	for _, tool := range Tools {
		content := fmt.Sprintf(scriptTemplate, installwallPath, tool)
		path := filepath.Join(dir, tool)
		if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
			return "", fmt.Errorf("writing shim for %s: %w", tool, err)
		}
	}
	return dir, nil
}

// Uninstall removes every shim script installwall may have written.
func Uninstall() error {
	dir, err := BinDir()
	if err != nil {
		return err
	}
	for _, tool := range Tools {
		path := filepath.Join(dir, tool)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing shim for %s: %w", tool, err)
		}
	}
	return nil
}

// FindReal locates the real (non-shim) binary for a tool by searching
// PATH with the shim directory removed. This is what a shim script's
// "installwall exec <tool>" call resolves to once a check passes.
func FindReal(tool string) (string, error) {
	shimDir, err := BinDir()
	if err != nil {
		return "", err
	}
	pathEnv := os.Getenv("PATH")
	sep := string(os.PathListSeparator)
	var kept []string
	for _, dir := range filepath.SplitList(pathEnv) {
		if dir == shimDir {
			continue
		}
		kept = append(kept, dir)
	}
	// exec.LookPath reads PATH from the environment, so swap it, look up,
	// then restore it, rather than reimplementing PATH search.
	old := os.Getenv("PATH")
	defer os.Setenv("PATH", old)
	os.Setenv("PATH", joinPath(kept, sep))
	return exec.LookPath(tool)
}

func joinPath(dirs []string, sep string) string {
	out := ""
	for i, d := range dirs {
		if i > 0 {
			out += sep
		}
		out += d
	}
	return out
}
