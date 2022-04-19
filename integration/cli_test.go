package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration(t *testing.T) {
	// make sure executable exists
	testExecutable := fmt.Sprintf("./cifuzz_%s", runtime.GOOS)
	if runtime.GOOS == "windows" {
		testExecutable = testExecutable + ".exe"
	}

	require.FileExistsf(t, testExecutable, "make sure an executable of ci fuzz is present under %s", testExecutable)

	var out bytes.Buffer
	cmd := exec.Command(testExecutable)
	cmd.Stdout = &out

	err := cmd.Run()
	assert.NoError(t, err)
}
