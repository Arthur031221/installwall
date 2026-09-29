// Package audit writes and reads the append-only JSONL log of every
// install check installwall has run.
package audit

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Entry is one line of the audit log.
type Entry struct {
	Time      time.Time `json:"time"`
	Tool      string    `json:"tool"`
	Ecosystem string    `json:"ecosystem"`
	Package   string    `json:"package"`
	Level     string    `json:"level"`
	Reasons   []string  `json:"reasons,omitempty"`
	Forced    bool      `json:"forced,omitempty"`
}

// Home returns the installwall state directory. INSTALLWALL_HOME overrides
// it, which is how tests and CI keep this off the real home directory.
func Home() (string, error) {
	if v := os.Getenv("INSTALLWALL_HOME"); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".installwall"), nil
}

func LogPath() (string, error) {
	dir, err := Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "audit.jsonl"), nil
}

// Append writes one entry to the log, creating the state directory and
// file if needed. It never fails the caller's install: write errors are
// returned so the caller can decide, but installwall only logs them.
func Append(e Entry) error {
	path, err := LogPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	line, err := json.Marshal(e)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(f, string(line))
	return err
}

// ReadAll returns every entry in the log, oldest first. A missing log
// file is not an error, it just means nothing has run yet.
func ReadAll() ([]Entry, error) {
	path, err := LogPath()
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var e Entry
		if err := json.Unmarshal(line, &e); err != nil {
			continue // skip a corrupt line rather than fail the whole read
		}
		entries = append(entries, e)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}
