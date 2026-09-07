package errors

// Error carries a message, an optional operator-facing hint, an optional
// sentinel mark for errors.Is, and an optional wrapped cause.
type Error struct {
	msg     string
	hint    string
	mark    error
	wrapped error
}

// New starts a new error with the given message.
func New(msg string) *Error { return &Error{msg: msg} }

// Wrap starts a new error that preserves an underlying cause.
func Wrap(err error, msg string) *Error { return &Error{msg: msg, wrapped: err} }

// WithHint attaches operator-facing guidance: what to do about this.
func (e *Error) WithHint(hint string) *Error {
	e.hint = hint
	return e
}

// Mark classifies the error against a sentinel so callers can branch on
// errors.Is without string matching.
func (e *Error) Mark(sentinel error) *Error {
	e.mark = sentinel
	return e
}

// Hint returns the operator-facing guidance, or "" if none was set.
func (e *Error) Hint() string { return e.hint }

func (e *Error) Error() string {
	if e.wrapped == nil {
		return e.msg
	}
	return e.msg + ": " + e.wrapped.Error()
}

// Unwrap exposes the underlying cause to errors.Is and errors.As.
func (e *Error) Unwrap() error { return e.wrapped }

// Is reports whether this error was marked with target.
func (e *Error) Is(target error) bool {
	return e.mark != nil && e.mark == target
}
