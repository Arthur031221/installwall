package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func run(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := Run(args, &out, &errOut)
	return out.String(), errOut.String(), code
}

func TestVersionAndHelp(t *testing.T) {
	out, _, code := run(t, "version")
	if code != 0 || !strings.Contains(out, "installwall") {
		t.Fatalf("version: code=%d out=%q", code, out)
	}
	out, _, code = run(t, "help")
	if code != 0 || !strings.Contains(out, "Usage") {
		t.Fatalf("help: code=%d out=%q", code, out)
	}
	out, _, code = run(t)
	if code != 0 || !strings.Contains(out, "Usage") {
		t.Fatalf("no args: code=%d out=%q", code, out)
	}
}

func TestUnknownCommand(t *testing.T) {
	_, errOut, code := run(t, "bogus")
	if code != 2 || !strings.Contains(errOut, "unknown command") {
		t.Fatalf("bogus: code=%d err=%q", code, errOut)
	}
}

func TestCheckRequiresExactlyOnePackage(t *testing.T) {
	_, _, code := run(t, "check")
	if code != 2 {
		t.Fatalf("check with no args: code=%d, want 2", code)
	}
	_, _, code = run(t, "check", "a", "b")
	if code != 2 {
		t.Fatalf("check with two args: code=%d, want 2", code)
	}
}

func TestCheckAcceptsFlagsAfterPackageName(t *testing.T) {
	// Go's flag package stops parsing at the first positional argument,
	// so this ordering must be handled explicitly, see extractPositional.
	out, _, code := run(t, "check", "requests", "--ecosystem", "pypi", "--json")
	if code != 0 {
		t.Fatalf("code=%d out=%q", code, out)
	}
	if !strings.Contains(out, `"package": "requests"`) || !strings.Contains(out, `"ecosystem": "pypi"`) {
		t.Fatalf("output missing expected fields: %q", out)
	}

	_, errOut, code := run(t, "check", "a", "b", "--ecosystem", "pypi")
	if code != 2 || !strings.Contains(errOut, "unexpected extra argument") {
		t.Fatalf("two positionals: code=%d err=%q", code, errOut)
	}
}

func TestExecRequiresTool(t *testing.T) {
	_, _, code := run(t, "exec")
	if code != 2 {
		t.Fatalf("exec with no args: code=%d, want 2", code)
	}
}

func TestInstallUninstallCycle(t *testing.T) {
	home := t.TempDir()
	t.Setenv("INSTALLWALL_HOME", home)

	out, errOut, code := run(t, "install")
	if code != 0 {
		t.Fatalf("install: code=%d out=%q err=%q", code, out, errOut)
	}
	if _, err := os.Stat(filepath.Join(home, "bin", "npm")); err != nil {
		t.Fatalf("expected npm shim to exist: %v", err)
	}
	if !strings.Contains(out, "export PATH") {
		t.Errorf("install output should suggest a PATH line, got %q", out)
	}

	out, _, code = run(t, "uninstall")
	if code != 0 {
		t.Fatalf("uninstall: code=%d out=%q", code, out)
	}
	if _, err := os.Stat(filepath.Join(home, "bin", "npm")); !os.IsNotExist(err) {
		t.Fatalf("expected npm shim to be removed")
	}
}

func TestAuditEmptyLog(t *testing.T) {
	t.Setenv("INSTALLWALL_HOME", t.TempDir())
	out, _, code := run(t, "audit")
	if code != 0 || !strings.Contains(out, "No installs logged") {
		t.Fatalf("audit empty: code=%d out=%q", code, out)
	}
}
