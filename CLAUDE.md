# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

```bash
# Run all tests
go test ./...

# Run a single test
go test -run TestLogger_Levels ./...

# Test coverage
core go cov            # generate report
core go cov --open     # open in browser

# Code quality
core go fmt            # format
core go lint           # lint
core go vet            # vet
core go qa             # fmt + lint + vet + tests
core go qa full        # + race detector, vuln scan, security audit
```

The `core` CLI is optional; plain `go test` and `gofmt` work without it.

## Architecture

Single-package library (`package log`) split into two files that wire together:

- **log.go** — `Logger` type, `Level` enum (Quiet→Error→Warn→Info→Debug), key-value formatting with redaction and injection prevention, `Style*` function hooks for decoration, `RotationWriterFactory` injection point, default logger with package-level proxy functions
- **errors.go** — `Err` structured error type (Op/Msg/Err/Code), creation helpers (`E`, `Wrap`, `WrapCode`, `NewCode`), introspection (`Op`, `ErrCode`, `Root`, `StackTrace`), combined log-and-return helpers (`LogError`, `LogWarn`, `Must`), stdlib wrappers (`Is`, `As`, `Join`)

The logger automatically extracts `op` and `stack` from `*Err` values found in key-value pairs. `Wrap` propagates error codes upward through the chain.

Zero runtime dependencies. `testify` is test-only.

## Conventions

- **UK English** in comments and documentation (colour, organisation, centre)
- **Test naming**: `_Good` (happy path), `_Bad` (expected errors), `_Ugly` (edge cases/panics)
- **Commit messages**: conventional commits (`feat`, `fix`, `docs`, `chore`, etc.)
- **Dependencies**: no new runtime dependencies without justification; use `RotationWriterFactory` injection point for log rotation
- Requires **Go 1.26+** (uses `iter.Seq`)
- Module path: `forge.lthn.ai/core/go-log`
