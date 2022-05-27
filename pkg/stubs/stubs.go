package stubs

import (
	_ "embed"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"code-intelligence.com/cifuzz/internal/config"
	"code-intelligence.com/cifuzz/util/fileutil"
)

//go:embed fuzz-test.cpp.tmpl
var cppStub []byte

// Create creates a stub based for the given test type
func Create(path string, testType config.FuzzTestType) error {
	exists, err := fileutil.Exists(path)
	if err != nil {
		return err
	}
	if exists {
		return errors.WithStack(os.ErrExist)
	}

	// read matching template
	var content []byte
	switch testType {
	case config.CPP:
		content = cppStub
	}

	// write stub
	if content != nil && path != "" {
		if err := ioutil.WriteFile(path, content, 0644); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

// SuggestFilename returns a proposal for a filename,
// depending on the test type and given directory
func SuggestFilename(dir string, testType config.FuzzTestType) (string, error) {
	var basename, ext, filename string

	switch testType {
	case config.CPP:
		ext = "cpp"
		basename = "my_fuzz_test"
	default:
		return "", errors.New("unable to suggest filename: unknown test type")
	}

	for counter := 1; ; counter++ {
		filename = fmt.Sprintf("%s_%d.%s", basename, counter, ext)
		exists, err := fileutil.Exists(filepath.Join(dir, filename))
		if err != nil {
			return "", err
		}
		if !exists {
			return filename, nil
		}
	}
}
