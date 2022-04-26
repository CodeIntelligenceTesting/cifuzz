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

	// make sure filename exists
	filename := fmt.Sprintf("cifuzz_%s", runtime.GOOS)
	if runtime.GOOS == "windows" {
		filename = filename + ".exe"
	}
	executable := filepath.Join(cwd, filename)
	require.FileExistsf(t, executable, "make sure an executable of cifuzz is present under %s", executable)

	// create tempory directory for test
	dir, err := ioutil.TempDir("", "test")
	if err != nil {
		log.Fatal(err)
	}

	return executable, dir
}

func TestIntegration(t *testing.T) {
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
}
