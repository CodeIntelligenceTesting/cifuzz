package testutils

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func GetTestDataDir(t *testing.T) string {
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok, "unable to get filename from runtime")
	return filepath.Join(filepath.Dir(filepath.Dir(filename)), "testdata")
}

func GetFuzzTargetBuildDir(t *testing.T) string {
	testDataDir := GetTestDataDir(t)
	fuzzTargetBuildPath := filepath.Join(testDataDir, "build")
	require.DirExists(t, fuzzTargetBuildPath)
	return fuzzTargetBuildPath
}

func GetFuzzTargetPath(t *testing.T, fuzzTarget string) string {
	if runtime.GOOS == "windows" {
		fuzzTarget += ".exe"
	}
	fuzzTargetPath := filepath.Join(GetFuzzTargetBuildDir(t), fuzzTarget)
	require.FileExists(t, fuzzTargetPath)

	return fuzzTargetPath
}
