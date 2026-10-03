// Package shim generates the PATH-shadowing wrapper scripts for npm, pip,
// gem and cargo, and extracts the package names an install-like command
// line would actually install.
package shim

import (
	"regexp"
	"strings"
)

// Spec is one package the command line would install.
type Spec struct {
	Name    string
	Version string
}

// Plan is the result of parsing one command line for one tool.
type Plan struct {
	// Checked is true when this invocation installs new packages and
	// should be checked. False for things like "npm run build" or
	// "npm ci", which install nothing new and pass straight through.
	Checked  bool
	Packages []Spec
}

// flagsWithValue lists, per tool, the flags that consume the following
// token as a value rather than a package name (for example "pip install
// -r requirements.txt").
var flagsWithValue = map[string]map[string]bool{
	"pip":   {"-r": true, "--requirement": true, "-c": true, "--constraint": true, "-i": true, "--index-url": true, "--extra-index-url": true, "-t": true, "--target": true},
	"pip3":  {"-r": true, "--requirement": true, "-c": true, "--constraint": true, "-i": true, "--index-url": true, "--extra-index-url": true, "-t": true, "--target": true},
	"gem":   {"-v": true, "--version": true, "--source": true, "--platform": true},
	"npm":   {"--tag": true, "--registry": true, "-w": true, "--workspace": true},
	"cargo": {"-F": true, "--features": true},
}

// installSubcommands lists, per tool, the subcommands that install
// packages named on the command line.
var installSubcommands = map[string]map[string]bool{
	"npm":   {"install": true, "i": true, "add": true},
	"pip":   {"install": true},
	"pip3":  {"install": true},
	"gem":   {"install": true},
	"cargo": {"install": true, "add": true},
}

// Parse extracts the packages a shimmed command line would install.
func Parse(tool string, args []string) Plan {
	if len(args) == 0 {
		return Plan{Checked: false}
	}
	sub := args[0]
	subs, ok := installSubcommands[tool]
	if !ok || !subs[sub] {
		return Plan{Checked: false}
	}

	valueFlags := flagsWithValue[tool]
	var positionals []string
	skipNext := false
	for _, tok := range args[1:] {
		if skipNext {
			skipNext = false
			continue
		}
		if strings.HasPrefix(tok, "-") {
			if valueFlags[tok] {
				skipNext = true
			}
			continue
		}
		positionals = append(positionals, tok)
	}

	if len(positionals) == 0 {
		// "npm install" with no args installs from package.json, "cargo
		// install" with no args does not exist, "pip install" with no
		// args is a usage error. Nothing new to check either way.
		return Plan{Checked: false}
	}

	var specs []Spec
	for _, p := range positionals {
		specs = append(specs, splitSpec(tool, p))
	}
	return Plan{Checked: true, Packages: specs}
}

var pypiVersionSplit = regexp.MustCompile(`[=<>!~\[]`)

func splitSpec(tool string, raw string) Spec {
	switch tool {
	case "npm":
		return splitNPMSpec(raw)
	case "pip", "pip3":
		loc := pypiVersionSplit.FindStringIndex(raw)
		if loc == nil {
			return Spec{Name: raw}
		}
		return Spec{Name: raw[:loc[0]], Version: raw[loc[0]:]}
	case "gem":
		if idx := strings.Index(raw, ":"); idx > 0 {
			return Spec{Name: raw[:idx], Version: raw[idx+1:]}
		}
		return Spec{Name: raw}
	case "cargo":
		if idx := strings.LastIndex(raw, "@"); idx > 0 {
			return Spec{Name: raw[:idx], Version: raw[idx+1:]}
		}
		return Spec{Name: raw}
	default:
		return Spec{Name: raw}
	}
}

// splitNPMSpec handles plain "name@version" and scoped "@scope/name@version".
func splitNPMSpec(raw string) Spec {
	if strings.HasPrefix(raw, "@") {
		rest := raw[1:]
		if idx := strings.Index(rest, "@"); idx >= 0 {
			return Spec{Name: "@" + rest[:idx], Version: rest[idx+1:]}
		}
		return Spec{Name: raw}
	}
	if idx := strings.Index(raw, "@"); idx > 0 {
		return Spec{Name: raw[:idx], Version: raw[idx+1:]}
	}
	return Spec{Name: raw}
}

// Ecosystem maps a shimmed tool name to the rules.Ecosystem it checks
// against. Kept here, not in package rules, to avoid an import cycle
// between shim and the CLI wiring that already imports rules directly.
func Ecosystem(tool string) string {
	switch tool {
	case "npm":
		return "npm"
	case "pip", "pip3":
		return "pypi"
	case "gem":
		return "rubygems"
	case "cargo":
		return "crates"
	default:
		return ""
	}
}
