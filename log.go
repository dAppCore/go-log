// Package log provides structured logging and error handling for Core applications.
//
//	log.SetLevel(log.LevelDebug)
//	log.Info("server started", "port", 8080)
//	log.Error("failed to connect", "err", err)
package log

import (
	"fmt"
	goio "io"
	"os"
	"os/user"
	"slices"
	"strings"
	"sync"
	"time"
)

// Level defines logging verbosity.
type Level int

// Logging level constants ordered by increasing verbosity.
const (
	// LevelQuiet suppresses all log output.
	LevelQuiet Level = iota
	// LevelError shows only error messages.
	LevelError
	// LevelWarn shows warnings and errors.
	LevelWarn
	// LevelInfo shows informational messages, warnings, and errors.
	LevelInfo
	// LevelDebug shows all messages including debug details.
	LevelDebug
)

const (
	defaultRotationMaxSize    = 100
	defaultRotationMaxAge     = 28
	defaultRotationMaxBackups = 5
)

func normaliseLevel(level Level) Level {
	if level < LevelQuiet || level > LevelDebug {
		return LevelInfo
	}
	return level
}

// String returns the level name.
func (l Level) String() string {
	switch l {
	case LevelQuiet:
		return "quiet"
	case LevelError:
		return "error"
	case LevelWarn:
		return "warn"
	case LevelInfo:
		return "info"
	case LevelDebug:
		return "debug"
	default:
		return "unknown"
	}
}

// Logger provides structured logging.
type Logger struct {
	mu     sync.RWMutex
	level  Level
	output goio.Writer

	// RedactKeys is a list of keys whose values should be masked in logs.
	redactKeys []string

	// StyleTimestamp formats the rendered timestamp prefix.
	StyleTimestamp func(string) string
	// StyleDebug formats the debug level prefix.
	StyleDebug func(string) string
	// StyleInfo formats the info level prefix.
	StyleInfo func(string) string
	// StyleWarn formats the warning level prefix.
	StyleWarn func(string) string
	// StyleError formats the error level prefix.
	StyleError func(string) string
	// StyleSecurity formats the security event prefix.
	StyleSecurity func(string) string
}

// RotationOptions defines the log rotation and retention policy.
type RotationOptions struct {
	// Filename is the log file path. If empty, rotation is disabled.
	Filename string

	// MaxSize is the maximum size of the log file in megabytes before it gets rotated.
	// It defaults to 100 megabytes.
	MaxSize int

	// MaxAge is the maximum number of days to retain old log files based on their
	// file modification time. It defaults to 28 days.
	// Note: set to a negative value to disable age-based retention.
	MaxAge int

	// MaxBackups is the maximum number of old log files to retain.
	// It defaults to 5 backups.
	MaxBackups int

	// Compress determines if the rotated log files should be compressed using gzip.
	// It defaults to true.
	Compress bool
}

// Options configures a Logger.
type Options struct {
	// Level controls which messages are emitted.
	Level Level
	// Output is the destination for log messages. If Rotation is provided,
	// Output is ignored and logs are written to the rotating file instead.
	Output goio.Writer
	// Rotation enables log rotation to file. If provided, Filename must be set.
	Rotation *RotationOptions
	// RedactKeys is a list of keys whose values should be masked in logs.
	RedactKeys []string
}

// RotationWriterFactory creates a rotating writer from options.
// Set this to enable log rotation (provided by core/go-io integration).
var RotationWriterFactory func(RotationOptions) goio.WriteCloser

// New creates a new Logger with the given options.
//
//	logger := log.New(log.Options{
//	    Level:      log.LevelInfo,
//	    Output:     os.Stdout,
//	    RedactKeys: []string{"password", "token"},
//	})
func New(opts Options) *Logger {
	level := normaliseLevel(opts.Level)

	output := opts.Output
	if opts.Rotation != nil && opts.Rotation.Filename != "" && RotationWriterFactory != nil {
		output = RotationWriterFactory(normaliseRotationOptions(*opts.Rotation))
	}
	if output == nil {
		output = os.Stderr
	}

	return &Logger{
		level:          level,
		output:         output,
		redactKeys:     slices.Clone(opts.RedactKeys),
		StyleTimestamp: identity,
		StyleDebug:     identity,
		StyleInfo:      identity,
		StyleWarn:      identity,
		StyleError:     identity,
		StyleSecurity:  identity,
	}
}

func normaliseRotationOptions(opts RotationOptions) RotationOptions {
	if opts.MaxSize <= 0 {
		opts.MaxSize = defaultRotationMaxSize
	}
	if opts.MaxAge == 0 {
		opts.MaxAge = defaultRotationMaxAge
	}
	if opts.MaxBackups <= 0 {
		opts.MaxBackups = defaultRotationMaxBackups
	}
	return opts
}

func identity(s string) string { return s }

func safeStyle(style func(string) string) func(string) string {
	if style == nil {
		return identity
	}
	return style
}

// SetLevel changes the log level.
//
//	logger.SetLevel(log.LevelDebug)
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	l.level = normaliseLevel(level)
	l.mu.Unlock()
}

// Level returns the current log level.
//
//	current := logger.Level()
func (l *Logger) Level() Level {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.level
}

// SetOutput changes the output writer.
//
//	logger.SetOutput(os.Stdout)
func (l *Logger) SetOutput(w goio.Writer) {
	if w == nil {
		w = os.Stderr
	}
	l.mu.Lock()
	l.output = w
	l.mu.Unlock()
}

// SetRedactKeys sets the keys to be redacted.
//
//	logger.SetRedactKeys("password", "token", "secret")
func (l *Logger) SetRedactKeys(keys ...string) {
	l.mu.Lock()
	l.redactKeys = slices.Clone(keys)
	l.mu.Unlock()
}

func (l *Logger) shouldLog(level Level) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return level <= l.level
}

func (l *Logger) log(level Level, prefix, msg string, keyvals ...any) {
	_ = level
	l.mu.RLock()
	output := l.output
	styleTimestamp := l.StyleTimestamp
	redactKeys := l.redactKeys
	l.mu.RUnlock()

	if styleTimestamp == nil {
		styleTimestamp = identity
	}
	timestamp := styleTimestamp(time.Now().Format("15:04:05"))

	existing := make(map[string]struct{}, len(keyvals)/2+2)
	for i := 0; i < len(keyvals); i += 2 {
		if key, ok := keyvals[i].(string); ok {
			existing[key] = struct{}{}
		}
	}

	// Automatically extract context from error if present in keyvals
	origLen := len(keyvals)
	for i := 0; i < origLen; i += 2 {
		if i+1 >= origLen {
			continue
		}
		err, ok := keyvals[i+1].(error)
		if !ok {
			continue
		}

		if op := Op(err); op != "" {
			if _, hasOp := existing["op"]; !hasOp {
				existing["op"] = struct{}{}
				keyvals = append(keyvals, "op", op)
			}
		}
		if stack := FormatStackTrace(err); stack != "" {
			if _, hasStack := existing["stack"]; !hasStack {
				existing["stack"] = struct{}{}
				keyvals = append(keyvals, "stack", stack)
			}
		}
	}

	// Format key-value pairs
	var kvStr string
	if len(keyvals) > 0 {
		kvStr = " "
		for i := 0; i < len(keyvals); i += 2 {
			if i > 0 {
				kvStr += " "
			}
			key := normaliseLogText(fmt.Sprintf("%v", keyvals[i]))
			var val any
			if i+1 < len(keyvals) {
				val = keyvals[i+1]
			}

			// Redaction logic
			if shouldRedact(key, redactKeys) {
				val = "[REDACTED]"
			}

			// Secure formatting to prevent log injection
			if s, ok := val.(string); ok {
				kvStr += fmt.Sprintf("%s=%q", key, s)
			} else {
				kvStr += fmt.Sprintf("%s=%v", key, normaliseLogText(fmt.Sprintf("%v", val)))
			}
		}
	}

	_, _ = fmt.Fprintf(output, "%s %s %s%s\n", timestamp, prefix, normaliseLogText(msg), kvStr)
}

// Debug logs a debug message with optional key-value pairs.
//
//	logger.Debug("processing request", "method", "GET", "path", "/api/users")
func (l *Logger) Debug(msg string, keyvals ...any) {
	if l.shouldLog(LevelDebug) {
		l.mu.RLock()
		style := safeStyle(l.StyleDebug)
		l.mu.RUnlock()
		l.log(LevelDebug, style("[DBG]"), msg, keyvals...)
	}
}

// Info logs an info message with optional key-value pairs.
//
//	logger.Info("server started", "port", 8080)
func (l *Logger) Info(msg string, keyvals ...any) {
	if l.shouldLog(LevelInfo) {
		l.mu.RLock()
		style := safeStyle(l.StyleInfo)
		l.mu.RUnlock()
		l.log(LevelInfo, style("[INF]"), msg, keyvals...)
	}
}

// Warn logs a warning message with optional key-value pairs.
//
//	logger.Warn("high memory usage", "percent", 92)
func (l *Logger) Warn(msg string, keyvals ...any) {
	if l.shouldLog(LevelWarn) {
		l.mu.RLock()
		style := safeStyle(l.StyleWarn)
		l.mu.RUnlock()
		l.log(LevelWarn, style("[WRN]"), msg, keyvals...)
	}
}

// Error logs an error message with optional key-value pairs.
//
//	logger.Error("database connection failed", "err", err, "host", "db.local")
func (l *Logger) Error(msg string, keyvals ...any) {
	if l.shouldLog(LevelError) {
		l.mu.RLock()
		style := safeStyle(l.StyleError)
		l.mu.RUnlock()
		l.log(LevelError, style("[ERR]"), msg, keyvals...)
	}
}

// Security logs a security event with optional key-value pairs.
// It uses LevelError to ensure security events are visible even in restrictive
// log configurations.
//
//	logger.Security("brute force detected", "ip", remoteAddr, "attempts", 50)
func (l *Logger) Security(msg string, keyvals ...any) {
	if l.shouldLog(LevelError) {
		l.mu.RLock()
		style := safeStyle(l.StyleSecurity)
		l.mu.RUnlock()
		l.log(LevelError, style("[SEC]"), msg, keyvals...)
	}
}

// Username returns the current system username.
// It uses os/user for reliability and falls back to environment variables.
//
//	user := log.Username()
func Username() string {
	if u, err := user.Current(); err == nil {
		return u.Username
	}
	// Fallback for environments where user lookup might fail
	if u := os.Getenv("USER"); u != "" {
		return u
	}
	if u := os.Getenv("USERNAME"); u != "" {
		return u
	}
	return "unknown"
}

var logTextCleaner = strings.NewReplacer(
	"\r", "\\r",
	"\n", "\\n",
	"\t", "\\t",
)

func normaliseLogText(text string) string {
	return logTextCleaner.Replace(text)
}

// --- Default logger ---

var defaultLogger = New(Options{Level: LevelInfo})
var defaultLoggerMu sync.RWMutex

// Default returns the default logger.
//
//	logger := log.Default()
func Default() *Logger {
	defaultLoggerMu.RLock()
	defer defaultLoggerMu.RUnlock()
	return defaultLogger
}

// SetDefault sets the default logger.
// Passing nil is ignored to preserve the current default logger.
//
//	log.SetDefault(customLogger)
func SetDefault(l *Logger) {
	if l == nil {
		return
	}
	defaultLoggerMu.Lock()
	defaultLogger = l
	defaultLoggerMu.Unlock()
}

// SetLevel sets the default logger's level.
//
//	log.SetLevel(log.LevelDebug)
func SetLevel(level Level) {
	Default().SetLevel(level)
}

// SetRedactKeys sets the default logger's redaction keys.
//
//	log.SetRedactKeys("password", "token")
func SetRedactKeys(keys ...string) {
	Default().SetRedactKeys(keys...)
}

// Debug logs to the default logger.
//
//	log.Debug("query started", "sql", query)
func Debug(msg string, keyvals ...any) {
	Default().Debug(msg, keyvals...)
}

// Info logs to the default logger.
//
//	log.Info("server ready", "port", 8080)
func Info(msg string, keyvals ...any) {
	Default().Info(msg, keyvals...)
}

// Warn logs to the default logger.
//
//	log.Warn("retrying request", "attempt", 2)
func Warn(msg string, keyvals ...any) {
	Default().Warn(msg, keyvals...)
}

// Error logs to the default logger.
//
//	log.Error("request failed", "err", err)
func Error(msg string, keyvals ...any) {
	Default().Error(msg, keyvals...)
}

// Security logs to the default logger.
//
//	log.Security("suspicious login", "ip", remoteAddr)
func Security(msg string, keyvals ...any) {
	Default().Security(msg, keyvals...)
}

func shouldRedact(key any, redactKeys []string) bool {
	keyStr := fmt.Sprintf("%v", key)
	for _, redactKey := range redactKeys {
		if strings.EqualFold(redactKey, keyStr) {
			return true
		}
	}
	return false
}
