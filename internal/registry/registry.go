// Package registry adapts the four package registries installwall watches
// (npm, PyPI, RubyGems, crates.io) to one interface: does this package
// exist, and when was it first published.
package registry

import (
	"context"
	"time"
)

// Info is what installwall needs to know about one package name.
type Info struct {
	Exists       bool
	CreatedAt    time.Time
	HasCreatedAt bool
}

// Registry looks up a single package by name.
type Registry interface {
	Name() string
	Lookup(ctx context.Context, name string) (Info, error)
}

// UnreachableError wraps a network or transport failure so callers can
// tell "package does not exist" apart from "could not ask the registry".
// installwall fails open on this: it warns and allows rather than blocking
// an install because a registry was slow or down.
type UnreachableError struct {
	Registry string
	Err      error
}

func (e *UnreachableError) Error() string {
	return "registry " + e.Registry + " unreachable: " + e.Err.Error()
}

func (e *UnreachableError) Unwrap() error { return e.Err }
