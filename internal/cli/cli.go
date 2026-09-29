// Package cli implements the installwall command line: install/uninstall
// the shims, check a single package, run a shimmed tool, and read the
// audit log.
package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/Arthur031221/installwall/internal/audit"
	"github.com/Arthur031221/installwall/internal/checker"
	"github.com/Arthur031221/installwall/internal/rules"
	"github.com/Arthur031221/installwall/internal/shim"
)

const helpText = `installwall: check direct package installs before they run

Usage:
  installwall install [--yes]        shim npm, pip, pip3, gem and cargo
  installwall uninstall              remove the shims
  installwall check <pkg> [flags]    check one package without installing it
  installwall audit [flags]          show the install audit log
  installwall exec <tool> -- <args>  run a shimmed tool through the checker
  installwall version                print the version

Run "installwall <command> --help" for flags on that command.
`

// Version is set by -ldflags at build time (see .goreleaser.yaml).
var Version = "dev"

// Run executes one invocation and returns the process exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stdout, helpText)
		return 0
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "install":
		return runInstall(rest, stdout, stderr)
	case "uninstall":
		return runUninstall(rest, stdout, stderr)
	case "check":
		return runCheck(rest, stdout, stderr)
	case "audit":
		return runAudit(rest, stdout, stderr)
	case "exec":
		return runExec(rest, stdout, stderr)
	case "version", "--version", "-v":
		fmt.Fprintf(stdout, "installwall %s\n", Version)
		return 0
	case "help", "--help", "-h":
		fmt.Fprint(stdout, helpText)
		return 0
	default:
		fmt.Fprintf(stderr, "installwall: unknown command %q\n\n", cmd)
		fmt.Fprint(stderr, helpText)
		return 2
	}
}

func runInstall(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	fs.SetOutput(stderr)
	yes := fs.Bool("yes", false, "also append the shim PATH export to your shell rc file")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: installwall install [--yes]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}

	self, err := os.Executable()
	if err != nil {
		fmt.Fprintf(stderr, "installwall: could not find my own path: %v\n", err)
		return 1
	}
	dir, err := shim.Install(self)
	if err != nil {
		fmt.Fprintf(stderr, "installwall: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Shims written for %s to %s\n\n", strings.Join(shim.Tools, ", "), dir)

	exportLine := fmt.Sprintf("export PATH=%q", dir+":$PATH")
	if *yes {
		rc, appended, err := appendToShellRC(exportLine)
		if err != nil {
			fmt.Fprintf(stderr, "installwall: could not update %s: %v\n", rc, err)
		} else if appended {
			fmt.Fprintf(stdout, "Added the PATH line to %s. Open a new shell for it to take effect.\n", rc)
		} else {
			fmt.Fprintf(stdout, "%s already has the PATH line.\n", rc)
		}
	} else {
		fmt.Fprintln(stdout, "Add this to your shell profile so the shims come first on PATH:")
		fmt.Fprintf(stdout, "\n  %s\n\n", exportLine)
		fmt.Fprintln(stdout, "Or rerun with --yes to append it for you.")
	}
	return 0
}

func runUninstall(args []string, stdout, stderr io.Writer) int {
	if err := shim.Uninstall(); err != nil {
		fmt.Fprintf(stderr, "installwall: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "Shims removed. Remove the PATH line from your shell profile by hand.")
	return 0
}

func runCheck(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	eco := fs.String("ecosystem", "npm", "registry to check against: npm, pypi, rubygems, crates")
	fs.StringVar(eco, "e", "npm", "shorthand for --ecosystem")
	why := fs.Bool("why", false, "print the reasoning behind the verdict")
	asJSON := fs.Bool("json", false, "print the verdict as JSON")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: installwall check <package> [--ecosystem npm|pypi|rubygems|crates] [--why] [--json]")
		fs.PrintDefaults()
	}
	// Go's flag package stops parsing at the first non-flag argument, so
	// "check requests --ecosystem pypi" would otherwise be misread as
	// three positionals. Pull the one positional (the package name) out
	// first, wherever it sits, and hand flag.Parse only the flag tokens.
	flagArgs, pkg, err := extractPositional(args, map[string]bool{"-e": true, "--ecosystem": true, "-ecosystem": true})
	if err != nil {
		fmt.Fprintln(stderr, err)
		fs.Usage()
		return 2
	}
	if err := fs.Parse(flagArgs); err != nil {
		return 2
	}
	if pkg == "" {
		fs.Usage()
		return 2
	}

	c, err := checker.New()
	if err != nil {
		fmt.Fprintf(stderr, "installwall: %v\n", err)
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	v := c.Check(ctx, rules.Ecosystem(*eco), pkg)

	if *asJSON {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(v)
	} else {
		printVerdict(stdout, v, *why)
	}

	if v.Level == checker.Block {
		return 1
	}
	return 0
}

// extractPositional pulls the single non-flag argument (the package name)
// out of args, wherever it appears, and returns the remaining flag
// tokens in their original relative order. valueFlags lists flag names
// that consume the following token as their value.
func extractPositional(args []string, valueFlags map[string]bool) (flagArgs []string, positional string, err error) {
	skipNext := false
	for _, tok := range args {
		if skipNext {
			flagArgs = append(flagArgs, tok)
			skipNext = false
			continue
		}
		if strings.HasPrefix(tok, "-") {
			flagArgs = append(flagArgs, tok)
			if valueFlags[tok] && !strings.Contains(tok, "=") {
				skipNext = true
			}
			continue
		}
		if positional != "" {
			return nil, "", fmt.Errorf("installwall: unexpected extra argument %q", tok)
		}
		positional = tok
	}
	return flagArgs, positional, nil
}

func printVerdict(w io.Writer, v checker.Verdict, why bool) {
	switch v.Level {
	case checker.Block:
		fmt.Fprintf(w, "BLOCK  %s (%s)\n", v.Package, v.Ecosystem)
	case checker.Warn:
		fmt.Fprintf(w, "WARN   %s (%s)\n", v.Package, v.Ecosystem)
	default:
		fmt.Fprintf(w, "ALLOW  %s (%s)\n", v.Package, v.Ecosystem)
	}
	if len(v.Reasons) == 0 {
		if why {
			fmt.Fprintln(w, "  no rule matched")
		}
		return
	}
	if why || v.Level == checker.Block {
		for _, r := range v.Reasons {
			fmt.Fprintf(w, "  [%s] %s\n", r.Rule, r.Message)
			if r.Source != "" {
				fmt.Fprintf(w, "         source: %s\n", r.Source)
			}
		}
	}
}

func runAudit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("audit", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "print the log as JSON")
	limit := fs.Int("limit", 20, "number of most recent entries to show (0 for all)")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: installwall audit [--json] [--limit N]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}

	entries, err := audit.ReadAll()
	if err != nil {
		fmt.Fprintf(stderr, "installwall: %v\n", err)
		return 1
	}
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].Time.After(entries[j].Time) })
	if *limit > 0 && len(entries) > *limit {
		entries = entries[:*limit]
	}

	if *asJSON {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(entries)
		return 0
	}

	if len(entries) == 0 {
		fmt.Fprintln(stdout, "No installs logged yet. Run \"installwall install\" and use npm, pip, gem or cargo as usual.")
		return 0
	}
	fmt.Fprintf(stdout, "%-20s %-6s %-9s %-30s %-6s %s\n", "TIME", "TOOL", "ECOSYSTEM", "PACKAGE", "LEVEL", "REASON")
	for _, e := range entries {
		reason := ""
		if len(e.Reasons) > 0 {
			reason = e.Reasons[0]
		}
		forced := ""
		if e.Forced {
			forced = " (forced)"
		}
		fmt.Fprintf(stdout, "%-20s %-6s %-9s %-30s %-6s %s%s\n",
			e.Time.Local().Format("2006-01-02 15:04:05"), e.Tool, e.Ecosystem, e.Package, e.Level, reason, forced)
	}
	return 0
}

// runExec is what the shim scripts call. It checks every package the
// command line would install, then execs the real tool (or refuses to,
// on a block). Set INSTALLWALL_FORCE=1 to downgrade a block to a warning,
// for CI jobs that must not be interrupted.
func runExec(args []string, stdout, stderr io.Writer) int {
	dashIdx := -1
	for i, a := range args {
		if a == "--" {
			dashIdx = i
			break
		}
	}
	var tool string
	var toolArgs []string
	if dashIdx >= 0 {
		if dashIdx == 0 {
			fmt.Fprintln(stderr, "Usage: installwall exec <tool> -- <args...>")
			return 2
		}
		tool = args[0]
		toolArgs = args[dashIdx+1:]
	} else if len(args) >= 1 {
		tool = args[0]
		toolArgs = args[1:]
	} else {
		fmt.Fprintln(stderr, "Usage: installwall exec <tool> -- <args...>")
		return 2
	}

	force := os.Getenv("INSTALLWALL_FORCE") == "1"
	plan := shim.Parse(tool, toolArgs)
	eco := rules.Ecosystem(shim.Ecosystem(tool))

	if plan.Checked && eco != "" {
		c, err := checker.New()
		if err != nil {
			fmt.Fprintf(stderr, "installwall: %v, allowing (fail-open)\n", err)
		} else {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			blocked := false
			for _, spec := range plan.Packages {
				v := c.Check(ctx, eco, spec.Name)
				level := v.Level
				if level == checker.Block && force {
					fmt.Fprintf(stderr, "installwall: WARN (forced) %s\n", spec.Name)
					for _, r := range v.Reasons {
						fmt.Fprintf(stderr, "  [%s] %s\n", r.Rule, r.Message)
					}
					level = checker.Warn
				} else if level == checker.Block {
					fmt.Fprintf(stderr, "installwall: BLOCK %s\n", spec.Name)
					for _, r := range v.Reasons {
						fmt.Fprintf(stderr, "  [%s] %s\n", r.Rule, r.Message)
					}
					blocked = true
				} else if level == checker.Warn {
					fmt.Fprintf(stderr, "installwall: WARN %s\n", spec.Name)
					for _, r := range v.Reasons {
						fmt.Fprintf(stderr, "  [%s] %s\n", r.Rule, r.Message)
					}
				}
				var reasons []string
				for _, r := range v.Reasons {
					reasons = append(reasons, r.Rule+": "+r.Message)
				}
				_ = audit.Append(audit.Entry{
					Time: time.Now(), Tool: tool, Ecosystem: string(eco), Package: spec.Name,
					Level: string(level), Reasons: reasons, Forced: v.Level == checker.Block && force,
				})
			}
			cancel()
			if blocked {
				fmt.Fprintln(stderr, "installwall: install blocked. Set INSTALLWALL_FORCE=1 to override, or run \"installwall check <pkg> --why\" for detail.")
				return 1
			}
		}
	}

	return execReal(tool, toolArgs, stdout, stderr)
}
