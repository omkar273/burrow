package errors

type Error struct {
	msg     string
	hint    string
	mark    error
	wrapped error
}

func New(msg string) *Error { return &Error{msg: msg} }

func Wrap(err error, msg string) *Error { return &Error{msg: msg, wrapped: err} }

func (e *Error) WithHint(hint string) *Error {
	e.hint = hint
	return e
}

// Mark lets callers branch with errors.Is instead of matching strings.
func (e *Error) Mark(sentinel error) *Error {
	e.mark = sentinel
	return e
}

func (e *Error) Hint() string { return e.hint }

func (e *Error) Error() string {
	if e.wrapped == nil {
		return e.msg
	}
	return e.msg + ": " + e.wrapped.Error()
}

func (e *Error) Unwrap() error { return e.wrapped }

func (e *Error) Is(target error) bool {
	return e.mark != nil && e.mark == target
}
