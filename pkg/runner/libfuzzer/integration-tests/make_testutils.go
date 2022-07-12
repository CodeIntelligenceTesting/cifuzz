package integration_tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/otiai10/copy"
	"github.com/stretchr/testify/require"
)

var buildMutex sync.Mutex

func BuildFuzzTarget(t *testing.T, name string) string {
	buildMutex.Lock()
	defer buildMutex.Unlock()

	cmd := exec.Command("make", "-C", TestDataDir(t), "fuzz-targets/"+name)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	require.NoError(t, err)

	// Create a temporary build directory to avoid that parallel runs
	// access and write to the same build directory
	buildDir := TempBuildDir(t)
	err = copy.Copy(filepath.Join(TestDataDir(t), "build"), buildDir)
	require.NoError(t, err)

	return buildDir
}
