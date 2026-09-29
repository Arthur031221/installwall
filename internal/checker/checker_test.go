package checker

import (
	"context"
	"testing"
	"time"

	"github.com/Arthur031221/installwall/internal/registry"
	"github.com/Arthur031221/installwall/internal/rules"
)

// fakeRegistry lets tests control exactly what a "registry" answers
// without any network access, keeping the suite fast and deterministic.
type fakeRegistry struct {
	name string
	info registry.Info
	err  error
}

func (f *fakeRegistry) Name() string { return f.name }
func (f *fakeRegistry) Lookup(ctx context.Context, name string) (registry.Info, error) {
	return f.info, f.err
}

func newTestChecker(t *testing.T) *Checker {
	t.Helper()
	c, err := New()
	if err != nil {
		t.Fatalf("checker.New: %v", err)
	}
	c.Now = func() time.Time { return time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC) }
	return c
}

func TestCheckAllowsUnremarkablePackage(t *testing.T) {
	c := newTestChecker(t)
	c.WithRegistry(rules.PyPI, &fakeRegistry{name: "pypi", info: registry.Info{
		Exists: true, HasCreatedAt: true, CreatedAt: time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC),
	}})
	v := c.Check(context.Background(), rules.PyPI, "requests")
	if v.Level != Allow {
		t.Fatalf("Level = %v, want Allow, reasons=%+v", v.Level, v.Reasons)
	}
}

func TestCheckBlocksTyposquat(t *testing.T) {
	c := newTestChecker(t)
	c.WithRegistry(rules.PyPI, &fakeRegistry{name: "pypi", info: registry.Info{Exists: false}})
	v := c.Check(context.Background(), rules.PyPI, "reqeusts")
	if v.Level != Block {
		t.Fatalf("Level = %v, want Block, reasons=%+v", v.Level, v.Reasons)
	}
	found := false
	for _, r := range v.Reasons {
		if r.Rule == "typosquat" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a typosquat reason, got %+v", v.Reasons)
	}
}

func TestCheckBlocksNewPackage(t *testing.T) {
	c := newTestChecker(t)
	c.WithRegistry(rules.NPM, &fakeRegistry{name: "npm", info: registry.Info{
		Exists: true, HasCreatedAt: true, CreatedAt: time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC), // 12h old
	}})
	// left-pad is not in the popular list fixture path, use a name that
	// will not also trip the typosquat rule by picking one far from any
	// real popular name.
	v := c.Check(context.Background(), rules.NPM, "zzz-brand-new-tool-9182")
	if v.Level != Block {
		t.Fatalf("Level = %v, want Block, reasons=%+v", v.Level, v.Reasons)
	}
	found := false
	for _, r := range v.Reasons {
		if r.Rule == "new-package" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a new-package reason, got %+v", v.Reasons)
	}
}

func TestCheckAllowsOldPackagePastThreshold(t *testing.T) {
	c := newTestChecker(t)
	c.WithRegistry(rules.NPM, &fakeRegistry{name: "npm", info: registry.Info{
		Exists: true, HasCreatedAt: true, CreatedAt: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), // weeks old
	}})
	v := c.Check(context.Background(), rules.NPM, "zzz-brand-new-tool-9182")
	if v.Level != Allow {
		t.Fatalf("Level = %v, want Allow, reasons=%+v", v.Level, v.Reasons)
	}
}

func TestCheckFailsOpenOnUnreachableRegistry(t *testing.T) {
	c := newTestChecker(t)
	c.WithRegistry(rules.Crates, &fakeRegistry{name: "crates", err: &registry.UnreachableError{Registry: "crates", Err: context.DeadlineExceeded}})
	v := c.Check(context.Background(), rules.Crates, "zzz-brand-new-tool-9182")
	if v.Level != Warn {
		t.Fatalf("Level = %v, want Warn (fail-open), reasons=%+v", v.Level, v.Reasons)
	}
	found := false
	for _, r := range v.Reasons {
		if r.Rule == "registry-unreachable" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a registry-unreachable reason, got %+v", v.Reasons)
	}
}

func TestCheckBlocksKnownMalicious(t *testing.T) {
	c := newTestChecker(t)
	c.malicious = rules.IndexMalicious([]rules.Indicator{
		{Ecosystem: "npm", Name: "evil-test-package", Source: "https://example.invalid/advisory", Note: "test fixture, not a real advisory"},
	})
	c.WithRegistry(rules.NPM, &fakeRegistry{name: "npm", info: registry.Info{Exists: true, HasCreatedAt: false}})
	v := c.Check(context.Background(), rules.NPM, "evil-test-package")
	if v.Level != Block {
		t.Fatalf("Level = %v, want Block, reasons=%+v", v.Level, v.Reasons)
	}
	if v.Reasons[0].Rule != "known-malicious" {
		t.Errorf("expected known-malicious to be the first reason, got %+v", v.Reasons)
	}
	if v.Reasons[0].Source == "" {
		t.Error("known-malicious reason must cite its source")
	}
}
