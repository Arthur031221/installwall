# Contributing

## Setup

```sh
git clone https://github.com/Arthur031221/installwall
cd installwall
go build ./...
go test ./...
```

Go 1.27 or newer. No other dependencies.

## Before sending a pull request

- `gofmt -l .` must print nothing.
- `go vet ./...` must be clean.
- `go test ./...` must pass. Add a test for new rule logic in `internal/checker` or `internal/rules`, the same way the existing rules are tested.
- Keep the dependency list short. This project has zero third-party Go dependencies on purpose.

## Adding a known-malicious indicator

`internal/rules/data/malicious.json` only takes entries backed by a public advisory or write-up. Every entry needs a `source` URL. Do not add a package name because you suspect it, only because a cited source names it.

## Adding a registry adapter

Implement the `registry.Registry` interface in `internal/registry`, add it to `checker.New`, and add a top-name list under `internal/rules/data/`. See `internal/registry/npm.go` for the shortest example.

## Reporting a security issue

Open a regular issue. There is no separate disclosure process for a project this size.
