package golog

import core "dappco.re/go"

type testWriteCloser struct {
	core.Writer
	closed bool
}

func (w *testWriteCloser) Close() error {
	w.closed = true
	return nil
}

func TestLog_Level_String_Good(t *core.T) {
	got := LevelInfo.String()
	core.AssertEqual(t, "info", got)
	core.AssertNotEqual(t, "debug", got)
}

func TestLog_Level_String_Bad(t *core.T) {
	got := Level(99).String()
	core.AssertEqual(t, "unknown", got)
	core.AssertNotEqual(t, "info", got)
}

func TestLog_Level_String_Ugly(t *core.T) {
	got := Level(-1).String()
	core.AssertEqual(t, "unknown", got)
	core.AssertNotEqual(t, "quiet", got)
}

func TestLog_New_Good(t *core.T) {
	buf := core.NewBuffer()
	logger := New(Options{Level: LevelDebug, Output: buf})
	logger.Debug("ready")
	core.AssertContains(t, buf.String(), "[DBG] ready")
}

func TestLog_New_Bad(t *core.T) {
	logger := New(Options{Level: Level(99), Output: core.Discard})
	got := logger.Level()
	core.AssertEqual(t, LevelInfo, got)
}

func TestLog_New_Ugly(t *core.T) {
	keys := []string{"secret"}
	logger := New(Options{Level: LevelInfo, Output: core.Discard, RedactKeys: keys})
	keys[0] = "mutated"
	core.AssertEqual(t, "secret", logger.redactKeys[0])
}

func TestLog_Logger_SetLevel_Good(t *core.T) {
	logger := New(Options{Level: LevelInfo, Output: core.Discard})
	logger.SetLevel(LevelDebug)
	core.AssertEqual(t, LevelDebug, logger.Level())
}

func TestLog_Logger_SetLevel_Bad(t *core.T) {
	logger := New(Options{Level: LevelDebug, Output: core.Discard})
	logger.SetLevel(Level(99))
	core.AssertEqual(t, LevelInfo, logger.Level())
}

func TestLog_Logger_SetLevel_Ugly(t *core.T) {
	logger := New(Options{Level: LevelDebug, Output: core.Discard})
	logger.SetLevel(LevelQuiet)
	core.AssertEqual(t, LevelQuiet, logger.Level())
}

func TestLog_Logger_Level_Good(t *core.T) {
	logger := New(Options{Level: LevelWarn, Output: core.Discard})
	got := logger.Level()
	core.AssertEqual(t, LevelWarn, got)
}

func TestLog_Logger_Level_Bad(t *core.T) {
	logger := New(Options{Level: Level(-5), Output: core.Discard})
	got := logger.Level()
	core.AssertEqual(t, LevelInfo, got)
}

func TestLog_Logger_Level_Ugly(t *core.T) {
	logger := New(Options{Level: LevelQuiet, Output: core.Discard})
	got := logger.Level()
	core.AssertEqual(t, LevelQuiet, got)
}

func TestLog_Logger_SetOutput_Good(t *core.T) {
	buf := core.NewBuffer()
	logger := New(Options{Level: LevelInfo, Output: core.Discard})
	logger.SetOutput(buf)
	logger.Info("switched")
	core.AssertContains(t, buf.String(), "switched")
}

func TestLog_Logger_SetOutput_Bad(t *core.T) {
	buf := core.NewBuffer()
	logger := New(Options{Level: LevelInfo, Output: buf})
	logger.SetOutput(nil)
	core.AssertNotNil(t, logger.output)
	core.AssertNotEqual(t, buf, logger.output)
}

func TestLog_Logger_SetOutput_Ugly(t *core.T) {
	first := core.NewBuffer()
	second := core.NewBuffer()
	logger := New(Options{Level: LevelInfo, Output: first})
	logger.SetOutput(second)
	logger.Info("after")
	core.AssertEqual(t, 0, first.Len())
	core.AssertContains(t, second.String(), "after")
}

func TestLog_Logger_SetRedactKeys_Good(t *core.T) {
	buf := core.NewBuffer()
	logger := New(Options{Level: LevelInfo, Output: buf})
	logger.SetRedactKeys("secret")
	logger.Info("entry", "secret", "token")
	core.AssertContains(t, buf.String(), `secret="[REDACTED]"`)
}

func TestLog_Logger_SetRedactKeys_Bad(t *core.T) {
	buf := core.NewBuffer()
	logger := New(Options{Level: LevelInfo, Output: buf, RedactKeys: []string{"secret"}})
	logger.SetRedactKeys()
	logger.Info("entry", "secret", "visible")
	core.AssertContains(t, buf.String(), `secret="visible"`)
}

func TestLog_Logger_SetRedactKeys_Ugly(t *core.T) {
	keys := []string{"secret"}
	logger := New(Options{Level: LevelInfo, Output: core.Discard})
	logger.SetRedactKeys(keys...)
	keys[0] = "mutated"
	core.AssertEqual(t, "secret", logger.redactKeys[0])
}

func TestLog_Logger_Debug_Good(t *core.T) {
	buf := core.NewBuffer()
	logger := New(Options{Level: LevelDebug, Output: buf})
	logger.Debug("debug", "agent", "codex")
	core.AssertContains(t, buf.String(), `[DBG] debug agent="codex"`)
}

func TestLog_Logger_Debug_Bad(t *core.T) {
	buf := core.NewBuffer()
	logger := New(Options{Level: LevelInfo, Output: buf})
	logger.Debug("hidden")
	core.AssertEqual(t, 0, buf.Len())
}

func TestLog_Logger_Debug_Ugly(t *core.T) {
	buf := core.NewBuffer()
	logger := New(Options{Level: LevelDebug, Output: buf})
	logger.StyleDebug = nil
	logger.Debug("nil-style")
	core.AssertContains(t, buf.String(), "[DBG] nil-style")
}

func TestLog_Logger_Info_Good(t *core.T) {
	buf := core.NewBuffer()
	logger := New(Options{Level: LevelInfo, Output: buf})
	logger.Info("info", "ok", true)
	core.AssertContains(t, buf.String(), "[INF] info ok=true")
}

func TestLog_Logger_Info_Bad(t *core.T) {
	buf := core.NewBuffer()
	logger := New(Options{Level: LevelWarn, Output: buf})
	logger.Info("hidden")
	core.AssertEqual(t, 0, buf.Len())
}

func TestLog_Logger_Info_Ugly(t *core.T) {
	buf := core.NewBuffer()
	logger := New(Options{Level: LevelInfo, Output: buf})
	logger.Info("line\nbreak", "key\tname", "value\r")
	core.AssertContains(t, buf.String(), `line\nbreak key\tname="value\r"`)
}

func TestLog_Logger_Warn_Good(t *core.T) {
	buf := core.NewBuffer()
	logger := New(Options{Level: LevelWarn, Output: buf})
	logger.Warn("warn", "attempt", 2)
	core.AssertContains(t, buf.String(), "[WRN] warn attempt=2")
}

func TestLog_Logger_Warn_Bad(t *core.T) {
	buf := core.NewBuffer()
	logger := New(Options{Level: LevelError, Output: buf})
	logger.Warn("hidden")
	core.AssertEqual(t, 0, buf.Len())
}

func TestLog_Logger_Warn_Ugly(t *core.T) {
	buf := core.NewBuffer()
	logger := New(Options{Level: LevelWarn, Output: buf})
	logger.StyleWarn = func(s string) string { return "WARN:" + s }
	logger.Warn("styled")
	core.AssertContains(t, buf.String(), "WARN:[WRN] styled")
}

func TestLog_Logger_Error_Good(t *core.T) {
	buf := core.NewBuffer()
	logger := New(Options{Level: LevelError, Output: buf})
	logger.Error("failed", "err", mustError(t, NewError("boom")))
	core.AssertContains(t, buf.String(), "[ERR] failed err=boom")
}

func TestLog_Logger_Error_Bad(t *core.T) {
	buf := core.NewBuffer()
	logger := New(Options{Level: LevelQuiet, Output: buf})
	logger.Error("hidden")
	core.AssertEqual(t, 0, buf.Len())
}

func TestLog_Logger_Error_Ugly(t *core.T) {
	buf := core.NewBuffer()
	logger := New(Options{Level: LevelError, Output: buf})
	err := mustError(t, Wrap(mustError(t, E("inner.Op", "inner failed", mustError(t, NewError("root")))), "outer.Op", "outer failed"))
	logger.Error("failed", "err", err)
	core.AssertContains(t, buf.String(), `stack="outer.Op -> inner.Op"`)
}

func TestLog_Logger_Security_Good(t *core.T) {
	buf := core.NewBuffer()
	logger := New(Options{Level: LevelError, Output: buf})
	logger.Security("entry", "user", "alice")
	core.AssertContains(t, buf.String(), `[SEC] entry user="alice"`)
}

func TestLog_Logger_Security_Bad(t *core.T) {
	buf := core.NewBuffer()
	logger := New(Options{Level: LevelQuiet, Output: buf})
	logger.Security("hidden")
	core.AssertEqual(t, 0, buf.Len())
}

func TestLog_Logger_Security_Ugly(t *core.T) {
	buf := core.NewBuffer()
	logger := New(Options{Level: LevelError, Output: buf})
	logger.StyleSecurity = func(s string) string { return "SEC:" + s }
	logger.Security("styled")
	core.AssertContains(t, buf.String(), "SEC:[SEC] styled")
}

func TestLog_Username_Good(t *core.T) {
	t.Setenv("USER", "codex")
	got := Username()
	core.AssertEqual(t, "codex", got)
}

func TestLog_Username_Bad(t *core.T) {
	t.Setenv("USER", "")
	t.Setenv("USERNAME", "fallback")
	core.AssertEqual(t, "fallback", Username())
}

func TestLog_Username_Ugly(t *core.T) {
	t.Setenv("USER", "")
	t.Setenv("USERNAME", "")
	core.AssertEqual(t, "unknown", Username())
}

func TestLog_Default_Good(t *core.T) {
	got := Default()
	core.AssertNotNil(t, got)
	core.AssertEqual(t, LevelInfo, got.Level())
}

func TestLog_Default_Bad(t *core.T) {
	original := Default()
	SetDefault(nil)
	core.AssertEqual(t, original, Default())
}

func TestLog_Default_Ugly(t *core.T) {
	original := Default()
	custom := New(Options{Level: LevelDebug, Output: core.Discard})
	SetDefault(custom)
	t.Cleanup(func() { SetDefault(original) })
	core.AssertEqual(t, custom, Default())
}

func TestLog_SetDefault_Good(t *core.T) {
	original := Default()
	custom := New(Options{Level: LevelWarn, Output: core.Discard})
	SetDefault(custom)
	t.Cleanup(func() { SetDefault(original) })
	core.AssertEqual(t, custom, Default())
}

func TestLog_SetDefault_Bad(t *core.T) {
	original := Default()
	SetDefault(nil)
	core.AssertEqual(t, original, Default())
}

func TestLog_SetDefault_Ugly(t *core.T) {
	original := Default()
	first := New(Options{Level: LevelInfo, Output: core.Discard})
	second := New(Options{Level: LevelDebug, Output: core.Discard})
	SetDefault(first)
	SetDefault(second)
	t.Cleanup(func() { SetDefault(original) })
	core.AssertEqual(t, second, Default())
}

func TestLog_SetLevel_Good(t *core.T) {
	original := Default()
	logger := New(Options{Level: LevelInfo, Output: core.Discard})
	SetDefault(logger)
	t.Cleanup(func() { SetDefault(original) })
	SetLevel(LevelDebug)
	core.AssertEqual(t, LevelDebug, logger.Level())
}

func TestLog_SetLevel_Bad(t *core.T) {
	original := Default()
	logger := New(Options{Level: LevelDebug, Output: core.Discard})
	SetDefault(logger)
	t.Cleanup(func() { SetDefault(original) })
	SetLevel(Level(99))
	core.AssertEqual(t, LevelInfo, logger.Level())
}

func TestLog_SetLevel_Ugly(t *core.T) {
	original := Default()
	logger := New(Options{Level: LevelDebug, Output: core.Discard})
	SetDefault(logger)
	t.Cleanup(func() { SetDefault(original) })
	SetLevel(LevelQuiet)
	core.AssertEqual(t, LevelQuiet, logger.Level())
}

func TestLog_SetRedactKeys_Good(t *core.T) {
	original := Default()
	buf := core.NewBuffer()
	SetDefault(New(Options{Level: LevelInfo, Output: buf}))
	t.Cleanup(func() { SetDefault(original) })
	SetRedactKeys("token")
	Info("entry", "token", "abc")
	core.AssertContains(t, buf.String(), `token="[REDACTED]"`)
}

func TestLog_SetRedactKeys_Bad(t *core.T) {
	original := Default()
	buf := core.NewBuffer()
	SetDefault(New(Options{Level: LevelInfo, Output: buf}))
	t.Cleanup(func() { SetDefault(original) })
	SetRedactKeys("token")
	SetRedactKeys()
	Info("entry", "token", "abc")
	core.AssertContains(t, buf.String(), `token="abc"`)
}

func TestLog_SetRedactKeys_Ugly(t *core.T) {
	original := Default()
	buf := core.NewBuffer()
	SetDefault(New(Options{Level: LevelInfo, Output: buf}))
	t.Cleanup(func() { SetDefault(original) })
	SetRedactKeys("")
	Info("empty-key", "", "secret")
	core.AssertContains(t, buf.String(), `="[REDACTED]"`)
}

func TestLog_Debug_Good(t *core.T) {
	original := Default()
	buf := core.NewBuffer()
	SetDefault(New(Options{Level: LevelDebug, Output: buf}))
	t.Cleanup(func() { SetDefault(original) })
	Debug("debug", "agent", "codex")
	core.AssertContains(t, buf.String(), `[DBG] debug agent="codex"`)
}

func TestLog_Debug_Bad(t *core.T) {
	original := Default()
	buf := core.NewBuffer()
	SetDefault(New(Options{Level: LevelInfo, Output: buf}))
	t.Cleanup(func() { SetDefault(original) })
	Debug("hidden")
	core.AssertEqual(t, 0, buf.Len())
}

func TestLog_Debug_Ugly(t *core.T) {
	original := Default()
	buf := core.NewBuffer()
	SetDefault(New(Options{Level: LevelDebug, Output: buf}))
	t.Cleanup(func() { SetDefault(original) })
	Debug("control\nchars", "key", "value\t")
	core.AssertContains(t, buf.String(), `control\nchars key="value\t"`)
}

func TestLog_Info_Good(t *core.T) {
	original := Default()
	buf := core.NewBuffer()
	SetDefault(New(Options{Level: LevelInfo, Output: buf}))
	t.Cleanup(func() { SetDefault(original) })
	Info("info", "ok", true)
	core.AssertContains(t, buf.String(), "[INF] info ok=true")
}

func TestLog_Info_Bad(t *core.T) {
	original := Default()
	buf := core.NewBuffer()
	SetDefault(New(Options{Level: LevelWarn, Output: buf}))
	t.Cleanup(func() { SetDefault(original) })
	Info("hidden")
	core.AssertEqual(t, 0, buf.Len())
}

func TestLog_Info_Ugly(t *core.T) {
	original := Default()
	buf := core.NewBuffer()
	SetDefault(New(Options{Level: LevelInfo, Output: buf}))
	t.Cleanup(func() { SetDefault(original) })
	Info("odd", "lonely")
	core.AssertContains(t, buf.String(), "lonely=<nil>")
}

func TestLog_Warn_Good(t *core.T) {
	original := Default()
	buf := core.NewBuffer()
	SetDefault(New(Options{Level: LevelDebug, Output: buf}))
	t.Cleanup(func() { SetDefault(original) })
	Warn("warn", "attempt", 2)
	core.AssertContains(t, buf.String(), "[WRN] warn attempt=2")
}

func TestLog_Warn_Bad(t *core.T) {
	original := Default()
	buf := core.NewBuffer()
	SetDefault(New(Options{Level: LevelError, Output: buf}))
	t.Cleanup(func() { SetDefault(original) })
	Warn("hidden")
	core.AssertEqual(t, 0, buf.Len())
}

func TestLog_Warn_Ugly(t *core.T) {
	original := Default()
	buf := core.NewBuffer()
	SetDefault(New(Options{Level: LevelDebug, Output: buf}))
	t.Cleanup(func() { SetDefault(original) })
	Warn("newline", "value", "a\nb")
	core.AssertContains(t, buf.String(), `value="a\nb"`)
}

func TestLog_Error_Good(t *core.T) {
	original := Default()
	buf := core.NewBuffer()
	SetDefault(New(Options{Level: LevelDebug, Output: buf}))
	t.Cleanup(func() { SetDefault(original) })
	Error("failed", "err", mustError(t, NewError("boom")))
	core.AssertContains(t, buf.String(), "[ERR] failed err=boom")
}

func TestLog_Error_Bad(t *core.T) {
	original := Default()
	buf := core.NewBuffer()
	SetDefault(New(Options{Level: LevelQuiet, Output: buf}))
	t.Cleanup(func() { SetDefault(original) })
	Error("hidden")
	core.AssertEqual(t, 0, buf.Len())
}

func TestLog_Error_Ugly(t *core.T) {
	original := Default()
	buf := core.NewBuffer()
	SetDefault(New(Options{Level: LevelDebug, Output: buf}))
	t.Cleanup(func() { SetDefault(original) })
	Error("failed", "err", mustError(t, E("op.Name", "failed", mustError(t, NewError("root")))))
	core.AssertContains(t, buf.String(), `op="op.Name"`)
}

func TestLog_Security_Good(t *core.T) {
	original := Default()
	buf := core.NewBuffer()
	SetDefault(New(Options{Level: LevelDebug, Output: buf}))
	t.Cleanup(func() { SetDefault(original) })
	Security("entry", "user", "alice")
	core.AssertContains(t, buf.String(), `[SEC] entry user="alice"`)
}

func TestLog_Security_Bad(t *core.T) {
	original := Default()
	buf := core.NewBuffer()
	SetDefault(New(Options{Level: LevelQuiet, Output: buf}))
	t.Cleanup(func() { SetDefault(original) })
	Security("hidden")
	core.AssertEqual(t, 0, buf.Len())
}

func TestLog_Security_Ugly(t *core.T) {
	original := Default()
	buf := core.NewBuffer()
	SetDefault(New(Options{Level: LevelDebug, Output: buf}))
	t.Cleanup(func() { SetDefault(original) })
	Security("line\nbreak", "ip", "127.0.0.1")
	core.AssertContains(t, buf.String(), `[SEC] line\nbreak ip="127.0.0.1"`)
}

func TestLog_New_RotationWriter_Good(t *core.T) {
	buf := core.NewBuffer()
	original := RotationWriterFactory
	RotationWriterFactory = func(opts RotationOptions) core.WriteCloser {
		core.AssertEqual(t, "app.out", opts.Filename)
		return &testWriteCloser{Writer: buf}
	}
	t.Cleanup(func() { RotationWriterFactory = original })
	logger := New(Options{Level: LevelInfo, Rotation: &RotationOptions{Filename: "app.out"}})
	logger.Info("rotated")
	core.AssertContains(t, buf.String(), "rotated")
}
