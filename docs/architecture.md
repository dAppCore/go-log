---
title: Architecture
description: Internals of go-log -- types, data flow, and design decisions
---

# Architecture

go-log is split into two complementary halves that share a single package:
**structured logging** (`log.go`) and **structured errors** (`errors.go`).
The two halves are wired together so that when an `*Err` value appears in a
log line's key-value pairs the logger automatically extracts the operation
name and stack trace.

## Key Types

### Level

```go
type Level int

const (
    LevelQuiet Level = iota  // suppress all output
    LevelError               // errors only
    LevelWarn                // warnings + errors
    LevelInfo                // info + warnings + errors
    LevelDebug               // everything
)
```

Levels are ordered by increasing verbosity.  A message is emitted only when its
level is less than or equal to the logger's configured level.  `LevelQuiet`
suppresses all output, including errors.

### Logger

```go
type Logger struct {
    mu         sync.RWMutex
    level      Level
    output     io.Writer
    redactKeys []string

    // Overridable style functions
    StyleTimestamp func(string) string
    StyleDebug     func(string) string
    StyleInfo      func(string) string
    StyleWarn      func(string) string
    StyleError     func(string) string
    StyleSecurity  func(string) string
}
```

All fields are protected by `sync.RWMutex`, making the logger safe for
concurrent use.  The `Style*` function fields default to the identity function;
consumers (such as a TUI layer) can replace them to add ANSI colour or other
decoration without forking the logger.

### Err

```go
type Err struct {
    Op   string // e.g. "user.Save"
    Msg  string // human-readable description
    Err  error  // underlying cause (optional)
    Code string // machine-readable code (optional, e.g. "VALIDATION_FAILED")
    Retryable  bool          // whether the caller can retry the operation
    RetryAfter *time.Duration // optional retry delay hint
    NextAction string        // suggested next step when not retryable
}
```

`Err` implements both the `error` and `Unwrap` interfaces so it participates
fully in the standard `errors.Is` / `errors.As` machinery.

### Options and RotationOptions

```go
type Options struct {
    Level      Level
    Output     io.Writer
    Rotation   *RotationOptions
    RedactKeys []string
}

type RotationOptions struct {
    Filename   string
    MaxSize    int  // megabytes, default 100
    MaxAge     int  // days, default 28
    MaxBackups int  // default 5
    Compress   bool // default true
}
```

When `Rotation` is provided and `RotationWriterFactory` is set, the logger
writes to a rotating file instead of the supplied `Output`.

## Data Flow

### Logging a Message

```
caller
  |
  v
log.Info("msg", "k1", v1, "k2", v2)
  |
  v
defaultLogger.Info(...)          -- package-level proxy
  |
  v
shouldLog(LevelInfo)             -- RLock, compare level, RUnlock
  |  (if filtered out, return immediately)
  v
log(LevelInfo, "[INF]", ...)
  |
  +-- format timestamp with StyleTimestamp
  +-- scan keyvals for error values:
  |     if any value implements `error`:
  |       extract Op  -> append "op"    key if not already present
  |       extract FormatStackTrace -> append "stack" key if not already present
  |       extract recovery hints -> append "retryable",
  |                                 "retry_after_seconds",
  |                                 "next_action" if not already present
  +-- format key-value pairs:
  |     string values -> %q (quoted, injection-safe)
  |     other values  -> %v
  |     redacted keys -> "[REDACTED]"
  +-- write single line to output:
        "<timestamp> <prefix> <msg> <kvpairs>\n"
```

### Building an Error Chain

```
root cause (any error)
  |
  v
log.E("db.Query", "query failed", rootErr)
  |   -> &Err{Op:"db.Query", Msg:"query failed", Err:rootErr}
  v
log.Wrap(err, "repo.FindUser", "user lookup failed")
  |   -> &Err{Op:"repo.FindUser", Msg:"user lookup failed", Err:prev}
  v
log.Wrap(err, "handler.Get", "request failed")
  |   -> &Err{Op:"handler.Get", Msg:"request failed", Err:prev}
  v
log.StackTrace(err)
  -> ["handler.Get", "repo.FindUser", "db.Query"]

log.FormatStackTrace(err)
  -> "handler.Get -> repo.FindUser -> db.Query"

log.Root(err)
  -> rootErr  (the original cause)
```

`Wrap` preserves any `Code` from a wrapped `*Err`, so error codes propagate
upward automatically.

### Combined Log-and-Return

`LogError` and `LogWarn` combine two operations into one call:

```go
func LogError(err error, op, msg string) error {
    wrapped := Wrap(err, op, msg)       // 1. wrap with context
    defaultLogger.Error(msg, ...)       // 2. log at Error level
    return wrapped                       // 3. return wrapped error
}
```

Both return `nil` when given a `nil` error, making them safe to use
unconditionally.

`Must` follows the same pattern but panics instead of returning, intended for
startup-time invariants that must hold.

## Security Features

### Log Injection Prevention

String values in key-value pairs are formatted with `%q`, which escapes
newlines, quotes, and other control characters.  This prevents an attacker
from injecting fake log lines via user-controlled input:

```go
l.Info("msg", "key", "value\n[SEC] injected message")
// Output: ... key="value\n[SEC] injected message"   (single line, escaped)
```

### Key Redaction

Keys listed in `RedactKeys` have their values replaced with `[REDACTED]`:

```go
l := log.New(log.Options{
    Level:      log.LevelInfo,
    RedactKeys: []string{"password", "token"},
})
l.Info("login", "user", "admin", "password", "secret123")
// Output: ... user="admin" password="[REDACTED]"
```

### Security Log Level

The `Security` method uses a dedicated `[SEC]` prefix and logs at `LevelError`
so that security events remain visible even in restrictive configurations:

```go
l.Security("unauthorised access", "user", "admin", "ip", "10.0.0.1")
// Output: 14:32:01 [SEC] unauthorised access user="admin" ip="10.0.0.1"
```

## Log Rotation

go-log defines the `RotationOptions` struct and an optional
`RotationWriterFactory` variable:

```go
var RotationWriterFactory func(RotationOptions) io.WriteCloser
```

This is a seam for dependency injection.  The `core/go-io` package (or any
other provider) can set this factory at init time.  When `Options.Rotation` is
provided and the factory is non-nil, the logger creates a rotating file writer
instead of using `Options.Output`.

This design keeps go-log free of file-system and compression dependencies.

## Concurrency Model

- All Logger fields are guarded by `sync.RWMutex`.
- `shouldLog` and `log` acquire a read lock to snapshot the level, output, and
  redact keys.
- `SetLevel`, `SetOutput`, and `SetRedactKeys` acquire a write lock.
- The default logger is a package-level variable set at init time.  `SetDefault`
  replaces it (not goroutine-safe itself, but intended for use during startup).

## Default Logger

A package-level `defaultLogger` is created at import time with `LevelInfo` and
`os.Stderr` output.  All top-level functions (`log.Info`, `log.Error`, etc.)
delegate to it.  Use `log.SetDefault` to replace it with a custom instance.
