package integration_tests

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/otiai10/copy"
	"github.com/stretchr/testify/require"
)

var baseTempDir string
var testDataDir string
var createTestDataDirOnce sync.Once

func TestDataDir(t *testing.T) string {
	createTestDataDirOnce.Do(func() {
		var err error
		_, filename, _, ok := runtime.Caller(0)
		require.True(t, ok, "unable to get filename from runtime")
		srcDir := filepath.Join(filepath.Dir(filename), "testdata")
		testDataDir, err = os.MkdirTemp(baseTempDir, "testdata-")
		require.NoError(t, err)
		err = copy.Copy(srcDir, testDataDir)
		require.NoError(t, err)
	})
	return testDataDir
}

func BuildDir(t *testing.T) string {
	return filepath.Join(TestDataDir(t), "build")
}

func FuzzTestExecutablePath(t *testing.T, testDataDir, fuzzTest string) string {
	if runtime.GOOS == "windows" {
		fuzzTest += ".exe"
	}
	return filepath.Join(BuildDir(t), fuzzTest)
}
