package install

import (
	"github.com/pkg/errors"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"code-intelligence.com/cifuzz/pkg/log"
)

func registerCMakePackage(packageDir string) error {
	// Store the path to the root directory of the CMake integration in the registry.
	// See https://cmake.org/cmake/help/latest/manual/cmake-packages.7.html#user-package-registry
	key := registry.CURRENT_USER
	isAdmin, err := currentUserIsAdmin()
	if err != nil {
		// Best effort given that Go has no direct support for detecting Windows administrator status.
		// Log the error and continue with the installation for the current user only.
		log.Debug(err)
	}
	if err == nil && isAdmin {
		key = registry.LOCAL_MACHINE
	}
	key, _, err = registry.CreateKey(key, `Software\Kitware\CMake\Packages\cifuzz`, registry.ALL_ACCESS)
	if err != nil {
		return errors.WithStack(err)
	}
	defer key.Close()
	err = key.SetStringValue("cifuzz", packageDir)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// currentUserIsAdmin returns true if the current user is acting as an administrator, e.g. because the current command
// prompt has been launched via "Run as administrator".
func currentUserIsAdmin() (bool, error) {
	// Based on:
	// https://github.com/golang/go/issues/28804#issuecomment-505326268
	var sid *windows.SID

	// See https://docs.microsoft.com/en-us/windows/desktop/api/securitybaseapi/nf-securitybaseapi-checktokenmembership
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid)
	if err != nil {
		return false, errors.WithStack(err)
	}
	defer windows.FreeSid(sid)

	// https://github.com/golang/go/issues/28804#issuecomment-438838144
	isAdmin, err := windows.Token(0).IsMember(sid)
	if err != nil {
		return false, errors.WithStack(err)
	}
	return isAdmin, nil
}
