package fileutil

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"code-intelligence.com/cifuzz/pkg/log"
)

func IsSymlink(path string) bool {
	f, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return f.Mode()&os.ModeSymlink != 0
}

// IsDir returns whether this path is a directory. Tries to behave the
// same as Python's pathlib.Path.is_dir()
func IsDir(path string) bool {
	f, err := os.Stat(path)
	if err != nil {
		return false
	}
	return f.Mode()&os.ModeDir != 0
}

// Touch creates a file at the given path
func Touch(path string) error {
	file, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil && !errors.Is(err, os.ErrExist) {
		return errors.WithStack(err)
	}
	err = file.Close()
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, errors.WithStack(err)
	}
	return !errors.Is(err, os.ErrNotExist), nil
}

// Cleanup removes the specified file or directory and prints any errors
// to stderr. It's supposed to be used in defer statements to clean up
// temporary directories.
func Cleanup(path string) {
	if os.Getenv("SKIP_CLEANUP") != "" {
		return
	}

	err := os.RemoveAll(path)
	if err != nil {
		log.Warnf("%+v", errors.WithStack(err))
	}
}

// PrettifyPath prints a possibly shortened path for display purposes.
// If path is located under the current working directory, the relative path to
// it is returned, otherwise or in case of an error the path is returned
// unchanged.
func PrettifyPath(path string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return path
	}
	rel, err := filepath.Rel(cwd, path)
	if err != nil {
		return path
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, filepath.FromSlash("../")) {
		return path
	}
	return rel
}
