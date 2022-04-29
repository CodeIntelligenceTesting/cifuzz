package stubs

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"code-intelligence.com/cifuzz/internal/config"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

//go:embed fuzz-test.cpp.tmpl
var cppStub []byte

// Create creates a stub based for the given targetType
func Create(path string, targetType config.TargetType, fs *afero.Afero) error {

	if _, err := fs.Stat(path); err == nil {
		return errors.WithStack(os.ErrExist)
	}

	// read matching template
	var content []byte
	switch targetType {
	case config.CPP:
		content = cppStub
	}

	// write stub
	if content != nil && path != "" {
		if err := fs.WriteFile(path, content, 0644); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

// SuggestFilename returns a proposal for a filename,
// depending on the test type and given directory
func SuggestFilename(dir string, testType config.FuzzTestType, fs *afero.Afero) string {
	var basename, ext, filename string

	switch targetType {
	case config.CPP:
		ext = "cpp"
		basename = "my_fuzz_test"
	}

	counter := 1
	for {
		filename = fmt.Sprintf("%s_%d.%s", basename, counter, ext)
		exists, _ := fs.Exists(filepath.Join(dir, filename))
		if exists {
			counter += 1
		} else {
			break
		}
	}

	return filename
}
