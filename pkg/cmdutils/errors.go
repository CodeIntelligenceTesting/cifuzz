package cmdutils

import (
	"fmt"
	"github.com/pkg/errors"
)

var ErrSilent = errors.New("SilentError")

func WrapSilentError(err error) error {
	return fmt.Errorf("%w:\n%+v", ErrSilent, err)
}
