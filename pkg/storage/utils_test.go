package storage

import (
	"os"
	"testing"

	"code-intelligence.com/cifuzz/pkg/workarounds"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestGetOutDir(t *testing.T) {
	fs := NewMemFileSystem()
	outDir, err := GetOutDir("/fuzz-tests", fs)
	assert.NoError(t, err)

	exists, err := fs.Exists(outDir)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestGetOutDir_Default(t *testing.T) {
	fs := NewMemFileSystem()
	outDir, err := GetOutDir("", fs)
	assert.NoError(t, err)

	cwd, err := os.Getwd()
	assert.NoError(t, err)
	assert.Equal(t, cwd, outDir)
}

func TestGetOutDir_NoPerm(t *testing.T) {
	fs := NewReadOnlyFileSystem()

	outDir, err := GetOutDir("/fuzz-tests", fs)
	assert.Error(t, err)
	assert.True(t, workarounds.IsPermission(errors.Cause(err)))
	assert.Equal(t, "/fuzz-tests", outDir)

	// directory should not exists
	exists, err := fs.Exists("/fuzz-tests")
	assert.NoError(t, err)
	assert.False(t, exists)
}
