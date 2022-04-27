package config

import (
	"os"
	"path/filepath"
	"testing"

	"code-intelligence.com/cifuzz/pkg/storage"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestCreateProjectConfig(t *testing.T) {
	fs := storage.NewMemFileSystem()

	path, err := CreateProjectConfig("foo", fs)
	assert.NoError(t, err)
	expectedPath := filepath.Join("foo", "cifuzz.yaml")
	assert.Equal(t, expectedPath, path)

	// file created?
	exists, err := fs.Exists(expectedPath)
	assert.NoError(t, err)
	assert.True(t, exists)

	// check for content
	content, err := fs.ReadFile(expectedPath)
	assert.NoError(t, err)
	assert.NotEmpty(t, content)
	assert.Contains(t, string(content), "Configuration for")

}

// Should return error if not allowed to write to directory
func TestCreateProjectConfig_NoPerm(t *testing.T) {
	// create read only filesystem
	fs := &afero.Afero{Fs: afero.NewReadOnlyFs(afero.NewOsFs())}

	path, err := CreateProjectConfig(".", fs)
	assert.Error(t, err)
	assert.True(t, os.IsPermission(errors.Cause(err)))
	assert.Empty(t, path)

	// file should not exists
	exists, err := fs.Exists("cifuzz.yaml")
	assert.NoError(t, err)
	assert.False(t, exists)
}

// Should return error if file already exists
func TestCreateProjectConfig_Exists(t *testing.T) {
	fs := storage.NewMemFileSystem()
	existingPath := filepath.Join("foo", "cifuzz.yaml")
	fs.WriteFile(existingPath, []byte{}, 0644)

	path, err := CreateProjectConfig(filepath.Dir(existingPath), fs)
	assert.Error(t, err)
	// check if path of the existing config is return and the error indicates it too
	assert.True(t, os.IsExist(errors.Cause(err)))
	assert.Equal(t, existingPath, path)

	// file should not exists
	exists, err := fs.Exists(existingPath)
	assert.NoError(t, err)
	assert.True(t, exists)
}
