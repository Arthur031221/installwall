package shim

import (
	"reflect"
	"testing"
)

func TestParseNPM(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want Plan
	}{
		{"plain install", []string{"install", "lodash"}, Plan{true, []Spec{{Name: "lodash"}}}},
		{"pinned version", []string{"install", "lodash@4.17.21"}, Plan{true, []Spec{{Name: "lodash", Version: "4.17.21"}}}},
		{"scoped package", []string{"i", "@types/node@20.0.0"}, Plan{true, []Spec{{Name: "@types/node", Version: "20.0.0"}}}},
		{"scoped no version", []string{"install", "@babel/core"}, Plan{true, []Spec{{Name: "@babel/core"}}}},
		{"flags skipped", []string{"install", "--save-dev", "typescript"}, Plan{true, []Spec{{Name: "typescript"}}}},
		{"multiple packages", []string{"add", "react", "react-dom"}, Plan{true, []Spec{{Name: "react"}, {Name: "react-dom"}}}},
		{"workspace shorthand", []string{"install", "abbrev", "-w", "a"}, Plan{true, []Spec{{Name: "abbrev"}}}},
		{"workspace option", []string{"install", "--workspace", "packages/api", "abbrev"}, Plan{true, []Spec{{Name: "abbrev"}}}},
		{"no args, from package.json", []string{"install"}, Plan{Checked: false}},
		{"ci does not install new", []string{"ci"}, Plan{Checked: false}},
		{"run build passes through", []string{"run", "build"}, Plan{Checked: false}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Parse("npm", c.args)
			if got.Checked != c.want.Checked {
				t.Fatalf("Checked = %v, want %v", got.Checked, c.want.Checked)
			}
			if c.want.Checked && !reflect.DeepEqual(got.Packages, c.want.Packages) {
				t.Fatalf("Packages = %+v, want %+v", got.Packages, c.want.Packages)
			}
		})
	}
}

func TestParsePip(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want Plan
	}{
		{"plain", []string{"install", "requests"}, Plan{true, []Spec{{Name: "requests"}}}},
		{"pinned", []string{"install", "requests==2.31.0"}, Plan{true, []Spec{{Name: "requests", Version: "==2.31.0"}}}},
		{"requirements file skipped", []string{"install", "-r", "requirements.txt"}, Plan{Checked: false}},
		{"extras bracket", []string{"install", "uvicorn[standard]"}, Plan{true, []Spec{{Name: "uvicorn", Version: "[standard]"}}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Parse("pip", c.args)
			if got.Checked != c.want.Checked {
				t.Fatalf("Checked = %v, want %v", got.Checked, c.want.Checked)
			}
			if c.want.Checked && !reflect.DeepEqual(got.Packages, c.want.Packages) {
				t.Fatalf("Packages = %+v, want %+v", got.Packages, c.want.Packages)
			}
		})
	}
}

func TestParseGem(t *testing.T) {
	got := Parse("gem", []string{"install", "rails", "-v", "7.0.0"})
	want := []Spec{{Name: "rails"}}
	if !got.Checked || !reflect.DeepEqual(got.Packages, want) {
		t.Fatalf("got %+v, want checked with %+v", got, want)
	}
}

func TestParseCargo(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want Plan
	}{
		{"add", []string{"add", "serde"}, Plan{true, []Spec{{Name: "serde"}}}},
		{"add pinned", []string{"add", "serde@1.0.0"}, Plan{true, []Spec{{Name: "serde", Version: "1.0.0"}}}},
		{"install crate", []string{"install", "ripgrep"}, Plan{true, []Spec{{Name: "ripgrep"}}}},
		{"build passes through", []string{"build"}, Plan{Checked: false}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Parse("cargo", c.args)
			if got.Checked != c.want.Checked {
				t.Fatalf("Checked = %v, want %v", got.Checked, c.want.Checked)
			}
			if c.want.Checked && !reflect.DeepEqual(got.Packages, c.want.Packages) {
				t.Fatalf("Packages = %+v, want %+v", got.Packages, c.want.Packages)
			}
		})
	}
}

func TestEcosystem(t *testing.T) {
	cases := map[string]string{"npm": "npm", "pip": "pypi", "pip3": "pypi", "gem": "rubygems", "cargo": "crates", "wget": ""}
	for tool, want := range cases {
		if got := Ecosystem(tool); got != want {
			t.Errorf("Ecosystem(%q) = %q, want %q", tool, got, want)
		}
	}
}
