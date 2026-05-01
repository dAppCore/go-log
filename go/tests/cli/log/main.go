// AX-10 CLI driver for go-log. It exercises the public logging and structured
// error helpers without depending on the repository's unit test package.
//
//	task -d go/tests/cli/log test
//	go run ./go/tests/cli/log
package main

import (
	"time"

	core "dappco.re/go"
	corelog "dappco.re/go/log"
)

func main() {
	if r := run(); !r.OK {
		core.Print(core.Stderr(), "%s", r.Error())
		core.Exit(1)
	}
}

func run() core.Result {
	if r := verifyLoggerOutput(); !r.OK {
		return core.Fail(core.Errorf("logger output: %s", r.Error()))
	}
	if r := verifyStructuredErrors(); !r.OK {
		return core.Fail(core.Errorf("structured values: %s", r.Error()))
	}
	if r := verifyDefaultLoggerAndRotationHooks(); !r.OK {
		return core.Fail(core.Errorf("default logger and rotation hooks: %s", r.Error()))
	}
	if username := corelog.Username(); username == "" {
		return core.Fail(core.NewError("username should not be empty"))
	}
	return core.Ok(nil)
}

func verifyLoggerOutput() core.Result {
	buf := core.NewBuffer()
	logger := corelog.New(corelog.Options{
		Level:      corelog.LevelDebug,
		Output:     buf,
		RedactKeys: []string{"secret"},
	})

	network := core.NewError("network")
	retryAfter := 3 * time.Second
	inner := requireError(corelog.EWithRecovery("driver.inner", "temporary failure", network, true, &retryAfter, "retry later"), "inner")
	if !inner.OK {
		return inner
	}
	wrapped := requireError(corelog.WrapCode(inner.Value.(error), "AX10_TEMPORARY", "driver.outer", "outer failure"), "outer")
	if !wrapped.OK {
		return wrapped
	}

	logger.Debug("debug event", "component", "ax-10")
	logger.Info("login\nattempt", "user", "alice", "secret", "token\nvalue")
	logger.Warn("slow route", "attempt", 2)
	logger.Error("request failed", "err", wrapped.Value.(error))
	logger.Security("suspicious entry", "ip", "127.0.0.1")

	output := buf.String()
	if r := containsAll(output, []string{
		"[DBG] debug event",
		"[INF] login\\nattempt",
		"user=\"alice\"",
		"secret=\"[REDACTED]\"",
		"[WRN] slow route attempt=2",
		"[ERR] request failed",
		"err=driver.outer: outer failure [AX10_TEMPORARY]: driver.inner: temporary failure: network",
		"op=\"driver.outer\"",
		"stack=\"driver.outer -> driver.inner\"",
		"retryable=true",
		"retry_after_seconds=3",
		"next_action=\"retry later\"",
		"[SEC] suspicious entry ip=\"127.0.0.1\"",
	}); !r.OK {
		return r
	}

	if core.Contains(output, "token\nvalue") {
		return core.Fail(core.NewError("output contains an unescaped newline in a value"))
	}
	if got := lineCount(output); got != 5 {
		return core.Fail(core.Errorf("expected 5 output lines, got %d", got))
	}

	quiet := core.NewBuffer()
	corelog.New(corelog.Options{Level: corelog.LevelQuiet, Output: quiet}).Security("hidden")
	if quiet.Len() != 0 {
		return core.Fail(core.NewError("quiet logger emitted security output"))
	}

	return core.Ok(nil)
}

func verifyStructuredErrors() core.Result {
	root := core.NewError("root cause")
	inner := requireError(corelog.WrapCode(root, "AX10_FAILURE", "driver.inner", "inner failed"), "inner")
	if !inner.OK {
		return inner
	}
	outer := requireError(corelog.Wrap(inner.Value.(error), "driver.outer", "outer failed"), "outer")
	if !outer.OK {
		return outer
	}
	outerErr := outer.Value.(error)

	if !corelog.Is(outerErr, root) {
		return core.Fail(core.NewError("wrapped value should match root cause"))
	}
	rootResult := requireError(corelog.Root(outerErr), "root")
	if !rootResult.OK {
		return rootResult
	}
	if rootResult.Value.(error) != root {
		return core.Fail(core.Errorf("root = %v, want %v", rootResult.Value, root))
	}
	if got := corelog.ErrCode(outerErr); got != "AX10_FAILURE" {
		return core.Fail(core.Errorf("code = %q, want AX10_FAILURE", got))
	}
	if got := corelog.Op(outerErr); got != "driver.outer" {
		return core.Fail(core.Errorf("op = %q, want driver.outer", got))
	}
	if got := corelog.Message(outerErr); got != "outer failed" {
		return core.Fail(core.Errorf("message = %q, want outer failed", got))
	}
	if got := corelog.FormatStackTrace(outerErr); got != "driver.outer -> driver.inner" {
		return core.Fail(core.Errorf("stack trace = %q", got))
	}

	ops := make([]string, 0, 2)
	for op := range corelog.AllOps(outerErr) {
		ops = append(ops, op)
	}
	if core.Join(",", ops...) != "driver.outer,driver.inner" {
		return core.Fail(core.Errorf("ops = %v", ops))
	}

	retryAfter := 5 * time.Second
	retryable := requireError(corelog.WrapWithRecovery(root, "driver.retry", "retryable failure", true, &retryAfter, "retry with backoff"), "retryable")
	if !retryable.OK {
		return retryable
	}
	retryErr := retryable.Value.(error)
	if !corelog.IsRetryable(retryErr) {
		return core.Fail(core.NewError("retryable value should report retryable"))
	}
	gotRetryAfter, ok := corelog.RetryAfter(retryErr)
	if !ok || gotRetryAfter == nil || *gotRetryAfter != retryAfter {
		return core.Fail(core.Errorf("retry after = %v, ok=%v", gotRetryAfter, ok))
	}
	if got := corelog.RecoveryAction(retryErr); got != "retry with backoff" {
		return core.Fail(core.Errorf("recovery action = %q", got))
	}

	joined := requireError(corelog.Join(outerErr, requireError(corelog.NewCode("AX10_JOINED", "joined failure"), "joined").Value.(error)), "joined")
	if !joined.OK {
		return joined
	}
	var logErr *corelog.Err
	if !corelog.As(joined.Value.(error), &logErr) {
		return core.Fail(core.NewError("joined value should expose a structured value"))
	}
	plain := requireError(corelog.NewError("plain"), "plain")
	if !plain.OK {
		return plain
	}
	if got := plain.Value.(error).Error(); got != "plain" {
		return core.Fail(core.Errorf("new value = %q, want plain", got))
	}

	return core.Ok(nil)
}

func verifyDefaultLoggerAndRotationHooks() core.Result {
	defaultBuf := core.NewBuffer()
	originalDefault := corelog.Default()
	corelog.SetDefault(corelog.New(corelog.Options{Level: corelog.LevelInfo, Output: defaultBuf}))
	defer corelog.SetDefault(originalDefault)

	corelog.Info("default info", "ok", true)
	if output := defaultBuf.String(); !core.Contains(output, "[INF] default info ok=true") {
		return core.Fail(core.Errorf("default logger output = %q", output))
	}

	fsys := (&core.Fs{}).NewUnrestricted()
	dir := fsys.TempDir("go-log-cli-")
	if dir == "" {
		return core.Fail(core.NewError("temporary directory unavailable"))
	}
	defer func() {
		if r := fsys.DeleteAll(dir); !r.OK {
			core.Print(core.Stderr(), "cleanup failed: %s", r.Error())
		}
	}()

	rotatedPath := core.PathJoin(dir, "ax10.out")
	originalFactory := corelog.RotationWriterFactory
	corelog.RotationWriterFactory = func(opts corelog.RotationOptions) core.WriteCloser {
		if opts.Filename != "ax10.out" {
			return nil
		}
		r := core.OpenFile(rotatedPath, core.O_CREATE|core.O_WRONLY|core.O_TRUNC, 0o600)
		if !r.OK {
			return nil
		}
		return r.Value.(core.WriteCloser)
	}
	defer func() {
		corelog.RotationWriterFactory = originalFactory
	}()

	logger := corelog.New(corelog.Options{
		Level:    corelog.LevelInfo,
		Rotation: &corelog.RotationOptions{Filename: "ax10.out"},
	})
	logger.Info("rotated output")
	read := core.ReadFile(rotatedPath)
	if !read.OK {
		return read
	}
	if output := string(read.Value.([]byte)); !core.Contains(output, "[INF] rotated output") {
		return core.Fail(core.Errorf("rotation writer output = %q", output))
	}

	return core.Ok(nil)
}

func requireError(r core.Result, label string) core.Result {
	if r.Value == nil {
		return core.Fail(core.Errorf("%s value is nil", label))
	}
	if _, ok := r.Value.(error); !ok {
		return core.Fail(core.Errorf("%s value is %T", label, r.Value))
	}
	return core.Ok(r.Value)
}

func containsAll(output string, wants []string) core.Result {
	for _, want := range wants {
		if !core.Contains(output, want) {
			return core.Fail(core.Errorf("output missing %q in %q", want, output))
		}
	}
	return core.Ok(nil)
}

func lineCount(output string) int {
	trimmed := core.Trim(output)
	if trimmed == "" {
		return 0
	}
	count := 1
	for _, r := range trimmed {
		if r == '\n' {
			count++
		}
	}
	return count
}
