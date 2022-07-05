package integration_tests

import (
	"os"
	"os/exec"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

var buildMutex sync.Mutex

func BuildFuzzTarget(t *testing.T, name string) {
	buildMutex.Lock()
	defer buildMutex.Unlock()

	cmd := exec.Command("make", "-C", TestDataDir(t), "fuzz-targets/"+name)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	require.NoError(t, err)
}
