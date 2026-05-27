package errs

import "fmt"

// Kind classifies an application error and maps to a CLI exit code.
type Kind int

const (
	// KindNotFound is returned when lyrics are not found (exit code 1).
	KindNotFound Kind = iota + 1
	// KindNetwork is returned on transport-level failures (exit code 2).
	KindNetwork
	// KindRateLimited is returned when the API responds with HTTP 429 (exit code 3).
	KindRateLimited
	// KindInternal is returned for unexpected internal errors (exit code 4).
	KindInternal
	// KindBadInput is returned when the user supplies invalid arguments (exit code 5).
	KindBadInput
)

// ExitCode returns the process exit code that corresponds to k.
func (k Kind) ExitCode() int {
	return int(k)
}

// String returns a human-readable label for the kind.
func (k Kind) String() string {
	switch k {
	case KindNotFound:
		return "not_found"
	case KindNetwork:
		return "network"
	case KindRateLimited:
		return "rate_limited"
	case KindInternal:
		return "internal"
	case KindBadInput:
		return "bad_input"
	default:
		return "unknown"
	}
}

// AppError is the structured error type used throughout lrclib.
// Use errors.Is / errors.As to inspect; never compare error strings.
type AppError struct {
	Kind    Kind
	Message string
	Cause   error
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap exposes the underlying cause for errors.Is / errors.As.
func (e *AppError) Unwrap() error {
	return e.Cause
}

// Is reports whether target matches this error by Kind.
func (e *AppError) Is(target error) bool {
	t, ok := target.(*AppError)
	if !ok {
		return false
	}
	return e.Kind == t.Kind
}

// New creates an AppError with no cause.
func New(kind Kind, message string) *AppError {
	return &AppError{Kind: kind, Message: message}
}

// Wrap creates an AppError that wraps an existing error.
func Wrap(kind Kind, message string, cause error) *AppError {
	return &AppError{Kind: kind, Message: message, Cause: cause}
}

// NotFound returns a KindNotFound error.
func NotFound(message string) *AppError { return New(KindNotFound, message) }

// Network returns a KindNetwork error wrapping cause.
func Network(message string, cause error) *AppError { return Wrap(KindNetwork, message, cause) }

// RateLimited returns a KindRateLimited error.
func RateLimited(message string) *AppError { return New(KindRateLimited, message) }

// Internal returns a KindInternal error wrapping cause.
func Internal(message string, cause error) *AppError { return Wrap(KindInternal, message, cause) }

// BadInput returns a KindBadInput error.
func BadInput(message string) *AppError { return New(KindBadInput, message) }
