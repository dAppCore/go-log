package log

import (
	// Note: test-only stdlib (no core equivalent for io.Writer capture).
	"bytes"
	goio "io"
	// Note: test-only stdlib (no core equivalent for rendered output assertions).
	"strings"
	"testing"
	"time"
)

// nopWriteCloser wraps a writer with a no-op Close for testing rotation.
type nopWriteCloser struct{ goio.Writer }

func (nopWriteCloser) Close() error { return nil }

func TestLogger_Levels_Good(t *testing.T) {
	tests := []struct {
		name     string
		level    Level
		logFunc  func(*Logger, string, ...any)
		expected bool
	}{
		{"debug at debug", LevelDebug, (*Logger).Debug, true},
		{"info at debug", LevelDebug, (*Logger).Info, true},
		{"warn at debug", LevelDebug, (*Logger).Warn, true},
		{"error at debug", LevelDebug, (*Logger).Error, true},

		{"debug at info", LevelInfo, (*Logger).Debug, false},
		{"info at info", LevelInfo, (*Logger).Info, true},
		{"warn at info", LevelInfo, (*Logger).Warn, true},
		{"error at info", LevelInfo, (*Logger).Error, true},

		{"debug at warn", LevelWarn, (*Logger).Debug, false},
		{"info at warn", LevelWarn, (*Logger).Info, false},
		{"warn at warn", LevelWarn, (*Logger).Warn, true},
		{"error at warn", LevelWarn, (*Logger).Error, true},

		{"debug at error", LevelError, (*Logger).Debug, false},
		{"info at error", LevelError, (*Logger).Info, false},
		{"warn at error", LevelError, (*Logger).Warn, false},
		{"error at error", LevelError, (*Logger).Error, true},

		{"debug at quiet", LevelQuiet, (*Logger).Debug, false},
		{"info at quiet", LevelQuiet, (*Logger).Info, false},
		{"warn at quiet", LevelQuiet, (*Logger).Warn, false},
		{"error at quiet", LevelQuiet, (*Logger).Error, false},

		{"security at info", LevelInfo, (*Logger).Security, true},
		{"security at error", LevelError, (*Logger).Security, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			l := New(Options{Level: tt.level, Output: &buf})
			tt.logFunc(l, "test message")

			hasOutput := buf.Len() > 0
			if hasOutput != tt.expected {
				t.Errorf("expected output=%v, got output=%v", tt.expected, hasOutput)
			}
		})
	}
}

func TestLogger_KeyValues_Good(t *testing.T) {
	var buf bytes.Buffer
	l := New(Options{Level: LevelDebug, Output: &buf})

	l.Info("test message", "key1", "value1", "key2", 42)

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Error("expected message in output")
	}
	if !strings.Contains(output, "key1=\"value1\"") {
		t.Errorf("expected key1=\"value1\" in output, got %q", output)
	}
	if !strings.Contains(output, "key2=42") {
		t.Error("expected key2=42 in output")
	}
}

func TestLogger_ErrorContext_Good(t *testing.T) {
	var buf bytes.Buffer
	l := New(Options{Output: &buf, Level: LevelInfo})

	err := E("test.Op", "failed", NewError("root cause"))
	err = Wrap(err, "outer.Op", "outer failed")

	l.Error("something failed", "err", err)

	got := buf.String()
	if !strings.Contains(got, "op=\"outer.Op\"") {
		t.Errorf("expected output to contain op=\"outer.Op\", got %q", got)
	}
	if !strings.Contains(got, "stack=\"outer.Op -> test.Op\"") {
		t.Errorf("expected output to contain stack=\"outer.Op -> test.Op\", got %q", got)
	}
}

func TestLogger_ErrorContextIncludesRecovery_Good(t *testing.T) {
	var buf bytes.Buffer
	l := New(Options{Output: &buf, Level: LevelInfo})
	retryAfter := 45 * time.Second

	err := EWithRecovery("retryable.Op", "temporary failure", NewError("temporary failure"), true, &retryAfter, "retry with backoff")
	l.Error("request failed", "err", err)

	output := buf.String()
	if !strings.Contains(output, "retryable=true") {
		t.Errorf("expected output to contain retryable=true, got %q", output)
	}
	if !strings.Contains(output, "retry_after_seconds=45") {
		t.Errorf("expected output to contain retry_after_seconds=45, got %q", output)
	}
	if !strings.Contains(output, "next_action=\"retry with backoff\"") {
		t.Errorf("expected output to contain next_action=\"retry with backoff\", got %q", output)
	}
}

func TestLogger_ErrorContextIncludesNestedRecovery_Good(t *testing.T) {
	var buf bytes.Buffer
	l := New(Options{Output: &buf, Level: LevelInfo})
	retryAfter := 30 * time.Second

	inner := &Err{
		Msg:        "inner failure",
		Retryable:  true,
		RetryAfter: &retryAfter,
		NextAction: "retry later",
	}
	outer := &Err{Msg: "outer failure", Err: inner}

	l.Error("request failed", "err", outer)

	output := buf.String()
	if !strings.Contains(output, "retryable=true") {
		t.Errorf("expected output to contain retryable=true, got %q", output)
	}
	if !strings.Contains(output, "retry_after_seconds=30") {
		t.Errorf("expected output to contain retry_after_seconds=30, got %q", output)
	}
	if !strings.Contains(output, "next_action=\"retry later\"") {
		t.Errorf("expected output to contain next_action=\"retry later\", got %q", output)
	}
}

func TestLogger_Redaction_Good(t *testing.T) {
	var buf bytes.Buffer
	l := New(Options{
		Level:      LevelInfo,
		Output:     &buf,
		RedactKeys: []string{"password", "token"},
	})

	l.Info("login", "user", "admin", "password", "secret123", "token", "abc-123")

	output := buf.String()
	if !strings.Contains(output, "user=\"admin\"") {
		t.Error("expected user=\"admin\"")
	}
	if !strings.Contains(output, "password=\"[REDACTED]\"") {
		t.Errorf("expected password=\"[REDACTED]\", got %q", output)
	}
	if !strings.Contains(output, "token=\"[REDACTED]\"") {
		t.Errorf("expected token=\"[REDACTED]\", got %q", output)
	}
}

func TestLogger_Redaction_Bad_CaseMismatchNotRedacted(t *testing.T) {
	var buf bytes.Buffer
	l := New(Options{
		Level:      LevelInfo,
		Output:     &buf,
		RedactKeys: []string{"password"},
	})

	l.Info("login", "PASSWORD", "secret123")

	output := buf.String()
	if !strings.Contains(output, "PASSWORD=\"secret123\"") {
		t.Errorf("expected case-mismatched key to remain visible, got %q", output)
	}
}

func TestLogger_InjectionPrevention_Good(t *testing.T) {
	var buf bytes.Buffer
	l := New(Options{Level: LevelInfo, Output: &buf})

	l.Info("message", "key", "value\n[SEC] injected message")

	output := buf.String()
	if !strings.Contains(output, "key=\"value\\n[SEC] injected message\"") {
		t.Errorf("expected escaped newline, got %q", output)
	}
	// Ensure it's still a single line (excluding trailing newline)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 line, got %d", len(lines))
	}
}

func TestLogger_KeySanitization_Good(t *testing.T) {
	var buf bytes.Buffer
	l := New(Options{Level: LevelInfo, Output: &buf})

	l.Info("message", "key\nwith newline", "value\nwith newline")
	output := buf.String()

	if !strings.Contains(output, "key\\nwith newline") {
		t.Errorf("expected sanitized key, got %q", output)
	}
	if !strings.Contains(output, "value\\nwith newline") {
		t.Errorf("expected sanitized value, got %q", output)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 line, got %d", len(lines))
	}
}

func TestLogger_MessageSanitization_Good(t *testing.T) {
	var buf bytes.Buffer
	l := New(Options{Level: LevelInfo, Output: &buf})

	l.Info("message\nwith\tcontrol\rchars")
	output := buf.String()

	if !strings.Contains(output, "message\\nwith\\tcontrol\\rchars") {
		t.Errorf("expected control characters to be escaped, got %q", output)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 line, got %d", len(lines))
	}
}

func TestLogger_SetLevel_Good(t *testing.T) {
	l := New(Options{Level: LevelInfo})

	if l.Level() != LevelInfo {
		t.Error("expected initial level to be Info")
	}

	l.SetLevel(LevelDebug)
	if l.Level() != LevelDebug {
		t.Error("expected level to be Debug after SetLevel")
	}

	l.SetLevel(99)
	if l.Level() != LevelInfo {
		t.Errorf("expected invalid level to default back to info, got %v", l.Level())
	}
}

func TestLevel_String_Good(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{LevelQuiet, "quiet"},
		{LevelError, "error"},
		{LevelWarn, "warn"},
		{LevelInfo, "info"},
		{LevelDebug, "debug"},
		{Level(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.level.String(); got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestLogger_Security_Good(t *testing.T) {
	var buf bytes.Buffer
	l := New(Options{Level: LevelError, Output: &buf})

	l.Security("unauthorized access", "user", "admin")

	output := buf.String()
	if !strings.Contains(output, "[SEC]") {
		t.Error("expected [SEC] prefix in security log")
	}
	if !strings.Contains(output, "unauthorized access") {
		t.Error("expected message in security log")
	}
	if !strings.Contains(output, "user=\"admin\"") {
		t.Error("expected context in security log")
	}
}

func TestLogger_SetOutput_Good(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	l := New(Options{Level: LevelInfo, Output: &buf1})

	l.Info("first")
	if buf1.Len() == 0 {
		t.Error("expected output in first buffer")
	}

	l.SetOutput(&buf2)
	l.Info("second")
	if buf2.Len() == 0 {
		t.Error("expected output in second buffer after SetOutput")
	}
}

func TestLogger_SetOutput_Bad_NilUsesFallback(t *testing.T) {
	var buf bytes.Buffer
	l := New(Options{Level: LevelInfo, Output: &buf})

	l.SetOutput(nil)

	if l.output == nil {
		t.Error("expected nil output to install a fallback writer")
	}
	if l.output == &buf {
		t.Error("expected nil output to replace the previous writer")
	}
}

func TestLogger_SetRedactKeys_Good(t *testing.T) {
	var buf bytes.Buffer
	l := New(Options{Level: LevelInfo, Output: &buf})

	// No redaction initially
	l.Info("msg", "secret", "visible")
	if !strings.Contains(buf.String(), "secret=\"visible\"") {
		t.Errorf("expected visible value, got %q", buf.String())
	}

	buf.Reset()
	l.SetRedactKeys("secret")
	l.Info("msg", "secret", "hidden")
	if !strings.Contains(buf.String(), "secret=\"[REDACTED]\"") {
		t.Errorf("expected redacted value, got %q", buf.String())
	}
}

func TestLogger_OddKeyvals_Good(t *testing.T) {
	var buf bytes.Buffer
	l := New(Options{Level: LevelInfo, Output: &buf})

	// Odd number of keyvals — last key should have no value
	l.Info("msg", "lonely_key")
	output := buf.String()
	if !strings.Contains(output, "lonely_key=<nil>") {
		t.Errorf("expected lonely_key=<nil>, got %q", output)
	}
}

func TestLogger_ExistingOpNotDuplicated_Good(t *testing.T) {
	var buf bytes.Buffer
	l := New(Options{Level: LevelInfo, Output: &buf})

	err := E("inner.Op", "failed", NewError("cause"))
	// Pass op explicitly — should not duplicate
	l.Error("failed", "op", "explicit.Op", "err", err)

	output := buf.String()
	if strings.Count(output, "op=") != 1 {
		t.Errorf("expected exactly one op= in output, got %q", output)
	}
	if !strings.Contains(output, "op=\"explicit.Op\"") {
		t.Errorf("expected explicit op, got %q", output)
	}
}

func TestLogger_ExistingStackNotDuplicated_Good(t *testing.T) {
	var buf bytes.Buffer
	l := New(Options{Level: LevelInfo, Output: &buf})

	err := E("inner.Op", "failed", NewError("cause"))
	// Pass stack explicitly — should not duplicate
	l.Error("failed", "stack", "custom.Stack", "err", err)

	output := buf.String()
	if strings.Count(output, "stack=") != 1 {
		t.Errorf("expected exactly one stack= in output, got %q", output)
	}
	if !strings.Contains(output, "stack=\"custom.Stack\"") {
		t.Errorf("expected custom stack, got %q", output)
	}
}

func TestNew_RotationFactory_Good(t *testing.T) {
	var buf bytes.Buffer
	// Set up a mock rotation writer factory
	original := RotationWriterFactory
	defer func() { RotationWriterFactory = original }()

	RotationWriterFactory = func(opts RotationOptions) goio.WriteCloser {
		return nopWriteCloser{&buf}
	}

	l := New(Options{
		Level:    LevelInfo,
		Rotation: &RotationOptions{Filename: "test.log"},
	})

	l.Info("rotated message")
	if buf.Len() == 0 {
		t.Error("expected output via rotation writer")
	}
}

func TestNew_RotationFactory_Good_DefaultRetentionValues(t *testing.T) {
	original := RotationWriterFactory
	defer func() { RotationWriterFactory = original }()

	var captured RotationOptions
	RotationWriterFactory = func(opts RotationOptions) goio.WriteCloser {
		captured = opts
		return nopWriteCloser{goio.Discard}
	}

	_ = New(Options{
		Level:    LevelInfo,
		Rotation: &RotationOptions{Filename: "test.log"},
	})

	if captured.MaxSize != defaultRotationMaxSize {
		t.Errorf("expected default MaxSize=%d, got %d", defaultRotationMaxSize, captured.MaxSize)
	}
	if captured.MaxAge != defaultRotationMaxAge {
		t.Errorf("expected default MaxAge=%d, got %d", defaultRotationMaxAge, captured.MaxAge)
	}
	if captured.MaxBackups != defaultRotationMaxBackups {
		t.Errorf("expected default MaxBackups=%d, got %d", defaultRotationMaxBackups, captured.MaxBackups)
	}
}

func TestNew_DefaultOutput_Good(t *testing.T) {
	// No output or rotation — should default to stderr (not nil)
	l := New(Options{Level: LevelInfo})
	if l.output == nil {
		t.Error("expected non-nil output when no Output specified")
	}
}

func TestNew_Bad_InvalidLevelDefaultsToInfo(t *testing.T) {
	l := New(Options{Level: Level(99)})
	if l.Level() != LevelInfo {
		t.Errorf("expected invalid level to default to info, got %v", l.Level())
	}
}

func TestUsername_Good(t *testing.T) {
	name := Username()
	if name == "" {
		t.Error("expected Username to return a non-empty string")
	}
}

func TestDefault_Good(t *testing.T) {
	if Default() == nil {
		t.Error("expected default logger to exist")
	}

	// All package-level proxy functions
	var buf bytes.Buffer
	l := New(Options{Level: LevelDebug, Output: &buf})
	SetDefault(l)
	defer SetDefault(New(Options{Level: LevelInfo}))

	SetLevel(LevelDebug)
	if l.Level() != LevelDebug {
		t.Error("expected package-level SetLevel to work")
	}

	SetRedactKeys("secret")

	Debug("debug msg")
	Info("info msg")
	Warn("warn msg")
	Error("error msg")
	Security("sec msg")

	output := buf.String()
	for _, tag := range []string{"[DBG]", "[INF]", "[WRN]", "[ERR]", "[SEC]"} {
		if !strings.Contains(output, tag) {
			t.Errorf("expected %s in output, got %q", tag, output)
		}
	}
}

func TestDefault_Bad_SetDefaultNilIgnored(t *testing.T) {
	original := Default()
	var buf bytes.Buffer
	custom := New(Options{Level: LevelInfo, Output: &buf})
	SetDefault(custom)
	defer SetDefault(original)

	SetDefault(nil)

	if Default() != custom {
		t.Error("expected SetDefault(nil) to preserve the current default logger")
	}
}

func TestLogger_StyleHooks_Bad_NilHooksDoNotPanic(t *testing.T) {
	var buf bytes.Buffer
	l := New(Options{Level: LevelDebug, Output: &buf})
	l.StyleTimestamp = nil
	l.StyleDebug = nil
	l.StyleInfo = nil
	l.StyleWarn = nil
	l.StyleError = nil
	l.StyleSecurity = nil

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("expected nil style hooks not to panic, got panic: %v", r)
		}
	}()

	l.Debug("debug")
	l.Info("info")
	l.Warn("warn")
	l.Error("error")
	l.Security("security")

	output := buf.String()
	for _, tag := range []string{"[DBG]", "[INF]", "[WRN]", "[ERR]", "[SEC]"} {
		if !strings.Contains(output, tag) {
			t.Errorf("expected %s in output, got %q", tag, output)
		}
	}
}
