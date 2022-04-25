package cmdutils

import (
	"errors"
	"fmt"
)

var ErrSilent = errors.New("SilentError")

func WrapSilentError(err error) error {
	return fmt.Errorf("%w:\n%+v", ErrSilent, err)
}
