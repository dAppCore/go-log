package log

import (
	"bytes"
	"fmt"
	"io"
	"slices"
	"strings"
	"testing"
	"time"
)

type ax7RotationWriter struct {
	io.Writer
	closed bool
}

func (w *ax7RotationWriter) Close() error {
	w.closed = true
	return nil
}

func ax7DefaultBuffer(t *testing.T) *bytes.Buffer {
	t.Helper()
	original := Default()
	buf := &bytes.Buffer{}
	SetDefault(New(Options{Level: LevelDebug, Output: buf}))
	t.Cleanup(func() { SetDefault(original) })
	return buf
}

func TestAX7_Username_Good(t *testing.T) {
	t.Setenv("USER", "codex")
	got := Username()
	if got != "codex" {
		t.Fatalf("want codex, got %q", got)
	}
}

func TestAX7_Username_Bad(t *testing.T) {
	t.Setenv("USER", "")
	t.Setenv("USERNAME", "fallback")
	got := Username()
	if got != "fallback" {
		t.Fatalf("want fallback, got %q", got)
	}
}

func TestAX7_Username_Ugly(t *testing.T) {
	t.Setenv("USER", "")
	t.Setenv("USERNAME", "")
	got := Username()
	if got != "unknown" {
		t.Fatalf("want unknown, got %q", got)
	}
}

func TestAX7_Level_String_Good(t *testing.T) {
	got := LevelInfo.String()
	want := "info"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestAX7_Level_String_Bad(t *testing.T) {
	got := Level(99).String()
	want := "unknown"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestAX7_Level_String_Ugly(t *testing.T) {
	got := Level(-1).String()
	want := "unknown"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestAX7_New_Good(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(Options{Level: LevelDebug, Output: buf})
	logger.Debug("ready")
	if !strings.Contains(buf.String(), "[DBG] ready") {
		t.Fatalf("expected debug output, got %q", buf.String())
	}
}

func TestAX7_New_Bad(t *testing.T) {
	logger := New(Options{Level: Level(99), Output: io.Discard})
	got := logger.Level()
	if got != LevelInfo {
		t.Fatalf("want info for invalid level, got %v", got)
	}
}

func TestAX7_New_Ugly(t *testing.T) {
	keys := []string{"secret"}
	logger := New(Options{Level: LevelInfo, Output: io.Discard, RedactKeys: keys})
	keys[0] = "mutated"
	if logger.redactKeys[0] != "secret" {
		t.Fatalf("redaction keys must be cloned, got %v", logger.redactKeys)
	}
}

func TestAX7_Logger_SetLevel_Good(t *testing.T) {
	logger := New(Options{Level: LevelInfo, Output: io.Discard})
	logger.SetLevel(LevelDebug)
	got := logger.Level()
	if got != LevelDebug {
		t.Fatalf("want debug, got %v", got)
	}
}

func TestAX7_Logger_SetLevel_Bad(t *testing.T) {
	logger := New(Options{Level: LevelDebug, Output: io.Discard})
	logger.SetLevel(Level(99))
	got := logger.Level()
	if got != LevelInfo {
		t.Fatalf("want invalid level to normalise to info, got %v", got)
	}
}

func TestAX7_Logger_SetLevel_Ugly(t *testing.T) {
	logger := New(Options{Level: LevelDebug, Output: io.Discard})
	logger.SetLevel(LevelQuiet)
	got := logger.Level()
	if got != LevelQuiet {
		t.Fatalf("want quiet, got %v", got)
	}
}

func TestAX7_Logger_Level_Good(t *testing.T) {
	logger := New(Options{Level: LevelWarn, Output: io.Discard})
	got := logger.Level()
	if got != LevelWarn {
		t.Fatalf("want warn, got %v", got)
	}
}

func TestAX7_Logger_Level_Bad(t *testing.T) {
	logger := New(Options{Level: Level(-5), Output: io.Discard})
	got := logger.Level()
	if got != LevelInfo {
		t.Fatalf("want invalid level to read as info, got %v", got)
	}
}

func TestAX7_Logger_Level_Ugly(t *testing.T) {
	logger := New(Options{Level: LevelQuiet, Output: io.Discard})
	got := logger.Level()
	if got != LevelQuiet {
		t.Fatalf("want quiet, got %v", got)
	}
}

func TestAX7_Logger_SetOutput_Good(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(Options{Level: LevelInfo, Output: io.Discard})
	logger.SetOutput(buf)
	logger.Info("switched")
	if !strings.Contains(buf.String(), "switched") {
		t.Fatalf("expected output in replacement writer, got %q", buf.String())
	}
}

func TestAX7_Logger_SetOutput_Bad(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(Options{Level: LevelInfo, Output: buf})
	logger.SetOutput(nil)
	if logger.output == nil || logger.output == buf {
		t.Fatalf("nil output should install fallback writer, got %#v", logger.output)
	}
}

func TestAX7_Logger_SetOutput_Ugly(t *testing.T) {
	first := &bytes.Buffer{}
	second := &bytes.Buffer{}
	logger := New(Options{Level: LevelInfo, Output: first})
	logger.SetOutput(second)
	logger.Info("after")
	if first.Len() != 0 || !strings.Contains(second.String(), "after") {
		t.Fatalf("expected only second writer to receive output, first=%q second=%q", first.String(), second.String())
	}
}

func TestAX7_Logger_SetRedactKeys_Good(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(Options{Level: LevelInfo, Output: buf})
	logger.SetRedactKeys("secret")
	logger.Info("login", "secret", "token")
	if !strings.Contains(buf.String(), `secret="[REDACTED]"`) {
		t.Fatalf("expected redaction, got %q", buf.String())
	}
}

func TestAX7_Logger_SetRedactKeys_Bad(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(Options{Level: LevelInfo, Output: buf, RedactKeys: []string{"secret"}})
	logger.SetRedactKeys()
	logger.Info("login", "secret", "visible")
	if !strings.Contains(buf.String(), `secret="visible"`) {
		t.Fatalf("expected cleared redaction keys, got %q", buf.String())
	}
}

func TestAX7_Logger_SetRedactKeys_Ugly(t *testing.T) {
	keys := []string{"secret"}
	logger := New(Options{Level: LevelInfo, Output: io.Discard})
	logger.SetRedactKeys(keys...)
	keys[0] = "mutated"
	if logger.redactKeys[0] != "secret" {
		t.Fatalf("SetRedactKeys must clone input, got %v", logger.redactKeys)
	}
}

func TestAX7_Logger_Debug_Good(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(Options{Level: LevelDebug, Output: buf})
	logger.Debug("debug", "agent", "codex")
	if !strings.Contains(buf.String(), `[DBG] debug agent="codex"`) {
		t.Fatalf("expected debug output, got %q", buf.String())
	}
}

func TestAX7_Logger_Debug_Bad(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(Options{Level: LevelInfo, Output: buf})
	logger.Debug("hidden")
	if buf.Len() != 0 {
		t.Fatalf("debug should be suppressed at info level, got %q", buf.String())
	}
}

func TestAX7_Logger_Debug_Ugly(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(Options{Level: LevelDebug, Output: buf})
	logger.StyleDebug = nil
	logger.Debug("nil-style")
	if !strings.Contains(buf.String(), "[DBG] nil-style") {
		t.Fatalf("expected fallback debug style, got %q", buf.String())
	}
}

func TestAX7_Logger_Info_Good(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(Options{Level: LevelInfo, Output: buf})
	logger.Info("info", "ok", true)
	if !strings.Contains(buf.String(), "[INF] info ok=true") {
		t.Fatalf("expected info output, got %q", buf.String())
	}
}

func TestAX7_Logger_Info_Bad(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(Options{Level: LevelWarn, Output: buf})
	logger.Info("hidden")
	if buf.Len() != 0 {
		t.Fatalf("info should be suppressed at warn level, got %q", buf.String())
	}
}

func TestAX7_Logger_Info_Ugly(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(Options{Level: LevelInfo, Output: buf})
	logger.Info("line\nbreak", "key\tname", "value\r")
	if !strings.Contains(buf.String(), `line\nbreak key\tname="value\r"`) {
		t.Fatalf("expected escaped info output, got %q", buf.String())
	}
}

func TestAX7_Logger_Warn_Good(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(Options{Level: LevelWarn, Output: buf})
	logger.Warn("warn", "attempt", 2)
	if !strings.Contains(buf.String(), "[WRN] warn attempt=2") {
		t.Fatalf("expected warn output, got %q", buf.String())
	}
}

func TestAX7_Logger_Warn_Bad(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(Options{Level: LevelError, Output: buf})
	logger.Warn("hidden")
	if buf.Len() != 0 {
		t.Fatalf("warn should be suppressed at error level, got %q", buf.String())
	}
}

func TestAX7_Logger_Warn_Ugly(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(Options{Level: LevelWarn, Output: buf})
	logger.StyleWarn = func(s string) string { return "WARN:" + s }
	logger.Warn("styled")
	if !strings.Contains(buf.String(), "WARN:[WRN] styled") {
		t.Fatalf("expected styled warn output, got %q", buf.String())
	}
}

func TestAX7_Logger_Error_Good(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(Options{Level: LevelError, Output: buf})
	logger.Error("failed", "err", NewError("boom"))
	if !strings.Contains(buf.String(), "[ERR] failed err=boom") {
		t.Fatalf("expected error output, got %q", buf.String())
	}
}

func TestAX7_Logger_Error_Bad(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(Options{Level: LevelQuiet, Output: buf})
	logger.Error("hidden")
	if buf.Len() != 0 {
		t.Fatalf("error should be suppressed at quiet level, got %q", buf.String())
	}
}

func TestAX7_Logger_Error_Ugly(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(Options{Level: LevelError, Output: buf})
	err := Wrap(E("inner.Op", "inner failed", NewError("root")), "outer.Op", "outer failed")
	logger.Error("failed", "err", err)
	if !strings.Contains(buf.String(), `stack="outer.Op -> inner.Op"`) {
		t.Fatalf("expected stack context, got %q", buf.String())
	}
}

func TestAX7_Logger_Security_Good(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(Options{Level: LevelError, Output: buf})
	logger.Security("login", "user", "alice")
	if !strings.Contains(buf.String(), `[SEC] login user="alice"`) {
		t.Fatalf("expected security output, got %q", buf.String())
	}
}

func TestAX7_Logger_Security_Bad(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(Options{Level: LevelQuiet, Output: buf})
	logger.Security("hidden")
	if buf.Len() != 0 {
		t.Fatalf("security should be suppressed at quiet level, got %q", buf.String())
	}
}

func TestAX7_Logger_Security_Ugly(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(Options{Level: LevelError, Output: buf})
	logger.StyleSecurity = func(s string) string { return "SEC:" + s }
	logger.Security("styled")
	if !strings.Contains(buf.String(), "SEC:[SEC] styled") {
		t.Fatalf("expected styled security output, got %q", buf.String())
	}
}

func TestAX7_Default_Good(t *testing.T) {
	got := Default()
	if got == nil {
		t.Fatal("expected default logger")
	}
	if got.Level() < LevelQuiet || got.Level() > LevelDebug {
		t.Fatalf("default logger level out of range: %v", got.Level())
	}
}

func TestAX7_Default_Bad(t *testing.T) {
	original := Default()
	SetDefault(nil)
	got := Default()
	if got != original {
		t.Fatal("SetDefault(nil) must preserve the existing default logger")
	}
}

func TestAX7_Default_Ugly(t *testing.T) {
	original := Default()
	custom := New(Options{Level: LevelDebug, Output: io.Discard})
	SetDefault(custom)
	t.Cleanup(func() { SetDefault(original) })
	if Default() != custom {
		t.Fatal("expected custom default logger")
	}
}

func TestAX7_SetDefault_Good(t *testing.T) {
	original := Default()
	custom := New(Options{Level: LevelWarn, Output: io.Discard})
	SetDefault(custom)
	t.Cleanup(func() { SetDefault(original) })
	if Default() != custom {
		t.Fatal("expected SetDefault to install custom logger")
	}
}

func TestAX7_SetDefault_Bad(t *testing.T) {
	original := Default()
	SetDefault(nil)
	got := Default()
	if got != original {
		t.Fatal("nil default logger should be ignored")
	}
}

func TestAX7_SetDefault_Ugly(t *testing.T) {
	original := Default()
	first := New(Options{Level: LevelInfo, Output: io.Discard})
	second := New(Options{Level: LevelDebug, Output: io.Discard})
	SetDefault(first)
	SetDefault(second)
	t.Cleanup(func() { SetDefault(original) })
	if Default() != second {
		t.Fatal("latest non-nil default logger should win")
	}
}

func TestAX7_SetLevel_Good(t *testing.T) {
	original := Default()
	logger := New(Options{Level: LevelInfo, Output: io.Discard})
	SetDefault(logger)
	t.Cleanup(func() { SetDefault(original) })
	SetLevel(LevelDebug)
	if logger.Level() != LevelDebug {
		t.Fatalf("want debug, got %v", logger.Level())
	}
}

func TestAX7_SetLevel_Bad(t *testing.T) {
	original := Default()
	logger := New(Options{Level: LevelDebug, Output: io.Discard})
	SetDefault(logger)
	t.Cleanup(func() { SetDefault(original) })
	SetLevel(Level(99))
	if logger.Level() != LevelInfo {
		t.Fatalf("invalid default level should normalise to info, got %v", logger.Level())
	}
}

func TestAX7_SetLevel_Ugly(t *testing.T) {
	original := Default()
	logger := New(Options{Level: LevelDebug, Output: io.Discard})
	SetDefault(logger)
	t.Cleanup(func() { SetDefault(original) })
	SetLevel(LevelQuiet)
	if logger.Level() != LevelQuiet {
		t.Fatalf("want quiet, got %v", logger.Level())
	}
}

func TestAX7_SetRedactKeys_Good(t *testing.T) {
	buf := ax7DefaultBuffer(t)
	SetRedactKeys("token")
	Info("login", "token", "abc")
	if !strings.Contains(buf.String(), `token="[REDACTED]"`) {
		t.Fatalf("expected package redaction, got %q", buf.String())
	}
}

func TestAX7_SetRedactKeys_Bad(t *testing.T) {
	buf := ax7DefaultBuffer(t)
	SetRedactKeys("token")
	SetRedactKeys()
	Info("login", "token", "abc")
	if !strings.Contains(buf.String(), `token="abc"`) {
		t.Fatalf("expected redaction keys to clear, got %q", buf.String())
	}
}

func TestAX7_SetRedactKeys_Ugly(t *testing.T) {
	buf := ax7DefaultBuffer(t)
	SetRedactKeys("")
	Info("empty-key", "", "secret")
	if !strings.Contains(buf.String(), `="[REDACTED]"`) {
		t.Fatalf("expected empty key redaction, got %q", buf.String())
	}
}

func TestAX7_Debug_Good(t *testing.T) {
	buf := ax7DefaultBuffer(t)
	Debug("debug", "agent", "codex")
	if !strings.Contains(buf.String(), `[DBG] debug agent="codex"`) {
		t.Fatalf("expected package debug output, got %q", buf.String())
	}
}

func TestAX7_Debug_Bad(t *testing.T) {
	buf := &bytes.Buffer{}
	original := Default()
	SetDefault(New(Options{Level: LevelInfo, Output: buf}))
	t.Cleanup(func() { SetDefault(original) })
	Debug("hidden")
	if buf.Len() != 0 {
		t.Fatalf("debug should be suppressed by default info level, got %q", buf.String())
	}
}

func TestAX7_Debug_Ugly(t *testing.T) {
	buf := ax7DefaultBuffer(t)
	Debug("control\nchars", "key", "value\t")
	if !strings.Contains(buf.String(), `control\nchars key="value\t"`) {
		t.Fatalf("expected escaped package debug output, got %q", buf.String())
	}
}

func TestAX7_Info_Good(t *testing.T) {
	buf := ax7DefaultBuffer(t)
	Info("info", "ok", true)
	if !strings.Contains(buf.String(), "[INF] info ok=true") {
		t.Fatalf("expected package info output, got %q", buf.String())
	}
}

func TestAX7_Info_Bad(t *testing.T) {
	buf := &bytes.Buffer{}
	original := Default()
	SetDefault(New(Options{Level: LevelWarn, Output: buf}))
	t.Cleanup(func() { SetDefault(original) })
	Info("hidden")
	if buf.Len() != 0 {
		t.Fatalf("info should be suppressed at warn level, got %q", buf.String())
	}
}

func TestAX7_Info_Ugly(t *testing.T) {
	buf := ax7DefaultBuffer(t)
	Info("odd", "lonely")
	if !strings.Contains(buf.String(), "lonely=<nil>") {
		t.Fatalf("expected odd keyvals to render nil value, got %q", buf.String())
	}
}

func TestAX7_Warn_Good(t *testing.T) {
	buf := ax7DefaultBuffer(t)
	Warn("warn", "attempt", 2)
	if !strings.Contains(buf.String(), "[WRN] warn attempt=2") {
		t.Fatalf("expected package warn output, got %q", buf.String())
	}
}

func TestAX7_Warn_Bad(t *testing.T) {
	buf := &bytes.Buffer{}
	original := Default()
	SetDefault(New(Options{Level: LevelError, Output: buf}))
	t.Cleanup(func() { SetDefault(original) })
	Warn("hidden")
	if buf.Len() != 0 {
		t.Fatalf("warn should be suppressed at error level, got %q", buf.String())
	}
}

func TestAX7_Warn_Ugly(t *testing.T) {
	buf := ax7DefaultBuffer(t)
	Warn("newline", "value", "a\nb")
	if !strings.Contains(buf.String(), `value="a\nb"`) {
		t.Fatalf("expected escaped package warn output, got %q", buf.String())
	}
}

func TestAX7_Error_Good(t *testing.T) {
	buf := ax7DefaultBuffer(t)
	Error("failed", "err", NewError("boom"))
	if !strings.Contains(buf.String(), "[ERR] failed err=boom") {
		t.Fatalf("expected package error output, got %q", buf.String())
	}
}

func TestAX7_Error_Bad(t *testing.T) {
	buf := &bytes.Buffer{}
	original := Default()
	SetDefault(New(Options{Level: LevelQuiet, Output: buf}))
	t.Cleanup(func() { SetDefault(original) })
	Error("hidden")
	if buf.Len() != 0 {
		t.Fatalf("error should be suppressed at quiet level, got %q", buf.String())
	}
}

func TestAX7_Error_Ugly(t *testing.T) {
	buf := ax7DefaultBuffer(t)
	err := E("op.Name", "failed", NewError("root"))
	Error("failed", "err", err)
	if !strings.Contains(buf.String(), `op="op.Name"`) {
		t.Fatalf("expected operation context, got %q", buf.String())
	}
}

func TestAX7_Security_Good(t *testing.T) {
	buf := ax7DefaultBuffer(t)
	Security("login", "user", "alice")
	if !strings.Contains(buf.String(), `[SEC] login user="alice"`) {
		t.Fatalf("expected package security output, got %q", buf.String())
	}
}

func TestAX7_Security_Bad(t *testing.T) {
	buf := &bytes.Buffer{}
	original := Default()
	SetDefault(New(Options{Level: LevelQuiet, Output: buf}))
	t.Cleanup(func() { SetDefault(original) })
	Security("hidden")
	if buf.Len() != 0 {
		t.Fatalf("security should be suppressed at quiet level, got %q", buf.String())
	}
}

func TestAX7_Security_Ugly(t *testing.T) {
	buf := ax7DefaultBuffer(t)
	Security("line\nbreak", "ip", "127.0.0.1")
	if !strings.Contains(buf.String(), `[SEC] line\nbreak ip="127.0.0.1"`) {
		t.Fatalf("expected escaped package security output, got %q", buf.String())
	}
}

func TestAX7_Err_Error_Good(t *testing.T) {
	err := &Err{Op: "agent.Dispatch", Msg: "failed", Err: NewError("root"), Code: "agent.failed"}
	got := err.Error()
	if got != "agent.Dispatch: failed [agent.failed]: root" {
		t.Fatalf("unexpected error string: %q", got)
	}
}

func TestAX7_Err_Error_Bad(t *testing.T) {
	err := &Err{}
	got := err.Error()
	if got != "" {
		t.Fatalf("empty Err should render empty string, got %q", got)
	}
}

func TestAX7_Err_Error_Ugly(t *testing.T) {
	var err *Err
	got := err.Error()
	if got != "" {
		t.Fatalf("nil Err should render empty string, got %q", got)
	}
}

func TestAX7_Err_Unwrap_Good(t *testing.T) {
	root := NewError("root")
	err := &Err{Err: root}
	got := err.Unwrap()
	if got != root {
		t.Fatalf("want root, got %v", got)
	}
}

func TestAX7_Err_Unwrap_Bad(t *testing.T) {
	err := &Err{}
	got := err.Unwrap()
	if got != nil {
		t.Fatalf("empty Err should unwrap to nil, got %v", got)
	}
}

func TestAX7_Err_Unwrap_Ugly(t *testing.T) {
	root := NewError("root")
	err := &Err{Err: Wrap(root, "outer", "failed")}
	if !Is(err.Unwrap(), root) {
		t.Fatal("unwrap should preserve wrapped root")
	}
}

func TestAX7_E_Good(t *testing.T) {
	root := NewError("root")
	err := E("agent.Dispatch", "failed", root)
	if !Is(err, root) || Op(err) != "agent.Dispatch" || Message(err) != "failed" {
		t.Fatalf("unexpected structured error: %v", err)
	}
}

func TestAX7_E_Bad(t *testing.T) {
	err := E("", "", nil)
	got := err.Error()
	if got != "" {
		t.Fatalf("empty E should render empty string, got %q", got)
	}
}

func TestAX7_E_Ugly(t *testing.T) {
	err := E("agent.Dispatch", "", NewError("root"))
	got := err.Error()
	if got != "agent.Dispatch: root" {
		t.Fatalf("unexpected E edge rendering: %q", got)
	}
}

func TestAX7_EWithRecovery_Good(t *testing.T) {
	retryAfter := 5 * time.Second
	err := EWithRecovery("agent.Dispatch", "retry", nil, true, &retryAfter, "retry later")
	if !IsRetryable(err) || RecoveryAction(err) != "retry later" {
		t.Fatalf("expected recovery metadata, got %v", err)
	}
}

func TestAX7_EWithRecovery_Bad(t *testing.T) {
	err := EWithRecovery("agent.Dispatch", "permanent", nil, false, nil, "")
	got, ok := RetryAfter(err)
	if IsRetryable(err) || ok || got != nil {
		t.Fatalf("expected no retry hints, got retryable=%v retryAfter=%v ok=%v", IsRetryable(err), got, ok)
	}
}

func TestAX7_EWithRecovery_Ugly(t *testing.T) {
	retryAfter := 10 * time.Millisecond
	err := EWithRecovery("", "", NewError("root"), true, &retryAfter, "")
	got, ok := RetryAfter(err)
	if !ok || got == nil || *got != retryAfter {
		t.Fatalf("expected retry-after edge metadata, got %v ok=%v", got, ok)
	}
}

func TestAX7_Wrap_Good(t *testing.T) {
	root := NewError("root")
	err := Wrap(root, "agent.Dispatch", "failed")
	if !Is(err, root) || Op(err) != "agent.Dispatch" {
		t.Fatalf("expected wrapped root with op, got %v", err)
	}
}

func TestAX7_Wrap_Bad(t *testing.T) {
	err := Wrap(nil, "agent.Dispatch", "failed")
	if err != nil {
		t.Fatalf("nil wrap should return nil, got %v", err)
	}
	if Root(err) != nil {
		t.Fatalf("nil wrapped root should stay nil")
	}
}

func TestAX7_Wrap_Ugly(t *testing.T) {
	inner := NewCode("agent.failed", "root")
	err := Wrap(inner, "agent.Dispatch", "failed")
	if ErrCode(err) != "agent.failed" {
		t.Fatalf("expected inherited code, got %q", ErrCode(err))
	}
}

func TestAX7_WrapWithRecovery_Good(t *testing.T) {
	retryAfter := time.Second
	err := WrapWithRecovery(NewError("root"), "agent.Dispatch", "failed", true, &retryAfter, "retry")
	if !IsRetryable(err) || RecoveryAction(err) != "retry" {
		t.Fatalf("expected explicit recovery metadata, got %v", err)
	}
}

func TestAX7_WrapWithRecovery_Bad(t *testing.T) {
	err := WrapWithRecovery(nil, "agent.Dispatch", "failed", true, nil, "retry")
	if err != nil {
		t.Fatalf("nil WrapWithRecovery should return nil, got %v", err)
	}
	if IsRetryable(err) {
		t.Fatal("nil error must not be retryable")
	}
}

func TestAX7_WrapWithRecovery_Ugly(t *testing.T) {
	innerDelay := time.Second
	outerDelay := 2 * time.Second
	inner := NewCodeWithRecovery("inner", "root", true, &innerDelay, "inner")
	err := WrapWithRecovery(inner, "outer", "failed", false, &outerDelay, "outer")
	got, _ := RetryAfter(err)
	if !IsRetryable(err) || got == nil || *got != outerDelay || RecoveryAction(err) != "outer" {
		t.Fatalf("outer hints should win while inner retryability remains visible, err=%v retryAfter=%v", err, got)
	}
}

func TestAX7_WrapCode_Good(t *testing.T) {
	root := NewError("root")
	err := WrapCode(root, "agent.failed", "agent.Dispatch", "failed")
	if ErrCode(err) != "agent.failed" || !Is(err, root) {
		t.Fatalf("expected coded wrapper, got %v", err)
	}
}

func TestAX7_WrapCode_Bad(t *testing.T) {
	err := WrapCode(nil, "", "agent.Dispatch", "failed")
	if err != nil {
		t.Fatalf("nil error and empty code should return nil, got %v", err)
	}
	if ErrCode(err) != "" {
		t.Fatalf("nil coded error should have empty code")
	}
}

func TestAX7_WrapCode_Ugly(t *testing.T) {
	err := WrapCode(nil, "agent.failed", "agent.Dispatch", "failed")
	if err == nil || ErrCode(err) != "agent.failed" {
		t.Fatalf("code-only WrapCode should create coded error, got %v", err)
	}
}

func TestAX7_WrapCodeWithRecovery_Good(t *testing.T) {
	retryAfter := time.Second
	err := WrapCodeWithRecovery(NewError("root"), "agent.failed", "agent.Dispatch", "failed", true, &retryAfter, "retry")
	if ErrCode(err) != "agent.failed" || !IsRetryable(err) {
		t.Fatalf("expected coded retryable error, got %v", err)
	}
}

func TestAX7_WrapCodeWithRecovery_Bad(t *testing.T) {
	err := WrapCodeWithRecovery(nil, "", "agent.Dispatch", "failed", true, nil, "retry")
	if err != nil {
		t.Fatalf("nil error and empty code should return nil, got %v", err)
	}
	if RecoveryAction(err) != "" {
		t.Fatal("nil error must not expose recovery action")
	}
}

func TestAX7_WrapCodeWithRecovery_Ugly(t *testing.T) {
	retryAfter := 3 * time.Second
	err := WrapCodeWithRecovery(nil, "agent.failed", "", "", true, &retryAfter, "")
	got, ok := RetryAfter(err)
	if err == nil || !ok || got == nil || *got != retryAfter {
		t.Fatalf("code-only recovery wrapper lost retry-after: err=%v got=%v ok=%v", err, got, ok)
	}
}

func TestAX7_NewCode_Good(t *testing.T) {
	err := NewCode("agent.failed", "dispatch failed")
	if ErrCode(err) != "agent.failed" || Message(err) != "dispatch failed" {
		t.Fatalf("unexpected coded error: %v", err)
	}
}

func TestAX7_NewCode_Bad(t *testing.T) {
	err := NewCode("", "dispatch failed")
	got := ErrCode(err)
	if got != "" {
		t.Fatalf("empty code should stay empty, got %q", got)
	}
}

func TestAX7_NewCode_Ugly(t *testing.T) {
	err := NewCode("", "")
	got := err.Error()
	if got != "" {
		t.Fatalf("empty coded error should render empty, got %q", got)
	}
}

func TestAX7_NewCodeWithRecovery_Good(t *testing.T) {
	retryAfter := time.Minute
	err := NewCodeWithRecovery("agent.retry", "retry", true, &retryAfter, "retry")
	if ErrCode(err) != "agent.retry" || !IsRetryable(err) {
		t.Fatalf("expected retryable coded error, got %v", err)
	}
}

func TestAX7_NewCodeWithRecovery_Bad(t *testing.T) {
	err := NewCodeWithRecovery("", "permanent", false, nil, "")
	got, ok := RetryAfter(err)
	if IsRetryable(err) || ok || got != nil {
		t.Fatalf("expected no retry metadata, got retryAfter=%v ok=%v", got, ok)
	}
}

func TestAX7_NewCodeWithRecovery_Ugly(t *testing.T) {
	err := NewCodeWithRecovery("", "", true, nil, "inspect")
	if !IsRetryable(err) || RecoveryAction(err) != "inspect" {
		t.Fatalf("expected recovery metadata without code/message, got %v", err)
	}
}

func TestAX7_RetryAfter_Good(t *testing.T) {
	delay := 42 * time.Second
	got, ok := RetryAfter(&Err{Msg: "retry", RetryAfter: &delay})
	if !ok || got == nil || *got != delay {
		t.Fatalf("want retry-after %v, got %v ok=%v", delay, got, ok)
	}
}

func TestAX7_RetryAfter_Bad(t *testing.T) {
	got, ok := RetryAfter(NewError("plain"))
	if ok || got != nil {
		t.Fatalf("plain error should not expose retry-after, got %v ok=%v", got, ok)
	}
	if retryableHint(NewError("plain")) {
		t.Fatal("plain error should not be retryable")
	}
}

func TestAX7_RetryAfter_Ugly(t *testing.T) {
	delay := time.Nanosecond
	err := Wrap(&Err{Msg: "inner", RetryAfter: &delay}, "outer", "failed")
	got, ok := RetryAfter(err)
	if !ok || got == nil || *got != delay {
		t.Fatalf("nested retry-after missing, got %v ok=%v", got, ok)
	}
}

func TestAX7_IsRetryable_Good(t *testing.T) {
	err := &Err{Msg: "retry", Retryable: true}
	got := IsRetryable(err)
	if !got {
		t.Fatal("expected retryable error")
	}
}

func TestAX7_IsRetryable_Bad(t *testing.T) {
	err := &Err{Msg: "permanent"}
	got := IsRetryable(err)
	if got {
		t.Fatal("non-retryable Err reported retryable")
	}
}

func TestAX7_IsRetryable_Ugly(t *testing.T) {
	inner := &Err{Msg: "inner", Retryable: true}
	err := fmt.Errorf("stdlib wrapper: %w", inner)
	if !IsRetryable(err) {
		t.Fatal("retryable metadata should be found through stdlib wrapper")
	}
}

func TestAX7_RecoveryAction_Good(t *testing.T) {
	err := &Err{Msg: "recover", NextAction: "retry later"}
	got := RecoveryAction(err)
	if got != "retry later" {
		t.Fatalf("want retry later, got %q", got)
	}
}

func TestAX7_RecoveryAction_Bad(t *testing.T) {
	err := &Err{Msg: "no action"}
	got := RecoveryAction(err)
	if got != "" {
		t.Fatalf("want empty recovery action, got %q", got)
	}
}

func TestAX7_RecoveryAction_Ugly(t *testing.T) {
	inner := &Err{Msg: "inner", NextAction: "inspect"}
	err := Wrap(inner, "outer", "failed")
	if RecoveryAction(err) != "inspect" {
		t.Fatalf("nested recovery action missing, got %q", RecoveryAction(err))
	}
}

func TestAX7_Is_Good(t *testing.T) {
	root := NewError("root")
	err := Wrap(root, "outer", "failed")
	if !Is(err, root) {
		t.Fatal("wrapped error should match root")
	}
}

func TestAX7_Is_Bad(t *testing.T) {
	left := NewError("left")
	right := NewError("right")
	if Is(left, right) {
		t.Fatal("different errors should not match")
	}
}

func TestAX7_Is_Ugly(t *testing.T) {
	got := Is(nil, nil)
	if !got {
		t.Fatal("errors.Is(nil, nil) should be true")
	}
}

func TestAX7_As_Good(t *testing.T) {
	err := E("op", "msg", nil)
	var got *Err
	if !As(err, &got) || got.Op != "op" {
		t.Fatalf("expected *Err with op, got %#v", got)
	}
}

func TestAX7_As_Bad(t *testing.T) {
	err := NewError("plain")
	var got *Err
	if As(err, &got) || got != nil {
		t.Fatalf("plain error should not match *Err, got %#v", got)
	}
}

func TestAX7_As_Ugly(t *testing.T) {
	var got *Err
	matched := As(nil, &got)
	if matched || got != nil {
		t.Fatalf("nil error should not match, matched=%v got=%#v", matched, got)
	}
}

func TestAX7_NewError_Good(t *testing.T) {
	err := NewError("simple")
	got := err.Error()
	if got != "simple" {
		t.Fatalf("want simple, got %q", got)
	}
}

func TestAX7_NewError_Bad(t *testing.T) {
	err := NewError("")
	got := err.Error()
	if got != "" {
		t.Fatalf("empty error should render empty, got %q", got)
	}
}

func TestAX7_NewError_Ugly(t *testing.T) {
	err := NewError("line\nbreak")
	got := err.Error()
	if got != "line\nbreak" {
		t.Fatalf("want newline-preserving error, got %q", got)
	}
}

func TestAX7_Join_Good(t *testing.T) {
	left := NewError("left")
	right := NewError("right")
	err := Join(left, right)
	if !Is(err, left) || !Is(err, right) {
		t.Fatalf("joined error should match both inputs, got %v", err)
	}
}

func TestAX7_Join_Bad(t *testing.T) {
	err := Join(nil, nil)
	if err != nil {
		t.Fatalf("joining nil errors should return nil, got %v", err)
	}
	if Root(err) != nil {
		t.Fatal("nil join should have nil root")
	}
}

func TestAX7_Join_Ugly(t *testing.T) {
	root := NewError("root")
	err := Join(nil, root)
	if !Is(err, root) {
		t.Fatalf("join with nil should preserve real error, got %v", err)
	}
}

func TestAX7_Op_Good(t *testing.T) {
	err := Wrap(E("inner", "failed", nil), "outer", "failed")
	got := Op(err)
	if got != "outer" {
		t.Fatalf("want outer op, got %q", got)
	}
}

func TestAX7_Op_Bad(t *testing.T) {
	err := NewError("plain")
	got := Op(err)
	if got != "" {
		t.Fatalf("plain error should have empty op, got %q", got)
	}
}

func TestAX7_Op_Ugly(t *testing.T) {
	got := Op(nil)
	if got != "" {
		t.Fatalf("nil error should have empty op, got %q", got)
	}
}

func TestAX7_ErrCode_Good(t *testing.T) {
	err := Wrap(NewCode("agent.failed", "root"), "outer", "failed")
	got := ErrCode(err)
	if got != "agent.failed" {
		t.Fatalf("want inherited code, got %q", got)
	}
}

func TestAX7_ErrCode_Bad(t *testing.T) {
	err := E("op", "msg", nil)
	got := ErrCode(err)
	if got != "" {
		t.Fatalf("uncoded error should have empty code, got %q", got)
	}
}

func TestAX7_ErrCode_Ugly(t *testing.T) {
	got := ErrCode(nil)
	if got != "" {
		t.Fatalf("nil error should have empty code, got %q", got)
	}
}

func TestAX7_Message_Good(t *testing.T) {
	err := E("op", "the message", NewError("root"))
	got := Message(err)
	if got != "the message" {
		t.Fatalf("want structured message, got %q", got)
	}
}

func TestAX7_Message_Bad(t *testing.T) {
	err := NewError("plain message")
	got := Message(err)
	if got != "plain message" {
		t.Fatalf("plain error should return Error text, got %q", got)
	}
}

func TestAX7_Message_Ugly(t *testing.T) {
	got := Message(nil)
	if got != "" {
		t.Fatalf("nil error should have empty message, got %q", got)
	}
}

func TestAX7_Root_Good(t *testing.T) {
	root := NewError("root")
	err := Wrap(Wrap(root, "inner", "failed"), "outer", "failed")
	got := Root(err)
	if got != root {
		t.Fatalf("want root, got %v", got)
	}
}

func TestAX7_Root_Bad(t *testing.T) {
	err := NewError("plain")
	got := Root(err)
	if got != err {
		t.Fatalf("plain error should be its own root, got %v", got)
	}
}

func TestAX7_Root_Ugly(t *testing.T) {
	left := NewError("left")
	err := Join(left, NewError("right"))
	got := Root(err)
	if got != left {
		t.Fatalf("joined root should follow first child, got %v", got)
	}
}

func TestAX7_AllOps_Good(t *testing.T) {
	err := Wrap(E("inner", "failed", nil), "outer", "failed")
	var ops []string
	for op := range AllOps(err) {
		ops = append(ops, op)
	}
	if !slices.Equal(ops, []string{"outer", "inner"}) {
		t.Fatalf("unexpected ops: %v", ops)
	}
}

func TestAX7_AllOps_Bad(t *testing.T) {
	var ops []string
	for op := range AllOps(NewError("plain")) {
		ops = append(ops, op)
	}
	if len(ops) != 0 {
		t.Fatalf("plain error should have no ops, got %v", ops)
	}
}

func TestAX7_AllOps_Ugly(t *testing.T) {
	err := Join(E("left", "failed", nil), E("right", "failed", nil))
	var ops []string
	for op := range AllOps(err) {
		ops = append(ops, op)
	}
	if !slices.Equal(ops, []string{"left", "right"}) {
		t.Fatalf("joined ops should walk both branches, got %v", ops)
	}
}

func TestAX7_StackTrace_Good(t *testing.T) {
	err := Wrap(E("inner", "failed", nil), "outer", "failed")
	got := StackTrace(err)
	if !slices.Equal(got, []string{"outer", "inner"}) {
		t.Fatalf("unexpected stack trace: %v", got)
	}
}

func TestAX7_StackTrace_Bad(t *testing.T) {
	got := StackTrace(NewError("plain"))
	if len(got) != 0 {
		t.Fatalf("plain error should have empty stack, got %v", got)
	}
	if FormatStackTrace(NewError("plain")) != "" {
		t.Fatal("plain error should format as empty stack")
	}
}

func TestAX7_StackTrace_Ugly(t *testing.T) {
	got := StackTrace(nil)
	if len(got) != 0 {
		t.Fatalf("nil error should have empty stack, got %v", got)
	}
	if FormatStackTrace(nil) != "" {
		t.Fatal("nil error should format as empty stack")
	}
}

func TestAX7_FormatStackTrace_Good(t *testing.T) {
	err := Wrap(E("inner", "failed", nil), "outer", "failed")
	got := FormatStackTrace(err)
	if got != "outer -> inner" {
		t.Fatalf("want outer -> inner, got %q", got)
	}
}

func TestAX7_FormatStackTrace_Bad(t *testing.T) {
	got := FormatStackTrace(NewError("plain"))
	if got != "" {
		t.Fatalf("plain error should have empty formatted stack, got %q", got)
	}
	if len(StackTrace(NewError("plain"))) != 0 {
		t.Fatal("plain error should have empty stack slice")
	}
}

func TestAX7_FormatStackTrace_Ugly(t *testing.T) {
	err := Join(E("left", "failed", nil), E("right", "failed", nil))
	got := FormatStackTrace(err)
	if got != "left -> right" {
		t.Fatalf("joined stack should include both branches, got %q", got)
	}
}

func TestAX7_LogError_Good(t *testing.T) {
	buf := ax7DefaultBuffer(t)
	root := NewError("root")
	err := LogError(root, "agent.Dispatch", "failed")
	if !Is(err, root) || !strings.Contains(buf.String(), "[ERR] failed") {
		t.Fatalf("expected logged wrapped error, err=%v output=%q", err, buf.String())
	}
}

func TestAX7_LogError_Bad(t *testing.T) {
	buf := ax7DefaultBuffer(t)
	err := LogError(nil, "agent.Dispatch", "failed")
	if err != nil || buf.Len() != 0 {
		t.Fatalf("nil LogError should return nil and not log, err=%v output=%q", err, buf.String())
	}
}

func TestAX7_LogError_Ugly(t *testing.T) {
	buf := ax7DefaultBuffer(t)
	err := LogError(E("inner", "failed", NewError("root")), "outer", "failed")
	if Op(err) != "outer" || !strings.Contains(buf.String(), `stack="inner"`) {
		t.Fatalf("expected outer wrapped error and original stack log, err=%v output=%q", err, buf.String())
	}
}

func TestAX7_LogWarn_Good(t *testing.T) {
	buf := ax7DefaultBuffer(t)
	root := NewError("root")
	err := LogWarn(root, "cache.Get", "miss")
	if !Is(err, root) || !strings.Contains(buf.String(), "[WRN] miss") {
		t.Fatalf("expected logged warning, err=%v output=%q", err, buf.String())
	}
}

func TestAX7_LogWarn_Bad(t *testing.T) {
	buf := ax7DefaultBuffer(t)
	err := LogWarn(nil, "cache.Get", "miss")
	if err != nil || buf.Len() != 0 {
		t.Fatalf("nil LogWarn should return nil and not log, err=%v output=%q", err, buf.String())
	}
}

func TestAX7_LogWarn_Ugly(t *testing.T) {
	buf := ax7DefaultBuffer(t)
	err := LogWarn(NewCode("cache.miss", "miss"), "cache.Get", "fallback")
	if ErrCode(err) != "cache.miss" || !strings.Contains(buf.String(), "[WRN] fallback") {
		t.Fatalf("expected coded warning, err=%v output=%q", err, buf.String())
	}
}

func TestAX7_Must_Good(t *testing.T) {
	buf := ax7DefaultBuffer(t)
	Must(nil, "startup", "ready")
	if buf.Len() != 0 {
		t.Fatalf("nil Must should not log, got %q", buf.String())
	}
}

func TestAX7_Must_Bad(t *testing.T) {
	buf := ax7DefaultBuffer(t)
	didPanic := false
	func() {
		defer func() { didPanic = recover() != nil }()
		Must(NewError("boom"), "startup", "failed")
	}()
	if !didPanic || !strings.Contains(buf.String(), "[ERR] failed") {
		t.Fatalf("Must should panic after logging, panic=%v output=%q", didPanic, buf.String())
	}
}

func TestAX7_Must_Ugly(t *testing.T) {
	buf := ax7DefaultBuffer(t)
	var recovered any
	func() {
		defer func() { recovered = recover() }()
		Must(NewCode("startup.failed", "boom"), "startup", "failed")
	}()
	if ErrCode(recovered.(error)) != "startup.failed" || !strings.Contains(buf.String(), "[ERR] failed") {
		t.Fatalf("Must should preserve code in panic, recovered=%v output=%q", recovered, buf.String())
	}
}
