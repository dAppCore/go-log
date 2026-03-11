---
title: go-log
description: Structured logging and error handling for Core applications
---

# go-log

`forge.lthn.ai/core/go-log` provides structured logging and contextual error
handling for Go applications built on the Core framework.  It is a small,
zero-dependency library (only `testify` at test time) that replaces ad-hoc
`fmt.Println` / `log.Printf` calls with level-filtered, key-value structured
output and a rich error type that carries operation context through the call
stack.

## Quick Start

```go
import "forge.lthn.ai/core/go-log"

// Use the package-level default logger straight away
log.SetLevel(log.LevelDebug)
log.Info("server started", "port", 8080)
log.Warn("high latency", "ms", 320)
log.Error("request failed", "err", err)

// Security events are always visible at Error level
log.Security("brute force detected", "ip", "10.0.0.1", "attempts", 47)
```

### Creating a Custom Logger

```go
logger := log.New(log.Options{
    Level:      log.LevelInfo,
    Output:     os.Stdout,
    RedactKeys: []string{"password", "token", "secret"},
})

logger.Info("login", "user", "admin", "password", "hunter2")
// Output: 14:32:01 [INF] login user="admin" password="[REDACTED]"
```

### Structured Errors

```go
// Create an error with operational context
err := log.E("db.Connect", "connection refused", underlyingErr)

// Wrap errors as they bubble up through layers
err = log.Wrap(err, "user.Save", "failed to persist user")

// Inspect the chain
log.Op(err)             // "user.Save"
log.Root(err)           // the original underlyingErr
log.StackTrace(err)     // ["user.Save", "db.Connect"]
log.FormatStackTrace(err) // "user.Save -> db.Connect"
```

### Combined Log-and-Return

```go
if err != nil {
    return log.LogError(err, "handler.Process", "request failed")
    // Logs at Error level AND returns a wrapped error -- one line instead of three
}
```

## Package Layout

| File | Purpose |
|------|---------|
| `log.go` | Logger type, log levels, key-value formatting, redaction, default logger, `Username()` helper |
| `errors.go` | `Err` structured error type, creation helpers (`E`, `Wrap`, `WrapCode`, `NewCode`), introspection (`Op`, `ErrCode`, `Root`, `StackTrace`), combined log-and-return helpers (`LogError`, `LogWarn`, `Must`) |
| `log_test.go` | Tests for the Logger: level filtering, key-value output, redaction, injection prevention, security logging |
| `errors_test.go` | Tests for structured errors: creation, wrapping, code propagation, introspection, stack traces, log-and-return helpers |

## Dependencies

| Module | Purpose |
|--------|---------|
| Go standard library only | Runtime -- no external dependencies |
| `github.com/stretchr/testify` | Test assertions (test-only) |

The package deliberately avoids external runtime dependencies.  Log rotation is
supported through an optional `RotationWriterFactory` hook that can be wired up
by `core/go-io` or any other provider -- go-log itself carries no file-rotation
code.

## Module Path

```
forge.lthn.ai/core/go-log
```

Requires **Go 1.26+** (uses `iter.Seq` from the standard library).

## Licence

EUPL-1.2
