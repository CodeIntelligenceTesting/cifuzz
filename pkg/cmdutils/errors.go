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
