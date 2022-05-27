package storage

import (
	"os"

	"github.com/pkg/errors"
)

// GetOutDir returns the output directory (requestedDir param or cwd)
// and ensure it exists
func GetOutDir(requestedDir string) (string, error) {

	// default case: return the current working directory
	if requestedDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", errors.WithStack(err)
		}
		return cwd, nil
	}

	if err := os.MkdirAll(requestedDir, 0755); err != nil {
		return requestedDir, errors.WithStack(err)
	}
	return requestedDir, nil
}
