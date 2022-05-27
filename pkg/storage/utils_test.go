package storage

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/hectane/go-acl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code-intelligence.com/cifuzz/util/fileutil"
)

var baseTempDir string

func TestMain(m *testing.M) {
	var err error
	baseTempDir, err = ioutil.TempDir("", "storage-test-")
	if err != nil {
		log.Fatalf("Failed to create temp dir for tests: %+v", err)
	}
	defer fileutil.Cleanup(baseTempDir)

	m.Run()
}

func TestGetOutDir(t *testing.T) {
	projectDir, err := ioutil.TempDir(baseTempDir, "project-")
	require.NoError(t, err)

	outDir, err := GetOutDir(filepath.Join(projectDir, "fuzz-tests"))
	assert.NoError(t, err)

	exists, err := fileutil.Exists(outDir)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestGetOutDir_Default(t *testing.T) {
	outDir, err := GetOutDir("")
	assert.NoError(t, err)

	cwd, err := os.Getwd()
	assert.NoError(t, err)
	assert.Equal(t, cwd, outDir)
}

func TestGetOutDir_NoPerm(t *testing.T) {
	// create read only project dir
	projectDir, err := ioutil.TempDir(baseTempDir, "project-")
	require.NoError(t, err)
	err = acl.Chmod(projectDir, 0555)
	require.NoError(t, err)

	outDir, err := GetOutDir(filepath.Join(projectDir, "fuzz-tests"))
	assert.Error(t, err)
	assert.ErrorIs(t, err, os.ErrPermission)
	assert.Equal(t, filepath.Join(projectDir, "fuzz-tests"), outDir)

	// directory should not exists
	exists, err := fileutil.Exists(filepath.Join(projectDir, "fuzz-tests"))
	assert.NoError(t, err)
	assert.False(t, exists)
}
