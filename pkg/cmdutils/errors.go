package cmdutils

import (
	"fmt"

	"github.com/pkg/errors"
)

var ErrSilent = errors.New("SilentError")

// WrapSilentError wraps an existing error into ErrSilent
// so we can silently fail if an expected error happens
func WrapSilentError(err error) error {
	return fmt.Errorf("%w:\n%+v", ErrSilent, err)
}

var ErrIncorrectUsage = errors.New("IncorrectUsageError")

// WrapIncorrectUsageError wraps an existing error into
// ErrIncorrectUsage so we can print a usage message when this error
// is handled.
func WrapIncorrectUsageError(err error) error {
	return fmt.Errorf("%w:\n%+v", ErrIncorrectUsage, err)
}
