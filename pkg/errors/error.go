package errors

type Error struct {
	Code    Code
	Message string
	Err     error
}

func (e *Error) Error() string {
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Err
}


// New creates a new business error.
func New(code Code, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// Wrap wraps an existing error.
func Wrap(code Code, message string, err error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

func NewConfigMissing(field string) *Error {
    return &Error{
        Code:    CodeConfigMissing,
        Message: field + " is required",
    }
}

func NewConfigInvalid(field string, reason string) *Error {
	return &Error{
		Code:    CodeConfigInvalid,
		Message: field + " " + reason,
	}
}