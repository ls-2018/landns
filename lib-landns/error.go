package landns

import (
	"fmt"
	"strings"
)

// ErrorType is type of Error.
type ErrorType uint8

const (
	// TypeInternalError is a error for Landns internal error.
	TypeInternalError ErrorType = iota + 1

	// TypeExternalError is a error for the error caused from external libraries.
	TypeExternalError

	// TypeArgumentError is a error for invalid argument error.
	TypeArgumentError

	// TypeExpirationError is error for record has expired.
	TypeExpirationError
)

// String is converter to human readable string.
func (t ErrorType) String() string {
	switch t {
	case TypeInternalError:
		return "InternalError"
	case TypeExternalError:
		return "ExternalError"
	case TypeArgumentError:
		return "ArgumentError"
	case TypeExpirationError:
		return "ExpirationError"
	default:
		return "UnknownError"
	}
}

// Error is error type of Landns.
type Error struct {
	Type     ErrorType
	Original error
	Message  string
}

// newError is make new Error by format string.
func newError(typ ErrorType, original error, format string, args ...interface{}) Error {
	return Error{
		Type:     typ,
		Message:  fmt.Sprintf(format, args...),
		Original: original,
	}
}

// wrapError is make new Error if err is not nil.
func wrapError(original error, typ ErrorType, message string) error {
	if original == nil {
		return nil
	}
	return Error{
		Type:     typ,
		Message:  message,
		Original: original,
	}
}

// String is getter to human readable string.
func (e Error) String() string {
	return e.Error()
}

// Error is converter to human readable string.
func (e Error) Error() string {
	if e.Original == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Message, e.Original.Error())
}

// Unwrap is getter of original error.
func (e Error) Unwrap() error {
	return e.Original
}

// ErrorSet is list of errors.
type ErrorSet []error

// Error is getter for description string.
func (e ErrorSet) Error() string {
	xs := make([]string, len(e))
	for i, x := range e {
		xs[i] = x.Error()
	}
	return strings.Join(xs, "\n")
}
