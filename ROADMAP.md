# Roadmap

- Check package names resolved from `package.json`, lockfiles, requirements
  files, and transitive dependencies. The first release checks only names
  explicitly supplied to supported install commands.
- Add version-aware OSV matching. The current local indicator list excludes
  known advisories that affect one removed version of an otherwise legitimate
  package.
- Build an independent, reproducible detection benchmark from cited malicious
  package advisories and a separately chosen benign set. Do not use the
  embedded blocklist as both the rule and the test set.
- Expand parser coverage for package-manager flags and wrapper commands after
  real install logs show the missing forms.
