package root

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code-intelligence.com/cifuzz/pkg/cmdutils"
	"code-intelligence.com/cifuzz/pkg/storage"
)

func TestRootCmd(t *testing.T) {
	fs := storage.NewMemFileSystem()
	_, err := cmdutils.ExecuteCommand(t, New(fs), os.Stdin)
	assert.NoError(t, err)
}

func TestChangingToNonExistingDirectory(t *testing.T) {
	// afero doesn't support Chdir and Getwd, so we have to use the
	// OS filesystem here instead of the in-memory one.
	fs := storage.WrapFileSystem()
	testDir, err := fs.TempDir("", "test-")
	require.NoError(t, err)
	err = os.Chdir(testDir)
	require.NoError(t, err)
	defer os.RemoveAll(testDir)

	origWorkDir, err := os.Getwd()
	require.NoError(t, err)

	args := []string{
		"-C", "foo",
		// The PersistentPreRunE function in which we change the
		// directory is only executed if a subcommand is specified,
		// else only the usage message is printed, so we specify a
		// subcommand.
		"init",
	}
	_, err = cmdutils.ExecuteCommand(t, New(fs), os.Stdin, args...)
	require.Error(t, err)

	// Check that the working directory did not change
	workDir, err := os.Getwd()
	require.NoError(t, err)
	require.Equal(t, origWorkDir, workDir)
}

func TestChangingToExistingDirectory(t *testing.T) {
	// afero doesn't support Chdir and Getwd, so we have to use the
	// OS filesystem here instead of the in-memory one.
	fs := storage.WrapFileSystem()
	testDir, err := fs.TempDir("", "test-")
	require.NoError(t, err)
	err = os.Chdir(testDir)
	require.NoError(t, err)
	defer os.RemoveAll(testDir)

	origWorkDir, err := os.Getwd()
	require.NoError(t, err)

	err = fs.Mkdir("foo", 0700)
	require.NoError(t, err)

	args := []string{
		"-C", "./foo",
		// The PersistentPreRunE function in which we change the
		// directory is only executed if a subcommand is specified,
		// else only the usage message is printed, so we specify a
		// subcommand.
		"init",
	}
	_, err = cmdutils.ExecuteCommand(t, New(fs), os.Stdin, args...)
	require.NoError(t, err)

	// Check that the working directory actually changed
	workDir, err := os.Getwd()
	require.NoError(t, err)
	require.Equal(t, filepath.Join(origWorkDir, "foo"), workDir)
}
