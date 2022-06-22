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

	"code-intelligence.com/cifuzz/util/fileutil"
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

func TestIntegration_InitCreate(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	executable, dir := prepareTestDir(t)
	defer fileutil.Cleanup(dir)
	fmt.Printf("executing cmake integration test in %s\n", dir)

	//execute root command
	cmd := exec.Command(executable)
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	assert.NoError(t, err)

	// execute init command
	cmd = exec.Command(executable, "init")
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	assert.NoError(t, err)

	// execute create command
	outputPath := "my_test.cpp"
	cmd = exec.Command(executable,
		"create", "cpp",
		"--output", outputPath,
	)
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	assert.NoError(t, err)
}

func TestIntegration_InitCreate_DirectoryFlagAndOutFlag(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	executable, dir := prepareTestDir(t)
	defer fileutil.Cleanup(dir)
	fmt.Printf("executing cmake integration test in %s\n", dir)

	//execute root command
	cmd := exec.Command(executable, "-C", dir)
	err := cmd.Run()
	assert.NoError(t, err)

	// execute init command
	cmd = exec.Command(executable, "init", "-C", dir)
	err = cmd.Run()
	assert.NoError(t, err)

	// execute create command
	outputPath := "my_test.cpp"
	cmd = exec.Command(executable,
		"create", "cpp",
		"-C", dir,
		"--output", outputPath,
	)
	err = cmd.Run()
	assert.NoError(t, err)

	// check that the fuzz test was created in the correct directory
	require.FileExists(t, filepath.Join(dir, outputPath))
}
