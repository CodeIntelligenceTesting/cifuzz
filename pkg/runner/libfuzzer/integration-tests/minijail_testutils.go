package integration_tests

import (
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"code-intelligence.com/cifuzz/pkg/runfiles"
	"code-intelligence.com/cifuzz/tools/install"
)

var installer *install.Installer
var installOnce sync.Once
var installMutex sync.Mutex

func TestWithAndWithoutMinijail(t *testing.T, f func(t *testing.T, disableMinijail bool)) {
	t.Run("WithoutMinijail", func(t *testing.T) {
		t.Parallel()
		f(t, true)
	})
	t.Run("WithMinijail", func(t *testing.T) {
		// we support minijail for linux only
		if runtime.GOOS != "linux" {
			t.Skip()
		}
		t.Parallel()

		installMutex.Lock()
		installOnce.Do(func() {
			err := installer.InstallMinijail()
			require.NoError(t, err)
			err = installer.InstallProcessWrapper()
			require.NoError(t, err)
		})
		installMutex.Unlock()

		runfiles.Finder = runfiles.RunfilesFinderImpl{InstallDir: installer.InstallDir}

		f(t, false)
	})
}
