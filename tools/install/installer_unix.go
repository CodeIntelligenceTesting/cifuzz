//go:build !windows

package install

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

func registerCMakePackage(packageDir string) error {
	// Install the CMake package for the current user only by registering it
	// with the user package registry. This requires creating a file
	// ~/.cmake/packages/CIFuzz/CIFuzz containing the path to the root directory
	// of the CMake integration.
	// See https://cmake.org/cmake/help/latest/manual/cmake-packages.7.html#user-package-registry
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return errors.WithStack(err)
	}
	cmakePackagesDir := filepath.Join(homeDir, ".cmake", "packages", "CIFuzz")
	err = os.MkdirAll(cmakePackagesDir, 0755)
	if err != nil {
		return errors.WithStack(err)
	}
	err = ioutil.WriteFile(filepath.Join(cmakePackagesDir, "CIFuzz"), []byte(packageDir), 0644)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
