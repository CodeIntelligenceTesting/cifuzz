package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateCmd(t *testing.T) {
	args := []string{
		"create",
		"cpp",
		"--name",
		"fuzz-test.cpp",
	}
	_, err := ExecuteCommand(t, os.Stdin, args...)
	assert.NoError(t, err)
}

func TestCreateCmd_InvalidType(t *testing.T) {
	args := []string{
		"create",
		"foo",
	}
	_, err := ExecuteCommand(t, os.Stdin, args...)
	assert.Error(t, err)
}

func TestCreateCmd_InputFilename(t *testing.T) {
	input := []byte("my_test_file.cpp")
	r, w, err := os.Pipe()
	assert.NoError(t, err)

	_, err = w.Write(input)
	assert.NoError(t, err)
	w.Close()

	args := []string{
		"create", "cpp",
		"--out", "/test",
	}
	_, err = ExecuteCommand(t, r, args...)
	assert.NoError(t, err)
}

func TestCreateCmd_OutDir(t *testing.T) {
	t.Skip()
}
