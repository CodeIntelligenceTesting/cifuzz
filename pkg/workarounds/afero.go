package workarounds

import (
	"os"
	"runtime"
	"syscall"

	"github.com/pkg/errors"
)

// IsPermission is a wrapper for os.IsPermission() to handle a bug present in afero
// see: https://github.com/spf13/afero/issues/350
// TODO should be removed after this PR is merged https://github.com/spf13/afero/pull/351
func IsPermission(err error) bool {
	if runtime.GOOS == "windows" && (err == syscall.EPERM || errors.Cause(err) == syscall.EPERM) {
		return true
	}
	return os.IsPermission(errors.Cause(err))
}
