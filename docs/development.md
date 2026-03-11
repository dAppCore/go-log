---
title: Development
description: How to build, test, and contribute to go-log
---

# Development

## Prerequisites

- **Go 1.26+** -- go-log uses `iter.Seq` from the standard library.
- **Core CLI** (`core` binary) -- for running tests and quality checks.
  Build it from `~/Code/host-uk/core` with `task cli:build`.

If you do not have the Core CLI, plain `go test` works fine.

## Running Tests

```bash
# All tests via Core CLI
core go test

# All tests via plain Go
go test ./...

# Single test
core go test --run TestLogger_Levels
# or
go test -run TestLogger_Levels ./...
```

### Test Coverage

```bash
core go cov            # generate coverage report
core go cov --open     # generate and open in browser
```

## Code Quality

```bash
core go fmt            # format with gofmt
core go lint           # run linters
core go vet            # run go vet

core go qa             # all of the above + tests
core go qa full        # + race detector, vulnerability scan, security audit
```

## Test Naming Convention

Tests follow the `_Good` / `_Bad` / `_Ugly` suffix pattern:

| Suffix | Meaning |
|--------|---------|
| `_Good` | Happy path -- the function behaves correctly with valid input |
| `_Bad` | Expected error conditions -- the function returns an error or handles invalid input gracefully |
| `_Ugly` | Edge cases, panics, or truly degenerate input |

Examples from the codebase:

```go
func TestErr_Error_Good(t *testing.T)       { /* valid Err produces correct string */ }
func TestMust_Good_NoError(t *testing.T)    { /* nil error does not panic */ }
func TestMust_Ugly_Panics(t *testing.T)     { /* non-nil error triggers panic */ }
```

## Project Structure

```
go-log/
  log.go            # Logger, levels, formatting, default logger
  log_test.go       # Logger tests
  errors.go         # Err type, creation, introspection, log-and-return helpers
  errors_test.go    # Error tests
  go.mod            # Module definition
  go.sum            # Dependency checksums
  .core/
    build.yaml      # Build configuration (targets, flags)
    release.yaml    # Release configuration (changelog rules)
  docs/
    index.md        # This documentation
    architecture.md # Internal design
    development.md  # Build and contribution guide
```

## Contributing

### Coding Standards

- **UK English** in comments and documentation (colour, organisation, centre).
- `declare(strict_types=1)` does not apply (this is Go), but do use strong
  typing: all exported functions should have explicit parameter and return types.
- Tests use the **Pest-style naming** adapted for Go: descriptive names with
  `_Good` / `_Bad` / `_Ugly` suffixes.
- Format with `gofmt` (or `core go fmt`).  The CI pipeline will reject
  unformatted code.

### Commit Messages

Use conventional commits:

```
type(scope): description
```

Common types: `feat`, `fix`, `perf`, `refactor`, `test`, `docs`, `chore`.

Include the co-author trailer:

```
Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
```

### Adding a New Log Level or Feature

1. Add the level constant to the `Level` iota block in `log.go`.
2. Add its `String()` case.
3. Add a method on `*Logger` and a package-level proxy function.
4. If the level needs a distinct prefix (like `[SEC]` for Security), add a
   `Style*` field to the Logger struct and initialise it to `identity` in `New`.
5. Write tests covering `_Good` and at least one `_Bad` or `_Ugly` case.

### Dependencies Policy

go-log has **zero runtime dependencies**.  `testify` is permitted for tests
only.  Any new dependency must be justified -- prefer the standard library.

Log rotation is handled via the `RotationWriterFactory` injection point, not
by importing a rotation library directly.

## Licence

EUPL-1.2
