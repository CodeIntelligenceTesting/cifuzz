package utils

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func GetProjectRoot(t *testing.T) string {
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok, "unable to get project root from runtime")

	rootDir := strings.TrimSuffix(filename, "integration/utils/filepath.go")
	require.DirExists(t, rootDir)

	return rootDir
}

func GetFuzzTargetBuildDir(t *testing.T) string {
	rootDir := GetProjectRoot(t)

	fuzzTargetBuildPath := filepath.Join(rootDir, "testdata", "build")
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
