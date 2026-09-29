package audit

import (
	"os"
	"testing"
	"time"
)

func TestAppendAndReadAll(t *testing.T) {
	t.Setenv("INSTALLWALL_HOME", t.TempDir())

	entries, err := ReadAll()
	if err != nil {
		t.Fatalf("ReadAll on empty state: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no entries, got %d", len(entries))
	}

	want := []Entry{
		{Time: time.Now().Add(-time.Minute), Tool: "npm", Ecosystem: "npm", Package: "reqeusts", Level: "block", Reasons: []string{"typosquat: close to requests"}},
		{Time: time.Now(), Tool: "pip", Ecosystem: "pypi", Package: "requests", Level: "allow"},
	}
	for _, e := range want {
		if err := Append(e); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}

	got, err := ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("got %d entries, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Package != want[i].Package || got[i].Level != want[i].Level || got[i].Tool != want[i].Tool {
			t.Errorf("entry %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestReadAllSkipsCorruptLines(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("INSTALLWALL_HOME", dir)

	if err := Append(Entry{Tool: "npm", Package: "ok", Level: "allow"}); err != nil {
		t.Fatalf("Append: %v", err)
	}
	path, err := LogPath()
	if err != nil {
		t.Fatalf("LogPath: %v", err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := f.WriteString("not json\n"); err != nil {
		t.Fatalf("write: %v", err)
	}
	f.Close()

	entries, err := ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected corrupt line to be skipped, got %d entries", len(entries))
	}
}
