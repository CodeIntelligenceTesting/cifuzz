package storage

import (
	"code-intelligence.com/cifuzz/pkg/log"
	"github.com/spf13/afero"
)

// InitFileSystem returns a wrapper for the os/host file system
func WrapFileSystem() *afero.Afero {
	log.Debug("Using os filesystem wrapper")
	return &afero.Afero{Fs: afero.NewOsFs()}
}

// InitMemFileSystem gives access to a memory based file system for using in tests
func NewMemFileSystem() *afero.Afero {
	log.Warn("Using in-memory filesystem wrapper")
	return &afero.Afero{Fs: afero.NewMemMapFs()}
}

func NewReadOnlyFileSystem() *afero.Afero {
	log.Warn("Using read only filesystem wrapper")
	return &afero.Afero{Fs: afero.NewReadOnlyFs(afero.NewMemMapFs())}
}
