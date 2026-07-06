// Package golog provides structured logging and error handling for Core applications.
package golog

import (
	"iter"
	"time"

	core "dappco.re/go"
)

type singleUnwrapper interface {
	Unwrap() error
}

type multiUnwrapper interface {
	Unwrap() []error
}

// Err represents a structured error with operational context.
type Err struct {
	Op   string // Operation being performed (for example, "user.Save")
	Msg  string // Human-readable message
	Err  error  // Underlying error, when present
	Code string // Error code, when present
	// Retryable indicates whether the caller can safely retry this error.
	Retryable bool
	// RetryAfter suggests a delay before retrying when Retryable is true.
	RetryAfter *time.Duration
	// NextAction suggests an alternative path when this error is not directly retryable.
	NextAction string
}

// Error implements the error interface.
func (e *Err) Error() string {
	if e == nil {
		return ""
	}

	body := e.Msg
	if body == "" {
		if e.Code != "" {
			body = core.Concat("[", e.Code, "]")
		}
	} else if e.Code != "" {
		body = core.Concat(body, " [", e.Code, "]")
	}

	if e.Err != nil {
		if body != "" {
			body = core.Concat(body, ": ", e.Err.Error())
		} else {
			body = e.Err.Error()
		}
	}

	if e.Op != "" {
		if body != "" {
			return core.Concat(e.Op, ": ", body)
		}
		return e.Op
	}
	return body
}

// Unwrap returns the underlying error for standard error-chain inspection.
func (e *Err) Unwrap() (
	err error,
) {
	if e == nil {
		return nil
	}
	return e.Err
}

// E creates a new Err with operation context.
func E(op, msg string, err error) core.Result {
	return core.Fail(&Err{Op: op, Msg: msg, Err: err})
}

// EWithRecovery creates a new Err with operation context and recovery metadata.
func EWithRecovery(op, msg string, err error, retryable bool, retryAfter *time.Duration, nextAction string) core.Result {
	recoveryErr := &Err{
		Op:  op,
		Msg: msg,
		Err: err,
	}
	inheritRecovery(recoveryErr, err)
	recoveryErr.Retryable = retryable
	recoveryErr.RetryAfter = retryAfter
	recoveryErr.NextAction = nextAction
	return core.Fail(recoveryErr)
}

// Wrap wraps an error with operation context.
func Wrap(err error, op, msg string) core.Result {
	if err == nil {
		return core.Ok(nil)
	}
	wrapped := &Err{Op: op, Msg: msg, Err: err, Code: inheritedCode(err)}
	inheritRecovery(wrapped, err)
	return core.Fail(wrapped)
}

// WrapWithRecovery wraps an error with operation context and explicit recovery metadata.
func WrapWithRecovery(err error, op, msg string, retryable bool, retryAfter *time.Duration, nextAction string) core.Result {
	if err == nil {
		return core.Ok(nil)
	}
	recoveryErr := &Err{
		Op:   op,
		Msg:  msg,
		Err:  err,
		Code: ErrCode(err),
	}
	inheritRecovery(recoveryErr, err)
	recoveryErr.Retryable = retryable
	recoveryErr.RetryAfter = retryAfter
	recoveryErr.NextAction = nextAction
	return core.Fail(recoveryErr)
}

// WrapCode wraps an error with operation context and error code.
func WrapCode(err error, code, op, msg string) core.Result {
	if err == nil && code == "" {
		return core.Ok(nil)
	}
	wrapped := &Err{Op: op, Msg: msg, Err: err, Code: code}
	inheritRecovery(wrapped, err)
	return core.Fail(wrapped)
}

// WrapCodeWithRecovery wraps an error with operation context, code, and recovery metadata.
func WrapCodeWithRecovery(err error, code, op, msg string, retryable bool, retryAfter *time.Duration, nextAction string) core.Result {
	if err == nil && code == "" {
		return core.Ok(nil)
	}
	recoveryErr := &Err{
		Op:   op,
		Msg:  msg,
		Err:  err,
		Code: code,
	}
	inheritRecovery(recoveryErr, err)
	recoveryErr.Retryable = retryable
	recoveryErr.RetryAfter = retryAfter
	recoveryErr.NextAction = nextAction
	return core.Fail(recoveryErr)
}

// NewCode creates an error with just code and message.
func NewCode(code, msg string) core.Result {
	return core.Fail(&Err{Msg: msg, Code: code})
}

// NewCodeWithRecovery creates a coded error with recovery metadata.
func NewCodeWithRecovery(code, msg string, retryable bool, retryAfter *time.Duration, nextAction string) core.Result {
	return core.Fail(&Err{
		Msg:        msg,
		Code:       code,
		Retryable:  retryable,
		RetryAfter: retryAfter,
		NextAction: nextAction,
	})
}

func inheritRecovery(
	dst *Err,
	err error,
) {
	if err == nil || dst == nil {
		return
	}
	walkErrorTree(err, func(current error) bool {
		source, ok := current.(*Err)
		if !ok {
			return true
		}
		if !source.hasRecovery() {
			return true
		}
		dst.Retryable = source.Retryable
		dst.RetryAfter = source.RetryAfter
		dst.NextAction = source.NextAction
		return false
	})
}

func inheritedCode(err error) string {
	var code string
	walkErrorTree(err, func(current error) bool {
		wrapped, ok := current.(*Err)
		if !ok || wrapped.Code == "" {
			return true
		}
		code = wrapped.Code
		return false
	})
	return code
}

// RetryAfter returns the first retry-after hint from an error chain, if present.
func RetryAfter(err error) (*time.Duration, bool) {
	var retryAfter *time.Duration
	var ok bool
	walkErrorTree(err, func(current error) bool {
		wrapped, match := current.(*Err)
		if !match || wrapped.RetryAfter == nil {
			return true
		}
		retryAfter = wrapped.RetryAfter
		ok = true
		return false
	})
	return retryAfter, ok
}

// IsRetryable reports whether the error chain contains a retryable Err.
func IsRetryable(err error) bool {
	var retryable bool
	walkErrorTree(err, func(current error) bool {
		wrapped, ok := current.(*Err)
		if !ok || !wrapped.Retryable {
			return true
		}
		retryable = true
		return false
	})
	return retryable
}

// RecoveryAction returns the first next action from an error chain.
func RecoveryAction(err error) string {
	var nextAction string
	walkErrorTree(err, func(current error) bool {
		wrapped, ok := current.(*Err)
		if !ok || wrapped.NextAction == "" {
			return true
		}
		nextAction = wrapped.NextAction
		return false
	})
	return nextAction
}

func retryableHint(err error) bool {
	return IsRetryable(err)
}

// Is reports whether any error in err's tree matches target.
func Is(err, target error) bool {
	return core.Is(err, target)
}

// As finds the first error in err's tree that matches target.
func As(err error, target any) bool {
	return core.As(err, target)
}

// NewError creates a simple error with the given text.
func NewError(text string) core.Result {
	return core.Fail(&Err{Msg: text})
}

// Join combines multiple errors into one.
// Returns Ok(nil) when every input is nil (nothing to join), Fail(joined)
// otherwise.
func Join(errs ...error) core.Result {
	joined := core.ErrorJoin(errs...)
	if joined == nil {
		return core.Ok(nil)
	}
	return core.Fail(joined)
}

// Op extracts the operation name from an error.
func Op(err error) string {
	var op string
	walkErrorTree(err, func(current error) bool {
		wrapped, ok := current.(*Err)
		if !ok || wrapped.Op == "" {
			return true
		}
		op = wrapped.Op
		return false
	})
	return op
}

// ErrCode extracts the error code from an error.
func ErrCode(err error) string {
	return inheritedCode(err)
}

// Message extracts the message from an error.
func Message(err error) string {
	if err == nil {
		return ""
	}
	var msg string
	walkErrorTree(err, func(current error) bool {
		wrapped, ok := current.(*Err)
		if !ok || wrapped.Msg == "" {
			return true
		}
		msg = wrapped.Msg
		return false
	})
	if msg != "" {
		return msg
	}
	return err.Error()
}

// Root returns the root cause of an error chain.
func Root(err error) core.Result {
	if err == nil {
		return core.Ok(nil)
	}
	switch current := any(err).(type) {
	case multiUnwrapper:
		children := current.Unwrap()
		if len(children) == 0 {
			return core.Ok(err)
		}
		return Root(children[0])
	case singleUnwrapper:
		unwrapped := current.Unwrap()
		if unwrapped == nil {
			return core.Ok(err)
		}
		return Root(unwrapped)
	default:
		return core.Ok(err)
	}
}

// AllOps returns an iterator over all operational contexts in the error chain.
func AllOps(err error) iter.Seq[string] {
	return func(yield func(string) bool) {
		walkErrorTree(err, func(current error) bool {
			if e, ok := current.(*Err); ok && e.Op != "" {
				return yield(e.Op)
			}
			return true
		})
	}
}

// StackTrace returns the logical stack trace from an error.
func StackTrace(err error) []string {
	var stack []string
	for op := range AllOps(err) {
		stack = append(stack, op)
	}
	return stack
}

// FormatStackTrace returns a pretty-printed logical stack trace.
func FormatStackTrace(err error) string {
	var ops []string
	for op := range AllOps(err) {
		ops = append(ops, op)
	}
	if len(ops) == 0 {
		return ""
	}
	return core.Join(" -> ", ops...)
}

// LogError logs at Error level and returns a wrapped error result.
func LogError(err error, op, msg string) core.Result {
	if err == nil {
		return core.Ok(nil)
	}
	wrapped := Wrap(err, op, msg)
	Default().Error(msg, "op", op, "err", err)
	return wrapped
}

func walkErrorTree(err error, visit func(error) bool) {
	if err == nil {
		return
	}
	if !visit(err) {
		return
	}
	switch current := any(err).(type) {
	case multiUnwrapper:
		for _, child := range current.Unwrap() {
			walkErrorTree(child, visit)
		}
	case singleUnwrapper:
		walkErrorTree(current.Unwrap(), visit)
	}
}

func (e *Err) hasRecovery() bool {
	return e != nil && (e.Retryable || e.RetryAfter != nil || e.NextAction != "")
}

// LogWarn logs at Warn level and returns a wrapped error result.
func LogWarn(err error, op, msg string) core.Result {
	if err == nil {
		return core.Ok(nil)
	}
	wrapped := Wrap(err, op, msg)
	Default().Warn(msg, "op", op, "err", err)
	return wrapped
}

// Must panics if err is not nil, logging first.
func Must(err error, op, msg string) {
	if err != nil {
		wrapped := Wrap(err, op, msg)
		Default().Error(msg, "op", op, "err", err)
		panic(wrapped.Value)
	}
}
