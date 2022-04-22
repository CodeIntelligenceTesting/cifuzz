package config

import (
	"testing"

	"code-intelligence.com/cifuzz/pkg/storage"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestCreateProjectConfig(t *testing.T) {
	fs := storage.NewMemFileSystem()

	path, err := CreateProjectConfig("/", fs)
	assert.NoError(t, err)
	assert.Equal(t, "/cifuzz.yaml", path)

	// file created?
	exists, err := fs.Exists("/cifuzz.yaml")
	assert.NoError(t, err)
	assert.True(t, exists)

	// check for content
	content, err := fs.ReadFile("/cifuzz.yaml")
	assert.NoError(t, err)
	assert.NotEmpty(t, content)
	assert.Contains(t, string(content), "Configuration for")

}

// Should return error if not allowed to write to directory
func TestCreateProjectConfig_NoPerm(t *testing.T) {
	// create read only filesystem
	fs := &storage.FileSystem{Afero: afero.Afero{Fs: afero.NewReadOnlyFs(afero.NewOsFs())}}

	path, err := CreateProjectConfig("/", fs)
	assert.Error(t, err)
	assert.Empty(t, path)

	// file should not exists
	exists, err := fs.Exists("/cifuzz.yaml")
	assert.NoError(t, err)
	assert.False(t, exists)
}

// Should return error if file already exists
func TestCreateProjectConfig_Exists(t *testing.T) {
	fs := storage.NewMemFileSystem()
	fs.WriteFile("/cifuzz.yaml", []byte{}, 0644)

	path, err := CreateProjectConfig("/", fs)
	assert.Error(t, err)
	assert.Empty(t, path)

	// file should not exists
	exists, err := fs.Exists("/cifuzz.yaml")
	assert.NoError(t, err)
	assert.True(t, exists)
}
