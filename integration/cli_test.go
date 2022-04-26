package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func prepareTestDir(t *testing.T) (string, string) {
	t.Helper()

	// get working directory
	cwd, err := os.Getwd()
	require.NoError(t, err)

	cifuzzName := fmt.Sprintf("cifuzz_%s", runtime.GOOS)
	if runtime.GOOS == "windows" {
		cifuzzName = cifuzzName + ".exe"
	}
	// The cwd of this test is always integration, so the root dir is one level up.
	cifuzzPath := filepath.Join(cwd, "..", "build", "bin", cifuzzName)
	require.FileExistsf(t, cifuzzPath, "cifuzz executable not present under %q", cifuzzPath)

	// create tempory directory for test
	dir, err := ioutil.TempDir("", "test")
	if err != nil {
		log.Fatal(err)
	}

	return cifuzzPath, dir
}

func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	executable, dir := prepareTestDir(t)

	//execute root command
	cmd := exec.Command(executable)
	cmd.Dir = dir
	err := cmd.Run()
	assert.NoError(t, err)

	// execute init command
	cmd = exec.Command(executable, "init")
	cmd.Dir = dir
	err = cmd.Run()
	assert.NoError(t, err)

	// execute create command
	cmd = exec.Command(executable, "create", "cpp")
	cmd.Dir = dir
	err = cmd.Run()
	assert.NoError(t, err)
}
