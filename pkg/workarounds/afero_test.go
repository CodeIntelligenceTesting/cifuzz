package workarounds

import (
	"os"
	"syscall"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestIsPermission(t *testing.T) {
	err := os.ErrPermission
	assert.True(t, IsPermission(err))

	errWrapped := errors.Wrap(err, "wrapped")
	assert.True(t, IsPermission(errWrapped))
}

func TestIsPermission_Syscall(t *testing.T) {
	err := syscall.EPERM
	assert.True(t, IsPermission(err))
	errWrapped := errors.Wrap(err, "wrapped")
	assert.True(t, IsPermission(errWrapped))
}
