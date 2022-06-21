package testutils

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func BuildFuzzTarget(t *testing.T, name string) {
	testDataDir := GetTestDataDir(t)
	cmd := exec.Command("make", "-C", testDataDir, "fuzz-targets/"+name)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	require.NoError(t, err)
}
