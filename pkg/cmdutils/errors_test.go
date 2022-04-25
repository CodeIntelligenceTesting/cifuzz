package cmdutils

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrapSilentError(t *testing.T) {
	errOriginal := errors.New("TestError")
	errSilent := WrapSilentError(errOriginal)

	assert.ErrorIs(t, errSilent, ErrSilent)
}
