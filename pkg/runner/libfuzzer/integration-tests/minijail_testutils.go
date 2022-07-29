package integration_tests

import (
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"code-intelligence.com/cifuzz/pkg/install"
	"code-intelligence.com/cifuzz/pkg/runfiles"
)

var bundler *install.InstallationBundler
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
			err := bundler.BuildMinijail()
			require.NoError(t, err)
			err = bundler.BuildProcessWrapper()
			require.NoError(t, err)
		})
		installMutex.Unlock()

		runfiles.Finder = runfiles.RunfilesFinderImpl{InstallDir: bundler.TargetDir}

		f(t, false)
	})
}
