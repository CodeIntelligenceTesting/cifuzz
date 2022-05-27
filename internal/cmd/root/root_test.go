package root

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code-intelligence.com/cifuzz/pkg/cmdutils"
	"code-intelligence.com/cifuzz/util/fileutil"
)

func TestRootCmd(t *testing.T) {
	_, err := cmdutils.ExecuteCommand(t, New(), os.Stdin)
	assert.NoError(t, err)
}

func TestChangingToNonExistingDirectory(t *testing.T) {
	testDir, err := ioutil.TempDir("", "test-")
	require.NoError(t, err)
	err = os.Chdir(testDir)
	require.NoError(t, err)
	defer fileutil.Cleanup(testDir)

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
	_, err = cmdutils.ExecuteCommand(t, New(), os.Stdin, args...)
	require.Error(t, err)

	// Check that the working directory did not change
	workDir, err := os.Getwd()
	require.NoError(t, err)
	require.Equal(t, origWorkDir, workDir)
}

func TestChangingToExistingDirectory(t *testing.T) {
	testDir, err := ioutil.TempDir("", "test-")
	require.NoError(t, err)
	err = os.Chdir(testDir)
	require.NoError(t, err)
	defer fileutil.Cleanup(testDir)

	origWorkDir, err := os.Getwd()
	require.NoError(t, err)

	err = os.Mkdir("foo", 0700)
	require.NoError(t, err)

	args := []string{
		"-C", "./foo",
		// The PersistentPreRunE function in which we change the
		// directory is only executed if a subcommand is specified,
		// else only the usage message is printed, so we specify a
		// subcommand.
		"init",
	}
	_, err = cmdutils.ExecuteCommand(t, New(), os.Stdin, args...)
	require.NoError(t, err)

	// Check that the working directory actually changed
	workDir, err := os.Getwd()
	require.NoError(t, err)
	require.Equal(t, filepath.Join(origWorkDir, "foo"), workDir)
}
