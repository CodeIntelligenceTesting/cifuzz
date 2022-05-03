package storage

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

// GetOutDir returns the output directory (requestedDir param or cwd)
// and ensure it exists
func GetOutDir(requestedDir string, fs *afero.Afero) (string, error) {

	// default case: return the current working directory
	if requestedDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", errors.WithStack(err)
		}
		return cwd, nil
	}

	if _, err := fs.Stat(requestedDir); err != nil && !os.IsNotExist(err) {
		return requestedDir, errors.WithStack(err)
	}
	if err := fs.MkdirAll(requestedDir, 0744); err != nil {
		return requestedDir, errors.WithStack(err)
	}
	return requestedDir, nil
}
