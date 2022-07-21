package cmdutils

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// ValidateSeedCorpusDirs checks if the seed dirs exist and can be
// accessed and ensures that the paths are absolute
func ValidateSeedCorpusDirs(seedCorpusDirs []string) ([]string, error) {
	for i, d := range seedCorpusDirs {
		_, err := os.Stat(d)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		seedCorpusDirs[i], err = filepath.Abs(d)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}
	return seedCorpusDirs, nil
}
