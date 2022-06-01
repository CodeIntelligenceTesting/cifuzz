package cmdutils

import (
	"fmt"
	"syscall"

	"github.com/pkg/errors"
)

var ErrSilent = SilentError{err: errors.New("SilentError")}

// SilentError indicates that the error message should not be printed
// when the error is handled.
type SilentError struct {
	err error
}

func (e SilentError) Error() string {
	return e.err.Error()
}

func (e SilentError) Unwrap() error {
	return e.err
}

// WrapSilentError wraps an existing error into a SilentError to avoid
// having the error message printed when the error is handled.
func WrapSilentError(err error) error {
	return &SilentError{err}
}

// IncorrectUsageError indicates that the command wasn't used correctly,
// for example because required arguments are missing.
// When an IncorrectUsageError is handled, the usage message should be
// printed.
type IncorrectUsageError struct {
	err error
}

func (e IncorrectUsageError) Error() string {
	return e.err.Error()
}

func (e IncorrectUsageError) Unwrap() error {
	return e.err
}

// WrapIncorrectUsageError wraps an existing error into a
// IncorrectUsageError to have the usage message printed when the error
// is handled.
func WrapIncorrectUsageError(err error) error {
	return &IncorrectUsageError{err}
}

func NewSignalError(signal syscall.Signal) *SignalError {
	return &SignalError{signal}
}

type SignalError struct {
	Signal syscall.Signal
}

func (e SignalError) Error() string {
	return fmt.Sprintf("terminated by signal %d (%s)", int(e.Signal), e.Signal.String())
}
