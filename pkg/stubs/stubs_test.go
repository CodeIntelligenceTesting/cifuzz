package stubs

import (
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/hectane/go-acl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code-intelligence.com/cifuzz/internal/config"
	"code-intelligence.com/cifuzz/util/fileutil"
)

var baseTempDir string

func TestMain(m *testing.M) {
	var err error
	baseTempDir, err = os.MkdirTemp("", "stubs-test-")
	if err != nil {
		log.Fatalf("Failed to create temp dir for tests: %+v", err)
	}
	defer fileutil.Cleanup(baseTempDir)
	m.Run()
}

func TestCreate(t *testing.T) {
	projectDir, err := os.MkdirTemp(baseTempDir, "project-")
	require.NoError(t, err)

	stubFile := filepath.Join(projectDir, "fuzz_test.cpp")
	err = Create(stubFile, config.CPP)
	assert.NoError(t, err)

	exists, err := fileutil.Exists(stubFile)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestCreate_Exists(t *testing.T) {
	projectDir, err := os.MkdirTemp(baseTempDir, "project-")
	require.NoError(t, err)

	stubFile := filepath.Join(projectDir, "fuzz_test.cpp")
	err = os.WriteFile(stubFile, []byte("TEST"), 0644)
	assert.NoError(t, err)

	err = Create(stubFile, config.CPP)
	assert.Error(t, err)
	assert.ErrorIs(t, err, os.ErrExist)
}

func TestCreate_NoPerm(t *testing.T) {
	// create read only project dir
	projectDir, err := os.MkdirTemp(baseTempDir, "project-")
	require.NoError(t, err)
	err = acl.Chmod(projectDir, 0555)
	require.NoError(t, err)

	stubFile := filepath.Join(projectDir, "fuzz_test.cpp")
	err = Create(stubFile, config.CPP)
	assert.Error(t, err)
	assert.ErrorIs(t, err, os.ErrPermission)
}

func TestSuggestFilename(t *testing.T) {
	projectDir, err := os.MkdirTemp(baseTempDir, "project-")
	require.NoError(t, err)
	err = os.Chdir(projectDir)
	require.NoError(t, err)

	filename1, err := FuzzTestFilename(config.CPP)
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(".", "my_fuzz_test_1.cpp"), filename1)

	err = os.WriteFile(filename1, []byte("TEST"), 0644)
	require.NoError(t, err)

	filename2, err := FuzzTestFilename(config.CPP)
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(".", "my_fuzz_test_2.cpp"), filename2)
}
