package create

import (
	"os"
	"testing"

	"code-intelligence.com/cifuzz/pkg/cmdutils"
	"code-intelligence.com/cifuzz/pkg/storage"
	"github.com/stretchr/testify/assert"
)

func TestCreateCmd(t *testing.T) {
	fs := storage.NewMemFileSystem()

	args := []string{
		"cpp",
		"--out",
		"/tests/fuzz",
		"--name",
		"fuzz-test.cpp",
	}
	_, err := cmdutils.ExecuteCommand(t, NewCmdCreate(fs), os.Stdin, args...)
	assert.NoError(t, err)
}

func TestCreateCmd_InvalidType(t *testing.T) {
	fs := storage.NewMemFileSystem()

	args := []string{
		"foo",
	}
	_, err := cmdutils.ExecuteCommand(t, NewCmdCreate(fs), os.Stdin, args...)
	assert.Error(t, err)
}

func TestCreateCmd_InputFilename(t *testing.T) {
	fs := storage.NewMemFileSystem()

	input := []byte("my_test_file.cpp\n")
	r, w, err := os.Pipe()
	assert.NoError(t, err)

	_, err = w.Write(input)
	assert.NoError(t, err)
	w.Close()

	args := []string{
		"cpp",
		"--out", "/test/",
	}

	_, err = cmdutils.ExecuteCommand(t, NewCmdCreate(fs), r, args...)
	assert.NoError(t, err)

	exists, err := fs.Exists("/test/my_test_file.cpp")
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestCreateCmd_OutDir(t *testing.T) {
	t.Skip()
}
