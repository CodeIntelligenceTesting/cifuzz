package cmd

import (
	"os"
	"testing"

	"code-intelligence.com/cifuzz/pkg/storage"
	"github.com/stretchr/testify/assert"
)

func TestCreateCmd(t *testing.T) {
	fs = storage.NewMemFileSystem()

	args := []string{
		"create",
		"cpp",
		"--out",
		"/tests/fuzz",
		"--name",
		"fuzz-test.cpp",
	}
	_, err := ExecuteCommand(t, os.Stdin, args...)
	assert.NoError(t, err)
}

func TestCreateCmd_InvalidType(t *testing.T) {
	fs = storage.NewMemFileSystem()

	args := []string{
		"create",
		"foo",
	}
	_, err := ExecuteCommand(t, os.Stdin, args...)
	assert.Error(t, err)
}

func TestCreateCmd_InputFilename(t *testing.T) {
	fs = storage.NewMemFileSystem()

	input := []byte("my_test_file.cpp\n")
	r, w, err := os.Pipe()
	assert.NoError(t, err)

	_, err = w.Write(input)
	assert.NoError(t, err)
	w.Close()

	args := []string{
		"create", "cpp",
		"--out", "/test/",
	}

	_, err = ExecuteCommand(t, r, args...)
	assert.NoError(t, err)

	exists, err := fs.Exists("/test/my_test_file.cpp")
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestCreateCmd_OutDir(t *testing.T) {
	t.Skip()
}
