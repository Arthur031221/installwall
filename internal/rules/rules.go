package rules

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed data/*.txt data/malicious.json
var dataFS embed.FS

// Ecosystem identifies a package registry.
type Ecosystem string

const (
	NPM      Ecosystem = "npm"
	PyPI     Ecosystem = "pypi"
	RubyGems Ecosystem = "rubygems"
	Crates   Ecosystem = "crates"
)

var files = map[Ecosystem]string{
	NPM:      "data/npm-top.txt",
	PyPI:     "data/pypi-top.txt",
	RubyGems: "data/rubygems-top.txt",
	Crates:   "data/crates-top.txt",
}

// Indicator is one published, named example of a malicious or removed
// package. Every entry must cite the advisory or write-up it came from.
// Nothing in this list is guessed: see data/malicious.json.
type Indicator struct {
	Ecosystem string `json:"ecosystem"`
	Name      string `json:"name"`
	Source    string `json:"source"`
	Note      string `json:"note"`
}

// PopularNames is the list of well known package names for one registry,
// used as the typosquat comparison set. Names lowercased, order preserved
// (most downloaded first, per the source data).
type PopularNames struct {
	Ecosystem Ecosystem
	Ordered   []string
	set       map[string]bool
}

func (p *PopularNames) Has(name string) bool {
	return p.set[strings.ToLower(name)]
}

// Nearest returns the first popular name within edit distance 1 of name,
// excluding an exact match. Empty string if none found.
func (p *PopularNames) Nearest(name string) string {
	name = strings.ToLower(name)
	if p.set[name] {
		return ""
	}
	for _, candidate := range p.Ordered {
		if candidate == name {
			continue
		}
		if EditDistance(name, candidate, 1) <= 1 {
			return candidate
		}
	}
	return ""
}

// LoadPopular reads the embedded top-name list for one ecosystem.
func LoadPopular(eco Ecosystem) (*PopularNames, error) {
	path, ok := files[eco]
	if !ok {
		return nil, fmt.Errorf("rules: unknown ecosystem %q", eco)
	}
	raw, err := dataFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("rules: reading %s: %w", path, err)
	}
	lines := strings.Split(string(raw), "\n")
	pn := &PopularNames{Ecosystem: eco, set: make(map[string]bool, len(lines))}
	for _, line := range lines {
		name := strings.ToLower(strings.TrimSpace(line))
		if name == "" || strings.HasPrefix(name, "#") {
			continue
		}
		if pn.set[name] {
			continue
		}
		pn.set[name] = true
		pn.Ordered = append(pn.Ordered, name)
	}
	return pn, nil
}

// LoadMalicious reads the embedded known-malicious indicator list.
func LoadMalicious() ([]Indicator, error) {
	raw, err := dataFS.ReadFile("data/malicious.json")
	if err != nil {
		return nil, fmt.Errorf("rules: reading malicious.json: %w", err)
	}
	var list []Indicator
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, fmt.Errorf("rules: parsing malicious.json: %w", err)
	}
	return list, nil
}

// MaliciousIndex indexes indicators by ecosystem and lowercase name for
// O(1) exact-match lookups.
type MaliciousIndex map[Ecosystem]map[string]Indicator

func IndexMalicious(list []Indicator) MaliciousIndex {
	idx := make(MaliciousIndex)
	for _, ind := range list {
		eco := Ecosystem(ind.Ecosystem)
		if idx[eco] == nil {
			idx[eco] = make(map[string]Indicator)
		}
		idx[eco][strings.ToLower(ind.Name)] = ind
	}
	return idx
}

func (m MaliciousIndex) Lookup(eco Ecosystem, name string) (Indicator, bool) {
	sub, ok := m[eco]
	if !ok {
		return Indicator{}, false
	}
	ind, ok := sub[strings.ToLower(name)]
	return ind, ok
}
