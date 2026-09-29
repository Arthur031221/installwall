package rules

import "testing"

func TestLoadPopularKnownNames(t *testing.T) {
	// requests is one of the most downloaded PyPI packages and must be in
	// any reasonable top-N list built from real download data.
	pn, err := LoadPopular(PyPI)
	if err != nil {
		t.Fatalf("LoadPopular(PyPI): %v", err)
	}
	if len(pn.Ordered) == 0 {
		t.Fatal("pypi popular list is empty")
	}
	if !pn.Has("requests") {
		t.Error("expected requests to be a known popular PyPI package")
	}
	if near := pn.Nearest("reqeusts"); near != "requests" {
		t.Errorf("Nearest(reqeusts) = %q, want requests", near)
	}
	if near := pn.Nearest("requests"); near != "" {
		t.Errorf("Nearest(requests) = %q, want empty (exact match is not a typosquat)", near)
	}
}

func TestLoadPopularUnknownEcosystem(t *testing.T) {
	if _, err := LoadPopular(Ecosystem("bogus")); err == nil {
		t.Fatal("expected an error for an unknown ecosystem")
	}
}

func TestNearestIgnoresPopularVsPopular(t *testing.T) {
	// A name that is itself well known should never be flagged as a
	// typosquat of a neighbor, even if one exists at edit distance 1.
	// This is what keeps the false-positive rate low on the top-N set
	// itself (see bench/).
	pn := &PopularNames{set: map[string]bool{"express": true, "expres": true}, Ordered: []string{"express", "expres"}}
	if near := pn.Nearest("express"); near != "" {
		t.Errorf("Nearest(express) = %q, want empty because express is itself popular", near)
	}
}

func TestLoadMalicious(t *testing.T) {
	list, err := LoadMalicious()
	if err != nil {
		t.Fatalf("LoadMalicious: %v", err)
	}
	for _, ind := range list {
		if ind.Source == "" {
			t.Errorf("indicator %q has no source citation", ind.Name)
		}
		if ind.Ecosystem == "" {
			t.Errorf("indicator %q has no ecosystem", ind.Name)
		}
	}
	for _, eco := range []Ecosystem{NPM, PyPI, RubyGems, Crates} {
		popular, err := LoadPopular(eco)
		if err != nil {
			t.Fatalf("LoadPopular(%s): %v", eco, err)
		}
		for _, ind := range list {
			if Ecosystem(ind.Ecosystem) == eco && popular.Has(ind.Name) {
				t.Errorf("popular %s package %q has a name-level malicious block; check whether only a version was affected", eco, ind.Name)
			}
		}
	}
}

func TestIndexMaliciousLookup(t *testing.T) {
	idx := IndexMalicious([]Indicator{
		{Ecosystem: "rubygems", Name: "Evil-Gem", Source: "https://example.invalid/advisory", Note: "test fixture"},
	})
	if _, ok := idx.Lookup(RubyGems, "evil-gem"); !ok {
		t.Error("expected case-insensitive lookup to find evil-gem")
	}
	if _, ok := idx.Lookup(NPM, "evil-gem"); ok {
		t.Error("lookup must be scoped to the right ecosystem")
	}
	if _, ok := idx.Lookup(RubyGems, "not-evil"); ok {
		t.Error("lookup must not match unrelated names")
	}
}
