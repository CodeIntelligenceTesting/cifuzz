package stubs

import (
	"errors"
	"os"
	"testing"

	"code-intelligence.com/cifuzz/internal/config"
	"code-intelligence.com/cifuzz/pkg/storage"
	"code-intelligence.com/cifuzz/pkg/workarounds"
	"github.com/stretchr/testify/assert"
)

func TestCreate(t *testing.T) {
	stubFile := "tests/fuzz_test.cpp"
	fs := storage.NewMemFileSystem()

	err := Create(stubFile, config.CPP, fs)
	assert.NoError(t, err)

	exists, err := fs.Exists(stubFile)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestCreate_Exists(t *testing.T) {
	stubFile := "tests/fuzz_test.cpp"
	fs := storage.NewMemFileSystem()

	err := fs.WriteFile(stubFile, []byte("TEST"), 0644)
	assert.NoError(t, err)

	err = Create(stubFile, config.CPP, fs)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, os.ErrExist))
}

func TestCreate_NoPerm(t *testing.T) {
	stubFile := "tests/fuzz_test.cpp"
	fs := storage.NewReadOnlyFileSystem()

	err := Create(stubFile, config.CPP, fs)
	assert.Error(t, err)
	assert.True(t, workarounds.IsPermission(err))
}

func TestSuggestFilenam(t *testing.T) {
	fs := storage.NewMemFileSystem()

	filename1, err := SuggestFilename("tests", config.CPP, fs)
	assert.NoError(t, err)
	assert.Equal(t, "my_fuzz_test_1.cpp", filename1)

	fs.WriteFile("tests/my_fuzz_test_1.cpp", []byte("TEST"), 0644)

	filename2, err := SuggestFilename("tests", config.CPP, fs)
	assert.NoError(t, err)
	assert.Equal(t, "my_fuzz_test_2.cpp", filename2)
}
