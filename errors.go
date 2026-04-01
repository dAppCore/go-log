// Package log provides structured logging and error handling for Core applications.
//
// This file implements structured error types and combined log-and-return helpers
// that simplify common error handling patterns.

package log

import (
	"errors"
	"iter"
	"strings"
	"time"
)

// Err represents a structured error with operational context.
// It implements the error interface and supports unwrapping.
type Err struct {
	Op   string // Operation being performed (e.g., "user.Save")
	Msg  string // Human-readable message
	Err  error  // Underlying error (optional)
	Code string // Error code (optional, e.g., "VALIDATION_FAILED")
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
			body = "[" + e.Code + "]"
		}
	} else if e.Code != "" {
		body += " [" + e.Code + "]"
	}

	if e.Err != nil {
		if body != "" {
			body += ": " + e.Err.Error()
		} else {
			body = e.Err.Error()
		}
	}

	if e.Op != "" {
		if body != "" {
			return e.Op + ": " + body
		}
		return e.Op
	}
	return body
}

// Unwrap returns the underlying error for use with errors.Is and errors.As.
func (e *Err) Unwrap() error {
	return e.Err
}

// --- Error Creation Functions ---

// E creates a new Err with operation context.
// The underlying error can be nil for creating errors without a cause.
//
// Example:
//
//	return log.E("user.Save", "failed to save user", err)
//	return log.E("api.Call", "rate limited", nil)  // No underlying cause
func E(op, msg string, err error) error {
	return &Err{Op: op, Msg: msg, Err: err}
}

// EWithRecovery creates a new Err with operation context and recovery metadata.
//
//	return log.EWithRecovery("api.Call", "temporary failure", err, true, &retryAfter, "retry with backoff")
func EWithRecovery(op, msg string, err error, retryable bool, retryAfter *time.Duration, nextAction string) error {
	recoveryErr := &Err{
		Op:  op,
		Msg: msg,
		Err: err,
	}
	inheritRecovery(recoveryErr, err)
	recoveryErr.Retryable = retryable
	recoveryErr.RetryAfter = retryAfter
	recoveryErr.NextAction = nextAction
	return recoveryErr
}

// Wrap wraps an error with operation context.
// Returns nil if err is nil, to support conditional wrapping.
// Preserves error Code if the wrapped error is an *Err.
//
// Example:
//
//	return log.Wrap(err, "db.Query", "database query failed")
func Wrap(err error, op, msg string) error {
	if err == nil {
		return nil
	}
	wrapped := &Err{Op: op, Msg: msg, Err: err, Code: ErrCode(err)}
	inheritRecovery(wrapped, err)
	return wrapped
}

// WrapWithRecovery wraps an error with operation context and explicit recovery metadata.
//
//	return log.WrapWithRecovery(err, "api.Call", "temporary failure", true, &retryAfter, "retry with backoff")
func WrapWithRecovery(err error, op, msg string, retryable bool, retryAfter *time.Duration, nextAction string) error {
	if err == nil {
		return nil
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
	return recoveryErr
}

// WrapCode wraps an error with operation context and error code.
// Returns nil only if both err is nil AND code is empty.
// Useful for API errors that need machine-readable codes.
//
// Example:
//
//	return log.WrapCode(err, "VALIDATION_ERROR", "user.Validate", "invalid email")
func WrapCode(err error, code, op, msg string) error {
	if err == nil && code == "" {
		return nil
	}
	wrapped := &Err{Op: op, Msg: msg, Err: err, Code: code}
	inheritRecovery(wrapped, err)
	return wrapped
}

// WrapCodeWithRecovery wraps an error with operation context, code, and recovery metadata.
//
//	return log.WrapCodeWithRecovery(err, "TEMPORARY_UNAVAILABLE", "api.Call", "temporary failure", true, &retryAfter, "retry with backoff")
func WrapCodeWithRecovery(err error, code, op, msg string, retryable bool, retryAfter *time.Duration, nextAction string) error {
	if err == nil && code == "" {
		return nil
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
	return recoveryErr
}

// NewCode creates an error with just code and message (no underlying error).
// Useful for creating sentinel errors with codes.
//
// Example:
//
//	var ErrNotFound = log.NewCode("NOT_FOUND", "resource not found")
func NewCode(code, msg string) error {
	return &Err{Msg: msg, Code: code}
}

// NewCodeWithRecovery creates a coded error with recovery metadata.
//
//	var ErrTemporary = log.NewCodeWithRecovery("TEMPORARY_UNAVAILABLE", "temporary failure", true, &retryAfter, "retry with backoff")
func NewCodeWithRecovery(code, msg string, retryable bool, retryAfter *time.Duration, nextAction string) error {
	return &Err{
		Msg:        msg,
		Code:       code,
		Retryable:  retryable,
		RetryAfter: retryAfter,
		NextAction: nextAction,
	}
}

// inheritRecovery copies recovery metadata from the first *Err in err's chain.
func inheritRecovery(dst *Err, err error) {
	if err == nil || dst == nil {
		return
	}
	var source *Err
	if As(err, &source) {
		dst.Retryable = source.Retryable
		dst.RetryAfter = source.RetryAfter
		dst.NextAction = source.NextAction
	}
}

// RetryAfter returns the first retry-after hint from an error chain, if present.
//
//	retryAfter, ok := log.RetryAfter(err)
func RetryAfter(err error) (*time.Duration, bool) {
	for err != nil {
		if wrapped, ok := err.(*Err); ok && wrapped.RetryAfter != nil {
			return wrapped.RetryAfter, true
		}
		err = errors.Unwrap(err)
	}
	return nil, false
}

// IsRetryable reports whether the error chain contains a retryable Err.
//
//	if log.IsRetryable(err) { /* retry the operation */ }
func IsRetryable(err error) bool {
	var wrapped *Err
	if As(err, &wrapped) {
		return wrapped.Retryable
	}
	return false
}

// RecoveryAction returns the first next action from an error chain.
//
//	next := log.RecoveryAction(err)
func RecoveryAction(err error) string {
	for err != nil {
		if wrapped, ok := err.(*Err); ok && wrapped.NextAction != "" {
			return wrapped.NextAction
		}
		err = errors.Unwrap(err)
	}
	return ""
}

func retryableHint(err error) bool {
	for err != nil {
		if wrapped, ok := err.(*Err); ok && wrapped.Retryable {
			return true
		}
		err = errors.Unwrap(err)
	}
	return false
}

// --- Standard Library Wrappers ---

// Is reports whether any error in err's tree matches target.
// Wrapper around errors.Is for convenience.
//
//	if log.Is(err, context.DeadlineExceeded) { /* handle timeout */ }
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As finds the first error in err's tree that matches target.
// Wrapper around errors.As for convenience.
//
//	var e *log.Err
//	if log.As(err, &e) { /* use e.Code */ }
func As(err error, target any) bool {
	return errors.As(err, target)
}

// NewError creates a simple error with the given text.
// Wrapper around errors.New for convenience.
//
//	return log.NewError("invalid state")
func NewError(text string) error {
	return errors.New(text)
}

// Join combines multiple errors into one.
// Wrapper around errors.Join for convenience.
//
//	return log.Join(validateErr, persistErr)
func Join(errs ...error) error {
	return errors.Join(errs...)
}

// --- Error Introspection Helpers ---

// Op extracts the operation name from an error.
// Returns empty string if the error is not an *Err.
//
//	op := log.Op(err) // e.g. "user.Save"
func Op(err error) string {
	var e *Err
	if As(err, &e) {
		return e.Op
	}
	return ""
}

// ErrCode extracts the error code from an error.
// Returns empty string if the error is not an *Err or has no code.
//
//	code := log.ErrCode(err) // e.g. "VALIDATION_FAILED"
func ErrCode(err error) string {
	var e *Err
	if As(err, &e) {
		return e.Code
	}
	return ""
}

// Message extracts the message from an error.
// Returns the error's Error() string if not an *Err.
//
//	msg := log.Message(err)
func Message(err error) string {
	if err == nil {
		return ""
	}
	var e *Err
	if As(err, &e) {
		return e.Msg
	}
	return err.Error()
}

// Root returns the root cause of an error chain.
// Unwraps until no more wrapped errors are found.
//
//	cause := log.Root(err)
func Root(err error) error {
	if err == nil {
		return nil
	}
	for {
		unwrapped := errors.Unwrap(err)
		if unwrapped == nil {
			return err
		}
		err = unwrapped
	}
}

// AllOps returns an iterator over all operational contexts in the error chain.
// It traverses the error tree using errors.Unwrap.
//
//	for op := range log.AllOps(err) { /* "api.Call" → "db.Query" → ... */ }
func AllOps(err error) iter.Seq[string] {
	return func(yield func(string) bool) {
		for err != nil {
			if e, ok := err.(*Err); ok {
				if e.Op != "" {
					if !yield(e.Op) {
						return
					}
				}
			}
			err = errors.Unwrap(err)
		}
	}
}

// StackTrace returns the logical stack trace (chain of operations) from an error.
// It returns an empty slice if no operational context is found.
//
//	ops := log.StackTrace(err) // ["api.Call", "db.Query", "sql.Exec"]
func StackTrace(err error) []string {
	var stack []string
	for op := range AllOps(err) {
		stack = append(stack, op)
	}
	return stack
}

// FormatStackTrace returns a pretty-printed logical stack trace.
//
//	trace := log.FormatStackTrace(err) // "api.Call -> db.Query -> sql.Exec"
func FormatStackTrace(err error) string {
	var ops []string
	for op := range AllOps(err) {
		ops = append(ops, op)
	}
	if len(ops) == 0 {
		return ""
	}
	return strings.Join(ops, " -> ")
}

// --- Combined Log-and-Return Helpers ---

// LogError logs an error at Error level and returns a wrapped error.
// Reduces boilerplate in error handling paths.
//
// Example:
//
//	// Before
//	if err != nil {
//	    log.Error("failed to save", "err", err)
//	    return errors.Wrap(err, "user.Save", "failed to save")
//	}
//
//	// After
//	if err != nil {
//	    return log.LogError(err, "user.Save", "failed to save")
//	}
func LogError(err error, op, msg string) error {
	if err == nil {
		return nil
	}
	wrapped := Wrap(err, op, msg)
	Default().Error(msg, "op", op, "err", err)
	return wrapped
}

// LogWarn logs at Warn level and returns a wrapped error.
// Use for recoverable errors that should be logged but not treated as critical.
//
// Example:
//
//	return log.LogWarn(err, "cache.Get", "cache miss, falling back to db")
func LogWarn(err error, op, msg string) error {
	if err == nil {
		return nil
	}
	wrapped := Wrap(err, op, msg)
	Default().Warn(msg, "op", op, "err", err)
	return wrapped
}

// Must panics if err is not nil, logging first.
// Use for errors that should never happen and indicate programmer error.
//
// Example:
//
//	log.Must(Initialize(), "app", "startup failed")
func Must(err error, op, msg string) {
	if err != nil {
		wrapped := Wrap(err, op, msg)
		Default().Error(msg, "op", op, "err", err)
		panic(wrapped)
	}
}
