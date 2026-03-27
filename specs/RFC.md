# log
**Import:** `dappco.re/go/core/log`
**Files:** 2

## Types

### Err
`type Err struct`

Structured error wrapper that carries operation context, a human-readable message, an optional wrapped cause, and an optional machine-readable code.

Fields:
- `Op string`: operation name. When non-empty, `Error` prefixes the formatted message with `Op + ": "`.
- `Msg string`: human-readable message stored on the error and returned by `Message`.
- `Err error`: wrapped cause returned by `Unwrap`.
- `Code string`: optional machine-readable code. When non-empty, `Error` includes it in square brackets.

Methods:
- `func (e *Err) Error() string`: formats the error text from `Op`, `Msg`, `Code`, and `Err`. The result omits missing parts, so the output can be `"{Msg}"`, `"{Msg} [{Code}]"`, `"{Msg}: {Err}"`, or `"{Op}: {Msg} [{Code}]: {Err}"`.
- `func (e *Err) Unwrap() error`: returns `e.Err`.

### Level
`type Level int`

Logging verbosity enum used by `Logger` and `Options`.

Methods:
- `func (l Level) String() string`: returns `quiet`, `error`, `warn`, `info`, or `debug`. Any other value returns `unknown`.

### Logger
`type Logger struct`

Concurrency-safe structured logger. `New` clones the configured redaction keys, stores the configured level and writer, and initializes all style hooks to identity functions.

Each log call writes one line in the form `HH:MM:SS {prefix} {msg}` followed by space-separated key/value pairs. String values are rendered with Go `%q` quoting, redacted keys are replaced with `"[REDACTED]"`, a trailing key without a value renders as `<nil>`, and any `error` value in `keyvals` can cause `op` and `stack` fields to be appended automatically if those keys were not already supplied.

Fields:
- `StyleTimestamp func(string) string`: transforms the rendered `HH:MM:SS` timestamp before it is written.
- `StyleDebug func(string) string`: transforms the debug prefix passed to debug log lines.
- `StyleInfo func(string) string`: transforms the info prefix passed to info log lines.
- `StyleWarn func(string) string`: transforms the warn prefix passed to warning log lines.
- `StyleError func(string) string`: transforms the error prefix passed to error log lines.
- `StyleSecurity func(string) string`: transforms the security prefix passed to security log lines.

Methods:
- `func (l *Logger) SetLevel(level Level)`: sets the logger’s current threshold.
- `func (l *Logger) Level() Level`: returns the logger’s current threshold.
- `func (l *Logger) SetOutput(w goio.Writer)`: replaces the writer used for future log lines.
- `func (l *Logger) SetRedactKeys(keys ...string)`: replaces the exact-match key list whose values are masked during formatting.
- `func (l *Logger) Debug(msg string, keyvals ...any)`: emits a debug line when the logger level is at least `LevelDebug`.
- `func (l *Logger) Info(msg string, keyvals ...any)`: emits an info line when the logger level is at least `LevelInfo`.
- `func (l *Logger) Warn(msg string, keyvals ...any)`: emits a warning line when the logger level is at least `LevelWarn`.
- `func (l *Logger) Error(msg string, keyvals ...any)`: emits an error line when the logger level is at least `LevelError`.
- `func (l *Logger) Security(msg string, keyvals ...any)`: emits a security line with the security style prefix and the same visibility threshold as `LevelError`.

### Options
`type Options struct`

Constructor input for `New`.

Fields:
- `Level Level`: initial logger threshold.
- `Output goio.Writer`: destination used when rotation is not selected.
- `Rotation *RotationOptions`: optional rotation configuration. `New` uses rotation only when this field is non-nil, `Rotation.Filename` is non-empty, and `RotationWriterFactory` is non-nil.
- `RedactKeys []string`: keys whose values should be masked in formatted log output.

### RotationOptions
`type RotationOptions struct`

Rotation configuration passed through to `RotationWriterFactory` when `New` selects a rotating writer.

Fields:
- `Filename string`: log file path. `New` only attempts rotation when this field is non-empty.
- `MaxSize int`: value forwarded to the rotation writer factory.
- `MaxAge int`: value forwarded to the rotation writer factory.
- `MaxBackups int`: value forwarded to the rotation writer factory.
- `Compress bool`: value forwarded to the rotation writer factory.

## Functions

### AllOps
`func AllOps(err error) iter.Seq[string]`

Returns an iterator over non-empty `Op` values found by repeatedly calling `errors.Unwrap` on `err`. Operations are yielded from the outermost `*Err` to the innermost one.

### As
`func As(err error, target any) bool`

Thin wrapper around `errors.As`.

### Debug
`func Debug(msg string, keyvals ...any)`

Calls `Default().Debug(msg, keyvals...)`.

### Default
`func Default() *Logger`

Returns the package-level default logger. The package initializes it with `New(Options{Level: LevelInfo})`.

### E
`func E(op, msg string, err error) error`

Returns `&Err{Op: op, Msg: msg, Err: err}` as an `error`. It always returns a non-nil error value, even when `err` is nil.

### ErrCode
`func ErrCode(err error) string`

If `err` contains an `*Err`, returns its `Code`. Otherwise returns the empty string.

### Error
`func Error(msg string, keyvals ...any)`

Calls `Default().Error(msg, keyvals...)`.

### FormatStackTrace
`func FormatStackTrace(err error) string`

Collects `AllOps(err)` and joins the operations with `" -> "`. Returns the empty string when no operations are found.

### Info
`func Info(msg string, keyvals ...any)`

Calls `Default().Info(msg, keyvals...)`.

### Is
`func Is(err, target error) bool`

Thin wrapper around `errors.Is`.

### Join
`func Join(errs ...error) error`

Thin wrapper around `errors.Join`.

### LogError
`func LogError(err error, op, msg string) error`

If `err` is nil, returns nil. Otherwise wraps the error with `Wrap(err, op, msg)`, logs `msg` through the default logger at error level with key/value pairs `"op", op, "err", err`, and returns the wrapped error.

### LogWarn
`func LogWarn(err error, op, msg string) error`

If `err` is nil, returns nil. Otherwise wraps the error with `Wrap(err, op, msg)`, logs `msg` through the default logger at warn level with key/value pairs `"op", op, "err", err`, and returns the wrapped error.

### Message
`func Message(err error) string`

Returns the `Msg` field from the first matching `*Err`. If `err` is nil, returns the empty string. For non-`*Err` errors, returns `err.Error()`.

### Must
`func Must(err error, op, msg string)`

If `err` is nil, does nothing. Otherwise logs `msg` through the default logger at error level with key/value pairs `"op", op, "err", err`, then panics with `Wrap(err, op, msg)`.

### New
`func New(opts Options) *Logger`

Constructs a logger from `opts`. It prefers a rotating writer only when `opts.Rotation` is non-nil, `opts.Rotation.Filename` is non-empty, and `RotationWriterFactory` is set; otherwise it uses `opts.Output`. If neither path yields a writer, it falls back to `os.Stderr`.

### NewCode
`func NewCode(code, msg string) error`

Returns `&Err{Msg: msg, Code: code}` as an `error`.

### NewError
`func NewError(text string) error`

Thin wrapper around `errors.New`.

### Op
`func Op(err error) string`

If `err` contains an `*Err`, returns its `Op`. Otherwise returns the empty string.

### Root
`func Root(err error) error`

Repeatedly unwraps `err` with `errors.Unwrap` until no further wrapped error exists, then returns the last error in that chain. If `err` is nil, returns nil.

### Security
`func Security(msg string, keyvals ...any)`

Calls `Default().Security(msg, keyvals...)`.

### SetDefault
`func SetDefault(l *Logger)`

Replaces the package-level default logger with `l`.

### SetLevel
`func SetLevel(level Level)`

Calls `Default().SetLevel(level)`.

### SetRedactKeys
`func SetRedactKeys(keys ...string)`

Calls `Default().SetRedactKeys(keys...)`.

### StackTrace
`func StackTrace(err error) []string`

Collects `AllOps(err)` into a slice in outermost-to-innermost order. When no operations are found, the returned slice is nil.

### Username
`func Username() string`

Returns the current username by trying `user.Current()` first, then the `USER` environment variable, then the `USERNAME` environment variable.

### Warn
`func Warn(msg string, keyvals ...any)`

Calls `Default().Warn(msg, keyvals...)`.

### Wrap
`func Wrap(err error, op, msg string) error`

If `err` is nil, returns nil. Otherwise returns a new `*Err` containing `op`, `msg`, and `err`. If the wrapped error chain already contains an `*Err` with a non-empty `Code`, the new wrapper copies that code.

### WrapCode
`func WrapCode(err error, code, op, msg string) error`

Returns nil only when both `err` is nil and `code` is empty. In every other case it returns `&Err{Op: op, Msg: msg, Err: err, Code: code}` as an `error`.
