// Package checker runs the detection rules against one package name and
// produces a verdict: allow it, warn and allow it, or block it.
package checker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Arthur031221/installwall/internal/registry"
	"github.com/Arthur031221/installwall/internal/rules"
)

// Level is the outcome severity. Block stops the install, Warn lets it
// through with a printed reason, Allow is silent.
type Level string

const (
	Allow Level = "allow"
	Warn  Level = "warn"
	Block Level = "block"
)

// NewPackageAge is the threshold from the project pitch: a package first
// published less than this long ago is treated as suspicious on its own,
// since a supply-chain drop is typically live for hours before takedown.
const NewPackageAge = 72 * time.Hour

// Reason is one rule result. Multiple reasons can fire for one package.
type Reason struct {
	Rule    string `json:"rule"`
	Level   Level  `json:"level"`
	Message string `json:"message"`
	Source  string `json:"source,omitempty"`
}

// Verdict is the combined result for one package name.
type Verdict struct {
	Ecosystem rules.Ecosystem `json:"ecosystem"`
	Package   string          `json:"package"`
	Level     Level           `json:"level"`
	Reasons   []Reason        `json:"reasons"`
}

// Checker holds the loaded rule data and registry adapters so repeated
// checks (one per package on an install line) do not re-read embedded
// data or reconstruct HTTP clients each time.
type Checker struct {
	malicious  rules.MaliciousIndex
	popular    map[rules.Ecosystem]*rules.PopularNames
	registries map[rules.Ecosystem]registry.Registry
	Now        func() time.Time
}

func New() (*Checker, error) {
	malList, err := rules.LoadMalicious()
	if err != nil {
		return nil, err
	}
	c := &Checker{
		malicious: rules.IndexMalicious(malList),
		popular:   make(map[rules.Ecosystem]*rules.PopularNames),
		registries: map[rules.Ecosystem]registry.Registry{
			rules.NPM:      registry.NewNPM(),
			rules.PyPI:     registry.NewPyPI(),
			rules.RubyGems: registry.NewRubyGems(),
			rules.Crates:   registry.NewCrates(),
		},
		Now: time.Now,
	}
	for _, eco := range []rules.Ecosystem{rules.NPM, rules.PyPI, rules.RubyGems, rules.Crates} {
		pn, err := rules.LoadPopular(eco)
		if err != nil {
			return nil, err
		}
		c.popular[eco] = pn
	}
	return c, nil
}

// WithRegistry overrides the adapter for one ecosystem. Used by tests to
// point at a fake HTTP server instead of the real registry.
func (c *Checker) WithRegistry(eco rules.Ecosystem, reg registry.Registry) {
	c.registries[eco] = reg
}

// Check evaluates one package name for one ecosystem.
func (c *Checker) Check(ctx context.Context, eco rules.Ecosystem, name string) Verdict {
	v := Verdict{Ecosystem: eco, Package: name, Level: Allow}

	if ind, ok := c.malicious.Lookup(eco, name); ok {
		v.Reasons = append(v.Reasons, Reason{
			Rule:    "known-malicious",
			Level:   Block,
			Message: fmt.Sprintf("%q matches a published malicious-package indicator: %s", name, ind.Note),
			Source:  ind.Source,
		})
	}

	if pn, ok := c.popular[eco]; ok {
		if near := pn.Nearest(name); near != "" {
			v.Reasons = append(v.Reasons, Reason{
				Rule:    "typosquat",
				Level:   Block,
				Message: fmt.Sprintf("%q is one character away from the popular package %q, and is not itself a well known name", name, near),
			})
		}
	}

	if reg, ok := c.registries[eco]; ok {
		info, err := reg.Lookup(ctx, name)
		var unreachable *registry.UnreachableError
		switch {
		case errors.As(err, &unreachable):
			v.Reasons = append(v.Reasons, Reason{
				Rule:    "registry-unreachable",
				Level:   Warn,
				Message: fmt.Sprintf("could not reach the %s registry to check publish age, allowing (fail-open): %v", eco, unreachable.Err),
			})
		case err != nil:
			v.Reasons = append(v.Reasons, Reason{
				Rule:    "registry-error",
				Level:   Warn,
				Message: fmt.Sprintf("error checking %s registry, allowing (fail-open): %v", eco, err),
			})
		case info.Exists && info.HasCreatedAt:
			age := c.Now().Sub(info.CreatedAt)
			if age >= 0 && age < NewPackageAge {
				v.Reasons = append(v.Reasons, Reason{
					Rule:    "new-package",
					Level:   Block,
					Message: fmt.Sprintf("%q was first published %s ago, under the %s threshold this tool blocks by default", name, age.Round(time.Minute), NewPackageAge),
				})
			}
		}
	}

	for _, r := range v.Reasons {
		if severer(r.Level, v.Level) {
			v.Level = r.Level
		}
	}
	return v
}

func severer(a, b Level) bool {
	rank := map[Level]int{Allow: 0, Warn: 1, Block: 2}
	return rank[a] > rank[b]
}
