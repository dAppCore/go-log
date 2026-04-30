package golog

import (
	"time"

	core "dappco.re/go"
)

func mustError(t *core.T, r core.Result) error {
	t.Helper()
	core.RequireTrue(t, r.OK)
	if r.Value == nil {
		return nil
	}
	err, ok := r.Value.(error)
	core.RequireTrue(t, ok)
	return err
}

func TestErrors_Err_Error_Good(t *core.T) {
	err := &Err{Op: "agent.Dispatch", Msg: "failed", Err: mustError(t, NewError("root")), Code: "agent.failed"}
	got := err.Error()
	core.AssertEqual(t, "agent.Dispatch: failed [agent.failed]: root", got)
}

func TestErrors_Err_Error_Bad(t *core.T) {
	err := &Err{}
	got := err.Error()
	core.AssertEqual(t, "", got)
}

func TestErrors_Err_Error_Ugly(t *core.T) {
	var err *Err
	got := err.Error()
	core.AssertEqual(t, "", got)
}

func TestErrors_Err_Unwrap_Good(t *core.T) {
	root := mustError(t, NewError("root"))
	err := &Err{Err: root}
	got := err.Unwrap()
	core.AssertEqual(t, root, got)
}

func TestErrors_Err_Unwrap_Bad(t *core.T) {
	err := &Err{}
	got := err.Unwrap()
	core.AssertNil(t, got)
}

func TestErrors_Err_Unwrap_Ugly(t *core.T) {
	root := mustError(t, NewError("root"))
	err := &Err{Err: mustError(t, Wrap(root, "outer", "failed"))}
	core.AssertTrue(t, Is(err.Unwrap(), root))
}

func TestErrors_E_Good(t *core.T) {
	root := mustError(t, NewError("root"))
	err := mustError(t, E("agent.Dispatch", "failed", root))
	core.AssertTrue(t, Is(err, root))
	core.AssertEqual(t, "agent.Dispatch", Op(err))
}

func TestErrors_E_Bad(t *core.T) {
	err := mustError(t, E("", "", nil))
	got := err.Error()
	core.AssertEqual(t, "", got)
}

func TestErrors_E_Ugly(t *core.T) {
	err := mustError(t, E("agent.Dispatch", "", mustError(t, NewError("root"))))
	got := err.Error()
	core.AssertEqual(t, "agent.Dispatch: root", got)
}

func TestErrors_EWithRecovery_Good(t *core.T) {
	retryAfter := 5 * time.Second
	err := mustError(t, EWithRecovery("agent.Dispatch", "retry", nil, true, &retryAfter, "retry later"))
	core.AssertTrue(t, IsRetryable(err))
	core.AssertEqual(t, "retry later", RecoveryAction(err))
}

func TestErrors_EWithRecovery_Bad(t *core.T) {
	err := mustError(t, EWithRecovery("agent.Dispatch", "permanent", nil, false, nil, ""))
	got, ok := RetryAfter(err)
	core.AssertFalse(t, IsRetryable(err))
	core.AssertFalse(t, ok)
	core.AssertNil(t, got)
}

func TestErrors_EWithRecovery_Ugly(t *core.T) {
	retryAfter := 10 * time.Millisecond
	err := mustError(t, EWithRecovery("", "", mustError(t, NewError("root")), true, &retryAfter, ""))
	got, ok := RetryAfter(err)
	core.AssertTrue(t, ok)
	core.AssertEqual(t, retryAfter, *got)
}

func TestErrors_Wrap_Good(t *core.T) {
	root := mustError(t, NewError("root"))
	err := mustError(t, Wrap(root, "agent.Dispatch", "failed"))
	core.AssertTrue(t, Is(err, root))
	core.AssertEqual(t, "agent.Dispatch", Op(err))
}

func TestErrors_Wrap_Bad(t *core.T) {
	err := mustError(t, Wrap(nil, "agent.Dispatch", "failed"))
	core.AssertNil(t, err)
	core.AssertNil(t, mustError(t, Root(err)))
}

func TestErrors_Wrap_Ugly(t *core.T) {
	inner := mustError(t, NewCode("agent.failed", "root"))
	err := mustError(t, Wrap(inner, "agent.Dispatch", "failed"))
	core.AssertEqual(t, "agent.failed", ErrCode(err))
}

func TestErrors_WrapWithRecovery_Good(t *core.T) {
	retryAfter := time.Second
	err := mustError(t, WrapWithRecovery(mustError(t, NewError("root")), "agent.Dispatch", "failed", true, &retryAfter, "retry"))
	core.AssertTrue(t, IsRetryable(err))
	core.AssertEqual(t, "retry", RecoveryAction(err))
}

func TestErrors_WrapWithRecovery_Bad(t *core.T) {
	err := mustError(t, WrapWithRecovery(nil, "agent.Dispatch", "failed", true, nil, "retry"))
	core.AssertNil(t, err)
	core.AssertFalse(t, IsRetryable(err))
}

func TestErrors_WrapWithRecovery_Ugly(t *core.T) {
	innerDelay := time.Second
	outerDelay := 2 * time.Second
	inner := mustError(t, NewCodeWithRecovery("inner", "root", true, &innerDelay, "inner"))
	err := mustError(t, WrapWithRecovery(inner, "outer", "failed", false, &outerDelay, "outer"))
	got, ok := RetryAfter(err)
	core.AssertTrue(t, ok)
	core.AssertEqual(t, outerDelay, *got)
}

func TestErrors_WrapCode_Good(t *core.T) {
	root := mustError(t, NewError("root"))
	err := mustError(t, WrapCode(root, "agent.failed", "agent.Dispatch", "failed"))
	core.AssertEqual(t, "agent.failed", ErrCode(err))
	core.AssertTrue(t, Is(err, root))
}

func TestErrors_WrapCode_Bad(t *core.T) {
	err := mustError(t, WrapCode(nil, "", "agent.Dispatch", "failed"))
	core.AssertNil(t, err)
	core.AssertEqual(t, "", ErrCode(err))
}

func TestErrors_WrapCode_Ugly(t *core.T) {
	err := mustError(t, WrapCode(nil, "agent.failed", "agent.Dispatch", "failed"))
	core.AssertNotNil(t, err)
	core.AssertEqual(t, "agent.failed", ErrCode(err))
}

func TestErrors_WrapCodeWithRecovery_Good(t *core.T) {
	retryAfter := time.Second
	err := mustError(t, WrapCodeWithRecovery(mustError(t, NewError("root")), "agent.failed", "agent.Dispatch", "failed", true, &retryAfter, "retry"))
	core.AssertEqual(t, "agent.failed", ErrCode(err))
	core.AssertTrue(t, IsRetryable(err))
}

func TestErrors_WrapCodeWithRecovery_Bad(t *core.T) {
	err := mustError(t, WrapCodeWithRecovery(nil, "", "agent.Dispatch", "failed", true, nil, "retry"))
	core.AssertNil(t, err)
	core.AssertEqual(t, "", RecoveryAction(err))
}

func TestErrors_WrapCodeWithRecovery_Ugly(t *core.T) {
	retryAfter := 3 * time.Second
	err := mustError(t, WrapCodeWithRecovery(nil, "agent.failed", "", "", true, &retryAfter, ""))
	got, ok := RetryAfter(err)
	core.AssertTrue(t, ok)
	core.AssertEqual(t, retryAfter, *got)
}

func TestErrors_NewCode_Good(t *core.T) {
	err := mustError(t, NewCode("agent.failed", "dispatch failed"))
	core.AssertEqual(t, "agent.failed", ErrCode(err))
	core.AssertEqual(t, "dispatch failed", Message(err))
}

func TestErrors_NewCode_Bad(t *core.T) {
	err := mustError(t, NewCode("", "dispatch failed"))
	got := ErrCode(err)
	core.AssertEqual(t, "", got)
}

func TestErrors_NewCode_Ugly(t *core.T) {
	err := mustError(t, NewCode("", ""))
	got := err.Error()
	core.AssertEqual(t, "", got)
}

func TestErrors_NewCodeWithRecovery_Good(t *core.T) {
	retryAfter := time.Minute
	err := mustError(t, NewCodeWithRecovery("agent.retry", "retry", true, &retryAfter, "retry"))
	core.AssertEqual(t, "agent.retry", ErrCode(err))
	core.AssertTrue(t, IsRetryable(err))
}

func TestErrors_NewCodeWithRecovery_Bad(t *core.T) {
	err := mustError(t, NewCodeWithRecovery("", "permanent", false, nil, ""))
	got, ok := RetryAfter(err)
	core.AssertFalse(t, IsRetryable(err))
	core.AssertFalse(t, ok)
	core.AssertNil(t, got)
}

func TestErrors_NewCodeWithRecovery_Ugly(t *core.T) {
	err := mustError(t, NewCodeWithRecovery("", "", true, nil, "inspect"))
	core.AssertTrue(t, IsRetryable(err))
	core.AssertEqual(t, "inspect", RecoveryAction(err))
}

func TestErrors_RetryAfter_Good(t *core.T) {
	delay := 42 * time.Second
	got, ok := RetryAfter(&Err{Msg: "retry", RetryAfter: &delay})
	core.AssertTrue(t, ok)
	core.AssertEqual(t, delay, *got)
}

func TestErrors_RetryAfter_Bad(t *core.T) {
	got, ok := RetryAfter(mustError(t, NewError("plain")))
	core.AssertFalse(t, ok)
	core.AssertNil(t, got)
}

func TestErrors_RetryAfter_Ugly(t *core.T) {
	delay := time.Nanosecond
	err := mustError(t, Wrap(&Err{Msg: "inner", RetryAfter: &delay}, "outer", "failed"))
	got, ok := RetryAfter(err)
	core.AssertTrue(t, ok)
	core.AssertEqual(t, delay, *got)
}

func TestErrors_IsRetryable_Good(t *core.T) {
	err := &Err{Msg: "retry", Retryable: true}
	got := IsRetryable(err)
	core.AssertTrue(t, got)
}

func TestErrors_IsRetryable_Bad(t *core.T) {
	err := &Err{Msg: "permanent"}
	got := IsRetryable(err)
	core.AssertFalse(t, got)
}

func TestErrors_IsRetryable_Ugly(t *core.T) {
	inner := &Err{Msg: "inner", Retryable: true}
	err := core.Errorf("wrapped: %w", inner)
	core.AssertTrue(t, IsRetryable(err))
}

func TestErrors_RecoveryAction_Good(t *core.T) {
	err := &Err{Msg: "recover", NextAction: "retry later"}
	got := RecoveryAction(err)
	core.AssertEqual(t, "retry later", got)
}

func TestErrors_RecoveryAction_Bad(t *core.T) {
	err := &Err{Msg: "no action"}
	got := RecoveryAction(err)
	core.AssertEqual(t, "", got)
}

func TestErrors_RecoveryAction_Ugly(t *core.T) {
	inner := &Err{Msg: "inner", NextAction: "inspect"}
	err := mustError(t, Wrap(inner, "outer", "failed"))
	core.AssertEqual(t, "inspect", RecoveryAction(err))
}

func TestErrors_Is_Good(t *core.T) {
	root := mustError(t, NewError("root"))
	err := mustError(t, Wrap(root, "outer", "failed"))
	core.AssertTrue(t, Is(err, root))
}

func TestErrors_Is_Bad(t *core.T) {
	left := mustError(t, NewError("left"))
	right := mustError(t, NewError("right"))
	core.AssertFalse(t, Is(left, right))
}

func TestErrors_Is_Ugly(t *core.T) {
	got := Is(nil, nil)
	other := mustError(t, NewError("other"))
	core.AssertTrue(t, got)
	core.AssertFalse(t, Is(nil, other))
}

func TestErrors_As_Good(t *core.T) {
	err := mustError(t, E("op", "msg", nil))
	var got *Err
	core.AssertTrue(t, As(err, &got))
	core.AssertEqual(t, "op", got.Op)
}

func TestErrors_As_Bad(t *core.T) {
	err := mustError(t, NewError("plain"))
	var got *Err
	core.AssertTrue(t, As(err, &got))
	core.AssertEqual(t, "plain", got.Msg)
}

func TestErrors_As_Ugly(t *core.T) {
	var got *Err
	matched := As(nil, &got)
	core.AssertFalse(t, matched)
	core.AssertNil(t, got)
}

func TestErrors_NewError_Good(t *core.T) {
	err := mustError(t, NewError("simple"))
	got := err.Error()
	core.AssertEqual(t, "simple", got)
}

func TestErrors_NewError_Bad(t *core.T) {
	err := mustError(t, NewError(""))
	got := err.Error()
	core.AssertEqual(t, "", got)
}

func TestErrors_NewError_Ugly(t *core.T) {
	err := mustError(t, NewError("line\nbreak"))
	got := err.Error()
	core.AssertEqual(t, "line\nbreak", got)
}

func TestErrors_Join_Good(t *core.T) {
	left := mustError(t, NewError("left"))
	right := mustError(t, NewError("right"))
	err := mustError(t, Join(left, right))
	core.AssertTrue(t, Is(err, left))
	core.AssertTrue(t, Is(err, right))
}

func TestErrors_Join_Bad(t *core.T) {
	err := mustError(t, Join(nil, nil))
	core.AssertNil(t, err)
	core.AssertNil(t, mustError(t, Root(err)))
}

func TestErrors_Join_Ugly(t *core.T) {
	root := mustError(t, NewError("root"))
	err := mustError(t, Join(nil, root))
	core.AssertTrue(t, Is(err, root))
}

func TestErrors_Op_Good(t *core.T) {
	err := mustError(t, Wrap(mustError(t, E("inner", "failed", nil)), "outer", "failed"))
	got := Op(err)
	core.AssertEqual(t, "outer", got)
}

func TestErrors_Op_Bad(t *core.T) {
	err := mustError(t, NewError("plain"))
	got := Op(err)
	core.AssertEqual(t, "", got)
}

func TestErrors_Op_Ugly(t *core.T) {
	got := Op(nil)
	blank := Op(&Err{Msg: "missing operation"})
	core.AssertEqual(t, "", got)
	core.AssertEqual(t, "", blank)
}

func TestErrors_ErrCode_Good(t *core.T) {
	err := mustError(t, Wrap(mustError(t, NewCode("agent.failed", "root")), "outer", "failed"))
	got := ErrCode(err)
	core.AssertEqual(t, "agent.failed", got)
}

func TestErrors_ErrCode_Bad(t *core.T) {
	err := mustError(t, E("op", "msg", nil))
	got := ErrCode(err)
	core.AssertEqual(t, "", got)
}

func TestErrors_ErrCode_Ugly(t *core.T) {
	got := ErrCode(nil)
	blank := ErrCode(&Err{Msg: "uncoded"})
	core.AssertEqual(t, "", got)
	core.AssertEqual(t, "", blank)
}

func TestErrors_Message_Good(t *core.T) {
	err := mustError(t, E("op", "the message", mustError(t, NewError("root"))))
	got := Message(err)
	core.AssertEqual(t, "the message", got)
}

func TestErrors_Message_Bad(t *core.T) {
	err := core.NewError("plain message")
	got := Message(err)
	core.AssertEqual(t, "plain message", got)
}

func TestErrors_Message_Ugly(t *core.T) {
	got := Message(nil)
	blank := Message(&Err{})
	core.AssertEqual(t, "", got)
	core.AssertEqual(t, "", blank)
}

func TestErrors_Root_Good(t *core.T) {
	root := mustError(t, NewError("root"))
	err := mustError(t, Wrap(mustError(t, Wrap(root, "inner", "failed")), "outer", "failed"))
	got := mustError(t, Root(err))
	core.AssertEqual(t, root, got)
}

func TestErrors_Root_Bad(t *core.T) {
	err := mustError(t, NewError("plain"))
	got := mustError(t, Root(err))
	core.AssertEqual(t, err, got)
}

func TestErrors_Root_Ugly(t *core.T) {
	left := mustError(t, NewError("left"))
	err := mustError(t, Join(left, mustError(t, NewError("right"))))
	got := mustError(t, Root(err))
	core.AssertEqual(t, left, got)
}

func TestErrors_AllOps_Good(t *core.T) {
	err := mustError(t, Wrap(mustError(t, E("inner", "failed", nil)), "outer", "failed"))
	var ops []string
	for op := range AllOps(err) {
		ops = append(ops, op)
	}
	core.AssertEqual(t, []string{"outer", "inner"}, ops)
}

func TestErrors_AllOps_Bad(t *core.T) {
	var ops []string
	for op := range AllOps(mustError(t, NewError("plain"))) {
		ops = append(ops, op)
	}
	core.AssertEqual(t, 0, len(ops))
}

func TestErrors_AllOps_Ugly(t *core.T) {
	err := mustError(t, Join(mustError(t, E("left", "failed", nil)), mustError(t, E("right", "failed", nil))))
	var ops []string
	for op := range AllOps(err) {
		ops = append(ops, op)
	}
	core.AssertEqual(t, []string{"left", "right"}, ops)
}

func TestErrors_StackTrace_Good(t *core.T) {
	err := mustError(t, Wrap(mustError(t, E("inner", "failed", nil)), "outer", "failed"))
	got := StackTrace(err)
	core.AssertEqual(t, []string{"outer", "inner"}, got)
}

func TestErrors_StackTrace_Bad(t *core.T) {
	got := StackTrace(mustError(t, NewError("plain")))
	core.AssertEqual(t, 0, len(got))
	core.AssertEqual(t, "", FormatStackTrace(mustError(t, NewError("plain"))))
}

func TestErrors_StackTrace_Ugly(t *core.T) {
	got := StackTrace(nil)
	core.AssertEqual(t, 0, len(got))
	core.AssertEqual(t, "", FormatStackTrace(nil))
}

func TestErrors_FormatStackTrace_Good(t *core.T) {
	err := mustError(t, Wrap(mustError(t, E("inner", "failed", nil)), "outer", "failed"))
	got := FormatStackTrace(err)
	core.AssertEqual(t, "outer -> inner", got)
}

func TestErrors_FormatStackTrace_Bad(t *core.T) {
	got := FormatStackTrace(mustError(t, NewError("plain")))
	core.AssertEqual(t, "", got)
	core.AssertEqual(t, 0, len(StackTrace(mustError(t, NewError("plain")))))
}

func TestErrors_FormatStackTrace_Ugly(t *core.T) {
	err := mustError(t, Join(mustError(t, E("left", "failed", nil)), mustError(t, E("right", "failed", nil))))
	got := FormatStackTrace(err)
	core.AssertEqual(t, "left -> right", got)
}

func TestErrors_LogError_Good(t *core.T) {
	original := Default()
	buf := core.NewBuffer()
	SetDefault(New(Options{Level: LevelDebug, Output: buf}))
	t.Cleanup(func() { SetDefault(original) })
	root := mustError(t, NewError("root"))
	err := mustError(t, LogError(root, "agent.Dispatch", "failed"))
	core.AssertTrue(t, Is(err, root))
	core.AssertContains(t, buf.String(), "[ERR] failed")
}

func TestErrors_LogError_Bad(t *core.T) {
	original := Default()
	buf := core.NewBuffer()
	SetDefault(New(Options{Level: LevelDebug, Output: buf}))
	t.Cleanup(func() { SetDefault(original) })
	err := mustError(t, LogError(nil, "agent.Dispatch", "failed"))
	core.AssertNil(t, err)
	core.AssertEqual(t, 0, buf.Len())
}

func TestErrors_LogError_Ugly(t *core.T) {
	original := Default()
	buf := core.NewBuffer()
	SetDefault(New(Options{Level: LevelDebug, Output: buf}))
	t.Cleanup(func() { SetDefault(original) })
	err := mustError(t, LogError(mustError(t, E("inner", "failed", mustError(t, NewError("root")))), "outer", "failed"))
	core.AssertEqual(t, "outer", Op(err))
	core.AssertContains(t, buf.String(), `stack="inner"`)
}

func TestErrors_LogWarn_Good(t *core.T) {
	original := Default()
	buf := core.NewBuffer()
	SetDefault(New(Options{Level: LevelDebug, Output: buf}))
	t.Cleanup(func() { SetDefault(original) })
	root := mustError(t, NewError("root"))
	err := mustError(t, LogWarn(root, "cache.Get", "miss"))
	core.AssertTrue(t, Is(err, root))
	core.AssertContains(t, buf.String(), "[WRN] miss")
}

func TestErrors_LogWarn_Bad(t *core.T) {
	original := Default()
	buf := core.NewBuffer()
	SetDefault(New(Options{Level: LevelDebug, Output: buf}))
	t.Cleanup(func() { SetDefault(original) })
	err := mustError(t, LogWarn(nil, "cache.Get", "miss"))
	core.AssertNil(t, err)
	core.AssertEqual(t, 0, buf.Len())
}

func TestErrors_LogWarn_Ugly(t *core.T) {
	original := Default()
	buf := core.NewBuffer()
	SetDefault(New(Options{Level: LevelDebug, Output: buf}))
	t.Cleanup(func() { SetDefault(original) })
	err := mustError(t, LogWarn(mustError(t, NewCode("cache.miss", "miss")), "cache.Get", "fallback"))
	core.AssertEqual(t, "cache.miss", ErrCode(err))
	core.AssertContains(t, buf.String(), "[WRN] fallback")
}

func TestErrors_Must_Good(t *core.T) {
	original := Default()
	buf := core.NewBuffer()
	SetDefault(New(Options{Level: LevelDebug, Output: buf}))
	t.Cleanup(func() { SetDefault(original) })
	Must(nil, "startup", "ready")
	core.AssertEqual(t, 0, buf.Len())
}

func TestErrors_Must_Bad(t *core.T) {
	original := Default()
	buf := core.NewBuffer()
	SetDefault(New(Options{Level: LevelDebug, Output: buf}))
	t.Cleanup(func() { SetDefault(original) })
	core.AssertPanics(t, func() { Must(mustError(t, NewError("boom")), "startup", "failed") })
	core.AssertContains(t, buf.String(), "[ERR] failed")
}

func TestErrors_Must_Ugly(t *core.T) {
	original := Default()
	buf := core.NewBuffer()
	SetDefault(New(Options{Level: LevelDebug, Output: buf}))
	t.Cleanup(func() { SetDefault(original) })
	var recovered any
	func() {
		defer func() { recovered = recover() }()
		Must(mustError(t, NewCode("startup.failed", "boom")), "startup", "failed")
	}()
	err, ok := recovered.(error)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "startup.failed", ErrCode(err))
	core.AssertContains(t, buf.String(), "[ERR] failed")
}
