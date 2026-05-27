// Package errs defines the error kinds and structured error type used
// throughout lrclib. All errors returned from internal packages should
// be wrapped in AppError so callers can inspect Kind and choose the
// appropriate exit code.
package errs
