package log

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
)

// --- Err Type Tests ---

func TestErr_Error_Good(t *testing.T) {
	// With underlying error
	err := &Err{Op: "db.Query", Msg: "failed to query", Err: errors.New("connection refused")}
	if want, got := "db.Query: failed to query: connection refused", err.Error(); want != got {
		t.Fatalf("want %v, got %v", want, got)
	}

	// With code
	err = &Err{Op: "api.Call", Msg: "request failed", Code: "TIMEOUT"}
	if want, got := "api.Call: request failed [TIMEOUT]", err.Error(); want != got {
		t.Fatalf("want %v, got %v", want, got)
	}

	// With both underlying error and code
	err = &Err{Op: "user.Save", Msg: "save failed", Err: errors.New("duplicate key"), Code: "DUPLICATE"}
	if want, got := "user.Save: save failed [DUPLICATE]: duplicate key", err.Error(); want != got {
		t.Fatalf("want %v, got %v", want, got)
	}

	// Just op and msg
	err = &Err{Op: "cache.Get", Msg: "miss"}
	if want, got := "cache.Get: miss", err.Error(); want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestErr_Error_EmptyOp_Good(t *testing.T) {
	// No Op - should not have leading colon
	err := &Err{Msg: "just a message"}
	if want, got := "just a message", err.Error(); want != got {
		t.Fatalf("want %v, got %v", want, got)
	}

	// No Op with code
	err = &Err{Msg: "error with code", Code: "ERR_CODE"}
	if want, got := "error with code [ERR_CODE]", err.Error(); want != got {
		t.Fatalf("want %v, got %v", want, got)
	}

	// No Op with underlying error
	err = &Err{Msg: "wrapped", Err: errors.New("underlying")}
	if want, got := "wrapped: underlying", err.Error(); want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestErr_Error_EmptyMsg_Good(t *testing.T) {
	err := &Err{Op: "api.Call", Code: "TIMEOUT"}
	if want, got := "api.Call: [TIMEOUT]", err.Error(); want != got {
		t.Fatalf("want %v, got %v", want, got)
	}

	err = &Err{Op: "api.Call", Err: errors.New("underlying")}
	if want, got := "api.Call: underlying", err.Error(); want != got {
		t.Fatalf("want %v, got %v", want, got)
	}

	err = &Err{Op: "api.Call", Code: "TIMEOUT", Err: errors.New("underlying")}
	if want, got := "api.Call: [TIMEOUT]: underlying", err.Error(); want != got {
		t.Fatalf("want %v, got %v", want, got)
	}

	err = &Err{Op: "api.Call"}
	if want, got := "api.Call", err.Error(); want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestErr_Unwrap_Good(t *testing.T) {
	underlying := errors.New("underlying error")
	err := &Err{Op: "test", Msg: "wrapped", Err: underlying}

	if want, got := underlying, errors.Unwrap(err); want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
	if !errors.Is(err, underlying) {
		t.Fatal("expected true")
	}
}

// --- Error Creation Function Tests ---

func TestE_Good(t *testing.T) {
	underlying := errors.New("base error")
	err := E("op.Name", "something failed", underlying)

	if err == nil {
		t.Fatal("expected non-nil")
	}
	var logErr *Err
	if !errors.As(err, &logErr) {
		t.Fatal("expected true")
	}
	if want, got := "op.Name", logErr.Op; want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
	if want, got := "something failed", logErr.Msg; want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
	if want, got := underlying, logErr.Err; want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestE_Good_NilError(t *testing.T) {
	// E creates an error even with nil underlying - useful for errors without causes
	err := E("op.Name", "message", nil)
	if err == nil {
		t.Fatal("expected non-nil")
	}
	if want, got := "op.Name: message", err.Error(); want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestEWithRecovery_Good(t *testing.T) {
	retryAfter := time.Second * 5
	err := EWithRecovery("op.Name", "message", nil, true, &retryAfter, "retry once")

	var logErr *Err
	if err == nil {
		t.Fatal("expected non-nil")
	}
	if !As(err, &logErr) {
		t.Fatal("expected true")
	}
	if !logErr.Retryable {
		t.Fatal("expected true")
	}
	if logErr.RetryAfter == nil {
		t.Fatal("expected non-nil")
	}
	if want, got := retryAfter, *logErr.RetryAfter; want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
	if want, got := "retry once", logErr.NextAction; want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestWrap_Good(t *testing.T) {
	underlying := errors.New("base")
	err := Wrap(underlying, "handler.Process", "processing failed")

	if err == nil {
		t.Fatal("expected non-nil")
	}
	if !strings.Contains(err.Error(), "handler.Process") {
		t.Fatalf("expected %q to contain %q", err.Error(), "handler.Process")
	}
	if !strings.Contains(err.Error(), "processing failed") {
		t.Fatalf("expected %q to contain %q", err.Error(), "processing failed")
	}
	if !errors.Is(err, underlying) {
		t.Fatal("expected true")
	}
}

func TestWrap_PreservesCode_Good(t *testing.T) {
	// Create an error with a code
	inner := WrapCode(errors.New("base"), "VALIDATION_ERROR", "inner.Op", "validation failed")

	// Wrap it - should preserve the code
	outer := Wrap(inner, "outer.Op", "outer context")

	if outer == nil {
		t.Fatal("expected non-nil")
	}
	if want, got := "VALIDATION_ERROR", ErrCode(outer); want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
	if !strings.Contains(outer.Error(), "[VALIDATION_ERROR]") {
		t.Fatalf("expected %q to contain %q", outer.Error(), "[VALIDATION_ERROR]")
	}
}

func TestWrap_PreservesCode_FromNestedErrWithEmptyOuterCode_Good(t *testing.T) {
	inner := WrapCode(errors.New("base"), "VALIDATION_ERROR", "inner.Op", "validation failed")
	mid := &Err{Op: "mid.Op", Msg: "mid failed", Err: inner}

	outer := Wrap(mid, "outer.Op", "outer context")

	if outer == nil {
		t.Fatal("expected non-nil")
	}
	if want, got := "VALIDATION_ERROR", ErrCode(outer); want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
	if !strings.Contains(outer.Error(), "[VALIDATION_ERROR]") {
		t.Fatalf("expected %q to contain %q", outer.Error(), "[VALIDATION_ERROR]")
	}
}

func TestWrap_PreservesRecovery_Good(t *testing.T) {
	retryAfter := 15 * time.Second
	inner := &Err{Msg: "inner", Retryable: true, RetryAfter: &retryAfter, NextAction: "inspect input"}

	outer := Wrap(inner, "outer.Op", "outer context")

	if outer == nil {
		t.Fatal("expected non-nil")
	}
	var logErr *Err
	if !As(outer, &logErr) {
		t.Fatal("expected true")
	}
	if !logErr.Retryable {
		t.Fatal("expected true")
	}
	if logErr.RetryAfter == nil {
		t.Fatal("expected non-nil")
	}
	if want, got := retryAfter, *logErr.RetryAfter; want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
	if want, got := "inspect input", logErr.NextAction; want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestWrap_PreservesCode_FromNestedChain_Good(t *testing.T) {
	root := WrapCode(errors.New("base"), "CHAIN_ERROR", "inner", "inner failed")
	wrapped := Wrap(fmt.Errorf("mid layer: %w", root), "outer", "outer context")

	if want, got := "CHAIN_ERROR", ErrCode(wrapped); want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
	if !strings.Contains(wrapped.Error(), "[CHAIN_ERROR]") {
		t.Fatalf("expected %q to contain %q", wrapped.Error(), "[CHAIN_ERROR]")
	}
}

func TestWrap_NilError_Good(t *testing.T) {
	err := Wrap(nil, "op", "msg")
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestWrapCode_Good(t *testing.T) {
	underlying := errors.New("validation failed")
	err := WrapCode(underlying, "INVALID_INPUT", "api.Validate", "bad request")

	if err == nil {
		t.Fatal("expected non-nil")
	}
	var logErr *Err
	if !errors.As(err, &logErr) {
		t.Fatal("expected true")
	}
	if want, got := "INVALID_INPUT", logErr.Code; want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
	if want, got := "api.Validate", logErr.Op; want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
	if !strings.Contains(err.Error(), "[INVALID_INPUT]") {
		t.Fatalf("expected %q to contain %q", err.Error(), "[INVALID_INPUT]")
	}
}

func TestWrapCode_Good_EmptyCodeDoesNotInherit(t *testing.T) {
	inner := WrapCode(errors.New("base"), "INNER_CODE", "inner.Op", "inner failed")

	outer := WrapCode(inner, "", "outer.Op", "outer failed")

	var logErr *Err
	if !As(outer, &logErr) {
		t.Fatal("expected true")
	}
	if want, got := "", logErr.Code; want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestWrapCodeWithRecovery_Good(t *testing.T) {
	retryAfter := time.Minute
	err := WrapCodeWithRecovery(errors.New("validation failed"), "INVALID_INPUT", "api.Validate", "bad request", true, &retryAfter, "retry with backoff")

	var logErr *Err
	if err == nil {
		t.Fatal("expected non-nil")
	}
	if !As(err, &logErr) {
		t.Fatal("expected true")
	}
	if !logErr.Retryable {
		t.Fatal("expected true")
	}
	if logErr.RetryAfter == nil {
		t.Fatal("expected non-nil")
	}
	if want, got := retryAfter, *logErr.RetryAfter; want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
	if want, got := "retry with backoff", logErr.NextAction; want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
	if want, got := "INVALID_INPUT", logErr.Code; want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestWrapCodeWithRecovery_Good_EmptyCodeDoesNotInherit(t *testing.T) {
	retryAfter := time.Minute
	inner := WrapCodeWithRecovery(errors.New("validation failed"), "INNER_CODE", "inner.Op", "inner failed", true, &retryAfter, "retry later")

	outer := WrapCodeWithRecovery(inner, "", "outer.Op", "outer failed", true, &retryAfter, "retry later")

	var logErr *Err
	if !As(outer, &logErr) {
		t.Fatal("expected true")
	}
	if want, got := "", logErr.Code; want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestWrapCode_Good_NilError(t *testing.T) {
	// WrapCode with nil error but with code still creates an error
	err := WrapCode(nil, "CODE", "op", "msg")
	if err == nil {
		t.Fatal("expected non-nil")
	}
	if !strings.Contains(err.Error(), "[CODE]") {
		t.Fatalf("expected %q to contain %q", err.Error(), "[CODE]")
	}

	// Only returns nil when both error and code are empty
	err = WrapCode(nil, "", "op", "msg")
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestNewCode_Good(t *testing.T) {
	err := NewCode("NOT_FOUND", "resource not found")

	var logErr *Err
	if !errors.As(err, &logErr) {
		t.Fatal("expected true")
	}
	if want, got := "NOT_FOUND", logErr.Code; want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
	if want, got := "resource not found", logErr.Msg; want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
	if logErr.Err != nil {
		t.Fatalf("expected nil, got %v", logErr.Err)
	}
}

func TestNewCodeWithRecovery_Good(t *testing.T) {
	retryAfter := 2 * time.Minute
	err := NewCodeWithRecovery("NOT_FOUND", "resource not found", false, &retryAfter, "contact support")

	var logErr *Err
	if err == nil {
		t.Fatal("expected non-nil")
	}
	if !As(err, &logErr) {
		t.Fatal("expected true")
	}
	if logErr.Retryable {
		t.Fatal("expected false")
	}
	if logErr.RetryAfter == nil {
		t.Fatal("expected non-nil")
	}
	if want, got := retryAfter, *logErr.RetryAfter; want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
	if want, got := "contact support", logErr.NextAction; want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
}

// --- Standard Library Wrapper Tests ---

func TestIs_Good(t *testing.T) {
	sentinel := errors.New("sentinel")
	wrapped := Wrap(sentinel, "test", "wrapped")

	if !Is(wrapped, sentinel) {
		t.Fatal("expected true")
	}
	if Is(wrapped, errors.New("other")) {
		t.Fatal("expected false")
	}
}

func TestAs_Good(t *testing.T) {
	err := E("test.Op", "message", errors.New("base"))

	var logErr *Err
	if !As(err, &logErr) {
		t.Fatal("expected true")
	}
	if want, got := "test.Op", logErr.Op; want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestNewError_Good(t *testing.T) {
	err := NewError("simple error")
	if err == nil {
		t.Fatal("expected non-nil")
	}
	if want, got := "simple error", err.Error(); want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestJoin_Good(t *testing.T) {
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")
	joined := Join(err1, err2)

	if !errors.Is(joined, err1) {
		t.Fatal("expected true")
	}
	if !errors.Is(joined, err2) {
		t.Fatal("expected true")
	}
}

// --- Helper Function Tests ---

func TestOp_Good(t *testing.T) {
	err := E("mypackage.MyFunc", "failed", errors.New("cause"))
	if want, got := "mypackage.MyFunc", Op(err); want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestOp_Good_NotLogError(t *testing.T) {
	err := errors.New("plain error")
	if want, got := "", Op(err); want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestErrCode_Good(t *testing.T) {
	err := WrapCode(errors.New("base"), "ERR_CODE", "op", "msg")
	if want, got := "ERR_CODE", ErrCode(err); want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestErrCode_Good_NoCode(t *testing.T) {
	err := E("op", "msg", errors.New("base"))
	if want, got := "", ErrCode(err); want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestErrCode_Good_PlainError(t *testing.T) {
	err := errors.New("plain error")
	if want, got := "", ErrCode(err); want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestErrCode_Good_Nil(t *testing.T) {
	if want, got := "", ErrCode(nil); want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestRetryAfter_Good(t *testing.T) {
	retryAfter := 42 * time.Second
	err := &Err{Msg: "typed", RetryAfter: &retryAfter}

	got, ok := RetryAfter(err)
	if !ok {
		t.Fatal("expected true")
	}
	if want, got := retryAfter, *got; want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestRetryAfter_Good_NestedChain(t *testing.T) {
	retryAfter := 42 * time.Second
	inner := &Err{Msg: "typed", RetryAfter: &retryAfter}
	outer := &Err{Msg: "outer", Err: inner}

	got, ok := RetryAfter(outer)
	if !ok {
		t.Fatal("expected true")
	}
	if want, got := retryAfter, *got; want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestIsRetryable_Good(t *testing.T) {
	err := &Err{Msg: "typed", Retryable: true}
	if !IsRetryable(err) {
		t.Fatal("expected true")
	}
}

func TestRecoveryAction_Good(t *testing.T) {
	err := &Err{Msg: "typed", NextAction: "inspect"}
	if want, got := "inspect", RecoveryAction(err); want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestRecoveryAction_Good_NestedChain(t *testing.T) {
	inner := &Err{Msg: "typed", NextAction: "inspect"}
	outer := &Err{Msg: "outer", Err: inner}

	if want, got := "inspect", RecoveryAction(outer); want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestMessage_Good(t *testing.T) {
	err := E("op", "the message", errors.New("base"))
	if want, got := "the message", Message(err); want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestMessage_Good_PlainError(t *testing.T) {
	err := errors.New("plain message")
	if want, got := "plain message", Message(err); want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestMessage_Good_Nil(t *testing.T) {
	if want, got := "", Message(nil); want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestRoot_Good(t *testing.T) {
	root := errors.New("root cause")
	level1 := Wrap(root, "level1", "wrapped once")
	level2 := Wrap(level1, "level2", "wrapped twice")

	if want, got := root, Root(level2); want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestRoot_Good_SingleError(t *testing.T) {
	err := errors.New("single")
	if want, got := err, Root(err); want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestRoot_Good_Nil(t *testing.T) {
	if got := Root(nil); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

// --- Log-and-Return Helper Tests ---

func TestLogError_Good(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	logger := New(Options{Level: LevelDebug, Output: &buf})
	SetDefault(logger)
	defer SetDefault(New(Options{Level: LevelInfo}))

	underlying := errors.New("connection failed")
	err := LogError(underlying, "db.Connect", "database unavailable")

	// Check returned error
	if err == nil {
		t.Fatal("expected non-nil")
	}
	if !strings.Contains(err.Error(), "db.Connect") {
		t.Fatalf("expected %q to contain %q", err.Error(), "db.Connect")
	}
	if !strings.Contains(err.Error(), "database unavailable") {
		t.Fatalf("expected %q to contain %q", err.Error(), "database unavailable")
	}
	if !errors.Is(err, underlying) {
		t.Fatal("expected true")
	}

	// Check log output
	output := buf.String()
	if !strings.Contains(output, "[ERR]") {
		t.Fatalf("expected %q to contain %q", output, "[ERR]")
	}
	if !strings.Contains(output, "database unavailable") {
		t.Fatalf("expected %q to contain %q", output, "database unavailable")
	}
	if !strings.Contains(output, "op=\"db.Connect\"") {
		t.Fatalf("expected %q to contain %q", output, "op=\"db.Connect\"")
	}
}

func TestLogError_Good_LogsOriginalErrorContext(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Options{Level: LevelDebug, Output: &buf})
	SetDefault(logger)
	defer SetDefault(New(Options{Level: LevelInfo}))

	underlying := E("db.Query", "query failed", errors.New("timeout"))
	err := LogError(underlying, "db.Connect", "database unavailable")

	if err == nil {
		t.Fatal("expected non-nil")
	}

	output := buf.String()
	if !strings.Contains(output, "op=\"db.Connect\"") {
		t.Fatalf("expected %q to contain %q", output, "op=\"db.Connect\"")
	}
	if !strings.Contains(output, "stack=\"db.Query\"") {
		t.Fatalf("expected %q to contain %q", output, "stack=\"db.Query\"")
	}
	if strings.Contains(output, "stack=\"db.Connect -> db.Query\"") {
		t.Fatalf("expected %q not to contain %q", output, "stack=\"db.Connect -> db.Query\"")
	}
}

func TestLogError_Good_NilError(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Options{Level: LevelDebug, Output: &buf})
	SetDefault(logger)
	defer SetDefault(New(Options{Level: LevelInfo}))

	err := LogError(nil, "op", "msg")
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if got := buf.String(); got != "" {
		t.Fatalf("expected empty, got %v", got)
	}
}

func TestLogWarn_Good(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Options{Level: LevelDebug, Output: &buf})
	SetDefault(logger)
	defer SetDefault(New(Options{Level: LevelInfo}))

	underlying := errors.New("cache miss")
	err := LogWarn(underlying, "cache.Get", "falling back to db")

	if err == nil {
		t.Fatal("expected non-nil")
	}
	if !errors.Is(err, underlying) {
		t.Fatal("expected true")
	}

	output := buf.String()
	if !strings.Contains(output, "[WRN]") {
		t.Fatalf("expected %q to contain %q", output, "[WRN]")
	}
	if !strings.Contains(output, "falling back to db") {
		t.Fatalf("expected %q to contain %q", output, "falling back to db")
	}
}

func TestLogWarn_Good_NilError(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Options{Level: LevelDebug, Output: &buf})
	SetDefault(logger)
	defer SetDefault(New(Options{Level: LevelInfo}))

	err := LogWarn(nil, "op", "msg")
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if got := buf.String(); got != "" {
		t.Fatalf("expected empty, got %v", got)
	}
}

func TestMust_Good_NoError(t *testing.T) {
	// Should not panic when error is nil
	func() {
		defer func() {
			if got := recover(); got != nil {
				t.Fatalf("unexpected panic: %v", got)
			}
		}()
		Must(nil, "test", "should not panic")
	}()
}

func TestMust_Ugly_Panics(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Options{Level: LevelDebug, Output: &buf})
	SetDefault(logger)
	defer SetDefault(New(Options{Level: LevelInfo}))

	didPanic := false
	func() {
		defer func() {
			if recover() != nil {
				didPanic = true
			}
		}()
		Must(errors.New("fatal error"), "startup", "initialization failed")
	}()
	if !didPanic {
		t.Fatal("expected panic")
	}

	// Verify error was logged before panic
	output := buf.String()
	if !(strings.Contains(output, "[ERR]") || len(output) > 0) {
		t.Fatal("expected true")
	}
}

func TestStackTrace_Good(t *testing.T) {
	// Nested operations
	err := E("op1", "msg1", nil)
	err = Wrap(err, "op2", "msg2")
	err = Wrap(err, "op3", "msg3")

	stack := StackTrace(err)
	if want, got := []string{"op3", "op2", "op1"}, stack; !slices.Equal(want, got) {
		t.Fatalf("want %v, got %v", want, got)
	}

	// Format
	formatted := FormatStackTrace(err)
	if want, got := "op3 -> op2 -> op1", formatted; want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestStackTrace_Bad_PlainError(t *testing.T) {
	err := errors.New("plain error")
	if got := StackTrace(err); len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
	if got := FormatStackTrace(err); got != "" {
		t.Fatalf("expected empty, got %v", got)
	}
}

func TestStackTrace_Bad_Nil(t *testing.T) {
	if got := StackTrace(nil); len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
	if got := FormatStackTrace(nil); got != "" {
		t.Fatalf("expected empty, got %v", got)
	}
}

func TestStackTrace_Bad_NoOp(t *testing.T) {
	err := &Err{Msg: "no op"}
	if got := StackTrace(err); len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
	if got := FormatStackTrace(err); got != "" {
		t.Fatalf("expected empty, got %v", got)
	}
}

func TestStackTrace_Mixed_Good(t *testing.T) {
	err := E("inner", "msg", nil)
	err = fmt.Errorf("wrapper: %w", err)
	err = Wrap(err, "outer", "msg")

	stack := StackTrace(err)
	if want, got := []string{"outer", "inner"}, stack; !slices.Equal(want, got) {
		t.Fatalf("want %v, got %v", want, got)
	}
}
