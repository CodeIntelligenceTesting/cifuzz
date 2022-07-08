package fileutil_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code-intelligence.com/cifuzz/util/fileutil"
)

func TestPrettifyPath(t *testing.T) {
	var filesystemRoot string
	if runtime.GOOS == "windows" {
		filesystemRoot = "C:\\"
	} else {
		filesystemRoot = "/"
	}
	cwd, err := os.Getwd()
	require.NoError(t, err)

	assert.Equal(t, filesystemRoot+filepath.Join("not", "cwd"), fileutil.PrettifyPath(filesystemRoot+filepath.Join("not", "cwd")))
	assert.Equal(t, filepath.Join("some", "dir"), fileutil.PrettifyPath(filepath.Join(cwd, "some", "dir")))
	assert.Equal(t, cwd, fileutil.PrettifyPath(cwd))
	assert.Equal(t, filepath.Dir(cwd), fileutil.PrettifyPath(filepath.Dir(cwd)))
	assert.Equal(t, filepath.Join("..some", "dir"), fileutil.PrettifyPath(filepath.Join(cwd, "..some", "dir")))
}

func TestIsUnder(t *testing.T) {
	isUnder, err := fileutil.IsUnder(filepath.Join("dir1", "dir2", "file"), filepath.Join("dir1", "dir2"))
	assert.NoError(t, err)
	assert.True(t, isUnder)

	isUnder, err = fileutil.IsUnder(filepath.Join("dir1", "dir2"), filepath.Join("dir1", "dir2"))
	assert.NoError(t, err)
	assert.True(t, isUnder)

	isUnder, err = fileutil.IsUnder("dir1", filepath.Join("dir1", "dir2"))
	assert.NoError(t, err)
	assert.False(t, isUnder)

	isUnder, err = fileutil.IsUnder(".", filepath.Join("dir1", "dir2"))
	assert.NoError(t, err)
	assert.False(t, isUnder)
}
