package shim

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallAndUninstall(t *testing.T) {
	t.Setenv("INSTALLWALL_HOME", t.TempDir())

	dir, err := Install("/usr/local/bin/installwall")
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	for _, tool := range Tools {
		path := filepath.Join(dir, tool)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("shim for %s not written: %v", tool, err)
		}
		if info.Mode()&0o111 == 0 {
			t.Errorf("shim for %s is not executable: mode %v", tool, info.Mode())
		}
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading shim for %s: %v", tool, err)
		}
		if !contains(string(content), "exec "+tool+" --") || !contains(string(content), "installwall") {
			t.Errorf("shim for %s does not call back into installwall exec: %s", tool, content)
		}
	}

	if err := Uninstall(); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	for _, tool := range Tools {
		if _, err := os.Stat(filepath.Join(dir, tool)); !os.IsNotExist(err) {
			t.Errorf("shim for %s still exists after uninstall", tool)
		}
	}
}

func TestFindRealExcludesShimDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("INSTALLWALL_HOME", home)

	shimDir, err := Install("/usr/local/bin/installwall")
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	realDir := t.TempDir()
	realNpm := filepath.Join(realDir, "npm")
	if err := os.WriteFile(realNpm, []byte("#!/bin/sh\necho real npm\n"), 0o755); err != nil {
		t.Fatalf("writing fake real npm: %v", err)
	}

	t.Setenv("PATH", shimDir+string(os.PathListSeparator)+realDir)

	found, err := FindReal("npm")
	if err != nil {
		t.Fatalf("FindReal: %v", err)
	}
	if found != realNpm {
		t.Errorf("FindReal(npm) = %q, want %q (must skip the shim dir)", found, realNpm)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (func() bool {
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	})()
}
