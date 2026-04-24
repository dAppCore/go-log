// AX-10 CLI driver for go-log. It exercises the public logging and structured
// error helpers without depending on the repository's unit test package.
//
//	task -d tests/cli/log test
//	go run ./tests/cli/log
package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	corelog "dappco.re/go/log"
)

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error {
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if err := verifyLoggerOutput(); err != nil {
		return fmt.Errorf("logger output: %w", err)
	}
	if err := verifyStructuredErrors(); err != nil {
		return fmt.Errorf("structured errors: %w", err)
	}
	if err := verifyDefaultLoggerAndRotationHooks(); err != nil {
		return fmt.Errorf("default logger and rotation hooks: %w", err)
	}
	if username := corelog.Username(); username == "" {
		return errors.New("username should not be empty")
	}
	return nil
}

func verifyLoggerOutput() error {
	var buf bytes.Buffer
	logger := corelog.New(corelog.Options{
		Level:      corelog.LevelDebug,
		Output:     &buf,
		RedactKeys: []string{"secret"},
	})

	retryAfter := 3 * time.Second
	err := corelog.EWithRecovery("driver.inner", "temporary failure", errors.New("network"), true, &retryAfter, "retry later")
	err = corelog.WrapCode(err, "AX10_TEMPORARY", "driver.outer", "outer failure")

	logger.Debug("debug event", "component", "ax-10")
	logger.Info("login\nattempt", "user", "alice", "secret", "token\nvalue")
	logger.Warn("slow path", "attempt", 2)
	logger.Error("request failed", "err", err)
	logger.Security("suspicious login", "ip", "127.0.0.1")

	output := buf.String()
	for _, want := range []string{
		"[DBG] debug event",
		"[INF] login\\nattempt",
		"user=\"alice\"",
		"secret=\"[REDACTED]\"",
		"[WRN] slow path attempt=2",
		"[ERR] request failed",
		"err=driver.outer: outer failure [AX10_TEMPORARY]: driver.inner: temporary failure: network",
		"op=\"driver.outer\"",
		"stack=\"driver.outer -> driver.inner\"",
		"retryable=true",
		"retry_after_seconds=3",
		"next_action=\"retry later\"",
		"[SEC] suspicious login ip=\"127.0.0.1\"",
	} {
		if !strings.Contains(output, want) {
			return fmt.Errorf("output missing %q in %q", want, output)
		}
	}

	if strings.Contains(output, "token\nvalue") {
		return errors.New("log output contains an unescaped newline in a value")
	}
	if got := strings.Count(strings.TrimSpace(output), "\n"); got != 4 {
		return fmt.Errorf("expected 5 log lines, got %d", got+1)
	}

	var quiet bytes.Buffer
	corelog.New(corelog.Options{Level: corelog.LevelQuiet, Output: &quiet}).Security("hidden")
	if quiet.Len() != 0 {
		return errors.New("quiet logger emitted security output")
	}

	return nil
}

func verifyStructuredErrors() error {
	root := errors.New("root cause")
	inner := corelog.WrapCode(root, "AX10_FAILURE", "driver.inner", "inner failed")
	outer := corelog.Wrap(inner, "driver.outer", "outer failed")

	if !corelog.Is(outer, root) {
		return errors.New("wrapped error should match root cause")
	}
	if got := corelog.Root(outer); got != root {
		return fmt.Errorf("root = %v, want %v", got, root)
	}
	if got := corelog.ErrCode(outer); got != "AX10_FAILURE" {
		return fmt.Errorf("error code = %q, want AX10_FAILURE", got)
	}
	if got := corelog.Op(outer); got != "driver.outer" {
		return fmt.Errorf("op = %q, want driver.outer", got)
	}
	if got := corelog.Message(outer); got != "outer failed" {
		return fmt.Errorf("message = %q, want outer failed", got)
	}
	if got := corelog.FormatStackTrace(outer); got != "driver.outer -> driver.inner" {
		return fmt.Errorf("stack trace = %q", got)
	}

	ops := make([]string, 0, 2)
	for op := range corelog.AllOps(outer) {
		ops = append(ops, op)
	}
	if strings.Join(ops, ",") != "driver.outer,driver.inner" {
		return fmt.Errorf("ops = %v", ops)
	}

	retryAfter := 5 * time.Second
	retryable := corelog.WrapWithRecovery(root, "driver.retry", "retryable failure", true, &retryAfter, "retry with backoff")
	if !corelog.IsRetryable(retryable) {
		return errors.New("retryable error should report retryable")
	}
	gotRetryAfter, ok := corelog.RetryAfter(retryable)
	if !ok || gotRetryAfter == nil || *gotRetryAfter != retryAfter {
		return fmt.Errorf("retry after = %v, ok=%v", gotRetryAfter, ok)
	}
	if got := corelog.RecoveryAction(retryable); got != "retry with backoff" {
		return fmt.Errorf("recovery action = %q", got)
	}

	joined := corelog.Join(outer, corelog.NewCode("AX10_JOINED", "joined failure"))
	var logErr *corelog.Err
	if !corelog.As(joined, &logErr) {
		return errors.New("joined error should expose a structured error")
	}
	if got := corelog.NewError("plain").Error(); got != "plain" {
		return fmt.Errorf("new error = %q, want plain", got)
	}

	return nil
}

func verifyDefaultLoggerAndRotationHooks() error {
	var defaultBuf bytes.Buffer
	originalDefault := corelog.Default()
	corelog.SetDefault(corelog.New(corelog.Options{Level: corelog.LevelInfo, Output: &defaultBuf}))
	defer corelog.SetDefault(originalDefault)

	corelog.Info("default info", "ok", true)
	if output := defaultBuf.String(); !strings.Contains(output, "[INF] default info ok=true") {
		return fmt.Errorf("default logger output = %q", output)
	}

	var rotated bytes.Buffer
	originalFactory := corelog.RotationWriterFactory
	corelog.RotationWriterFactory = func(opts corelog.RotationOptions) io.WriteCloser {
		if opts.Filename != "ax10.log" {
			return nopWriteCloser{Writer: io.Discard}
		}
		return nopWriteCloser{Writer: &rotated}
	}
	defer func() {
		corelog.RotationWriterFactory = originalFactory
	}()

	logger := corelog.New(corelog.Options{
		Level: corelog.LevelInfo,
		Rotation: &corelog.RotationOptions{
			Filename: "ax10.log",
		},
	})
	logger.Info("rotated output")
	if output := rotated.String(); !strings.Contains(output, "[INF] rotated output") {
		return fmt.Errorf("rotation writer output = %q", output)
	}

	return nil
}
