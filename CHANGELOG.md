# Changelog

## 0.1.0

Initial release.

- `installwall install` shims npm, pip, pip3, gem and cargo on PATH.
- `installwall check <pkg>` checks one package without installing it, with `--ecosystem`, `--why` and `--json`.
- `installwall audit` prints the append-only JSONL log of every install a shim has checked.
- Three rules: known-malicious exact match (seeded from published incident write-ups), typosquat by edit distance 1 against a popular-package list per registry, and packages first published under 72 hours ago.
- Registry adapters for npm, PyPI, RubyGems and crates.io, all fail open (allow with a warning) when a registry cannot be reached.
- Benchmark script and results in `bench/`.
