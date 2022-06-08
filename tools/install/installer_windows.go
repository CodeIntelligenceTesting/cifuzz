package install

import (
	"github.com/pkg/errors"

	"golang.org/x/sys/windows/registry"
)

func registerCMakePackage(packageDir string) error {
	// Store the path to the root directory of the CMake integration in the registry.
	// See https://cmake.org/cmake/help/latest/manual/cmake-packages.7.html#user-package-registry
	key, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Kitware\CMake\Packages\CIFuzz`, registry.ALL_ACCESS)
	if err != nil {
		return errors.WithStack(err)
	}
	defer key.Close()
	err = key.SetStringValue("CIFuzz", packageDir)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
