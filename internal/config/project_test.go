package config

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code-intelligence.com/cifuzz/util/fileutil"
)

var baseTempDir string

func TestMain(m *testing.M) {
	var err error
	baseTempDir, err = ioutil.TempDir("", "project-config-test-")
	if err != nil {
		log.Fatalf("Failed to create temp dir for tests: %+v", err)
	}
	defer fileutil.Cleanup(baseTempDir)

	m.Run()
}

func TestCreateProjectConfig(t *testing.T) {
	projectDir, err := ioutil.TempDir(baseTempDir, "project-")
	require.NoError(t, err)

	path, err := CreateProjectConfig(projectDir)
	assert.NoError(t, err)
	expectedPath := filepath.Join(projectDir, "cifuzz.yaml")
	assert.Equal(t, expectedPath, path)

	// file created?
	exists, err := fileutil.Exists(expectedPath)
	assert.NoError(t, err)
	assert.True(t, exists)

	// check for content
	content, err := ioutil.ReadFile(expectedPath)
	assert.NoError(t, err)
	assert.NotEmpty(t, content)
	assert.Contains(t, string(content), "Configuration for")

}

// Should return error if not allowed to write to directory
func TestCreateProjectConfig_NoPerm(t *testing.T) {
	// create read only project dir
	projectDir, err := ioutil.TempDir(baseTempDir, "project-")
	require.NoError(t, err)
	err = os.Chmod(projectDir, 0555)
	require.NoError(t, err)

	path, err := CreateProjectConfig(projectDir)
	assert.Error(t, err)
	assert.ErrorIs(t, err, os.ErrPermission)
	assert.Empty(t, path)

	// file should not exists
	exists, err := fileutil.Exists("cifuzz.yaml")
	assert.NoError(t, err)
	assert.False(t, exists)
}

// Should return error if file already exists
func TestCreateProjectConfig_Exists(t *testing.T) {
	projectDir, err := ioutil.TempDir(baseTempDir, "project-")
	require.NoError(t, err)

	existingPath := filepath.Join(projectDir, "cifuzz.yaml")
	err = ioutil.WriteFile(existingPath, []byte{}, 0644)
	require.NoError(t, err)

	path, err := CreateProjectConfig(filepath.Dir(existingPath))
	assert.Error(t, err)
	// check if path of the existing config is return and the error indicates it too
	assert.ErrorIs(t, err, os.ErrExist)
	assert.Equal(t, existingPath, path)

	// file should not exists
	exists, err := fileutil.Exists(existingPath)
	assert.NoError(t, err)
	assert.True(t, exists)
}
