package golog_test

import (
	"time"

	core "dappco.re/go"
	golog "dappco.re/go/log"
)

func ExampleErr_Error() {
	err := &golog.Err{Op: "agent.Run", Msg: "failed", Err: core.NewError("root")}
	if err.Error() == "" {
		panic("missing message")
	}
}

func ExampleErr_Unwrap() {
	root := core.NewError("root")
	err := &golog.Err{Err: root}
	if err.Unwrap() != root {
		panic("missing cause")
	}
}

func ExampleE() {
	err := core.MustCast[error](golog.E("agent.Run", "failed", nil))
	if golog.Op(err) != "agent.Run" {
		panic("missing op")
	}
}

func ExampleEWithRecovery() {
	delay := time.Second
	err := core.MustCast[error](golog.EWithRecovery("agent.Run", "temporary", nil, true, &delay, "retry"))
	if !golog.IsRetryable(err) {
		panic("missing retry")
	}
}

func ExampleWrap() {
	root := core.NewError("root")
	err := core.MustCast[error](golog.Wrap(root, "agent.Run", "failed"))
	if !golog.Is(err, root) {
		panic("missing cause")
	}
}

func ExampleWrapWithRecovery() {
	delay := time.Second
	root := core.NewError("root")
	err := core.MustCast[error](golog.WrapWithRecovery(root, "agent.Run", "temporary", true, &delay, "retry"))
	if golog.RecoveryAction(err) != "retry" {
		panic("missing action")
	}
}

func ExampleWrapCode() {
	root := core.NewError("root")
	err := core.MustCast[error](golog.WrapCode(root, "agent.failed", "agent.Run", "failed"))
	if golog.ErrCode(err) != "agent.failed" {
		panic("missing code")
	}
}

func ExampleWrapCodeWithRecovery() {
	delay := time.Second
	err := core.MustCast[error](golog.WrapCodeWithRecovery(nil, "agent.retry", "agent.Run", "temporary", true, &delay, "retry"))
	if golog.ErrCode(err) != "agent.retry" {
		panic("missing code")
	}
}

func ExampleNewCode() {
	err := core.MustCast[error](golog.NewCode("agent.failed", "failed"))
	if golog.ErrCode(err) != "agent.failed" {
		panic("missing code")
	}
}

func ExampleNewCodeWithRecovery() {
	delay := time.Second
	err := core.MustCast[error](golog.NewCodeWithRecovery("agent.retry", "retry", true, &delay, "retry"))
	if !golog.IsRetryable(err) {
		panic("missing retry")
	}
}

func ExampleRetryAfter() {
	delay := time.Second
	err := core.MustCast[error](golog.NewCodeWithRecovery("agent.retry", "retry", true, &delay, "retry"))
	got, ok := golog.RetryAfter(err)
	if !ok || *got != delay {
		panic("missing delay")
	}
}

func ExampleIsRetryable() {
	err := core.MustCast[error](golog.NewCodeWithRecovery("agent.retry", "retry", true, nil, "retry"))
	if !golog.IsRetryable(err) {
		panic("missing retry")
	}
}

func ExampleRecoveryAction() {
	err := core.MustCast[error](golog.NewCodeWithRecovery("agent.retry", "retry", true, nil, "inspect"))
	if golog.RecoveryAction(err) != "inspect" {
		panic("missing action")
	}
}

func ExampleIs() {
	root := core.NewError("root")
	err := core.MustCast[error](golog.Wrap(root, "agent.Run", "failed"))
	if !golog.Is(err, root) {
		panic("missing match")
	}
}

func ExampleAs() {
	err := core.MustCast[error](golog.E("agent.Run", "failed", nil))
	var typed *golog.Err
	if !golog.As(err, &typed) {
		panic("missing typed")
	}
}

func ExampleNewError() {
	err := core.MustCast[error](golog.NewError("simple"))
	if err.Error() != "simple" {
		panic("missing message")
	}
}

func ExampleJoin() {
	left := core.NewError("left")
	right := core.NewError("right")
	err := core.MustCast[error](golog.Join(left, right))
	if !golog.Is(err, left) || !golog.Is(err, right) {
		panic("missing join")
	}
}

func ExampleOp() {
	err := core.MustCast[error](golog.E("agent.Run", "failed", nil))
	if golog.Op(err) != "agent.Run" {
		panic("missing op")
	}
}

func ExampleErrCode() {
	err := core.MustCast[error](golog.NewCode("agent.failed", "failed"))
	if golog.ErrCode(err) != "agent.failed" {
		panic("missing code")
	}
}

func ExampleMessage() {
	err := core.MustCast[error](golog.E("agent.Run", "failed", nil))
	if golog.Message(err) != "failed" {
		panic("missing message")
	}
}

func ExampleRoot() {
	root := core.NewError("root")
	err := core.MustCast[error](golog.Wrap(root, "agent.Run", "failed"))
	got := core.MustCast[error](golog.Root(err))
	if got != root {
		panic("missing root")
	}
}

func ExampleAllOps() {
	err := core.MustCast[error](golog.Wrap(core.MustCast[error](golog.E("inner", "failed", nil)), "outer", "failed"))
	count := 0
	for range golog.AllOps(err) {
		count++
	}
	if count != 2 {
		panic("missing ops")
	}
}

func ExampleStackTrace() {
	err := core.MustCast[error](golog.Wrap(core.MustCast[error](golog.E("inner", "failed", nil)), "outer", "failed"))
	if len(golog.StackTrace(err)) != 2 {
		panic("missing stack")
	}
}

func ExampleFormatStackTrace() {
	err := core.MustCast[error](golog.Wrap(core.MustCast[error](golog.E("inner", "failed", nil)), "outer", "failed"))
	if golog.FormatStackTrace(err) != "outer -> inner" {
		panic("missing stack")
	}
}

func ExampleLogError() {
	original := golog.Default()
	buf := core.NewBuffer()
	golog.SetDefault(golog.New(golog.Options{Level: golog.LevelDebug, Output: buf}))
	defer golog.SetDefault(original)
	err := core.MustCast[error](golog.LogError(core.NewError("root"), "agent.Run", "failed"))
	if err == nil || !core.Contains(buf.String(), "[ERR]") {
		panic("missing failure")
	}
}

func ExampleLogWarn() {
	original := golog.Default()
	buf := core.NewBuffer()
	golog.SetDefault(golog.New(golog.Options{Level: golog.LevelDebug, Output: buf}))
	defer golog.SetDefault(original)
	err := core.MustCast[error](golog.LogWarn(core.NewError("root"), "agent.Run", "warn"))
	if err == nil || !core.Contains(buf.String(), "[WRN]") {
		panic("missing warn")
	}
}

func ExampleMust() {
	golog.Must(nil, "agent.Run", "ready")
}
