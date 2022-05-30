package cmdutils

import (
	"github.com/pkg/errors"
)

var ErrSilent = errors.New("SilentError")

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
