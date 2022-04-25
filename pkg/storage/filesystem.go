package storage

import (
	"code-intelligence.com/cifuzz/pkg/out"
	"github.com/spf13/afero"
)

// InitFileSystem returns a wrapper for the os/host file system
func WrapFileSystem() *afero.Afero {
	out.Debug("using os filesystem wrapper")
	return &afero.Afero{Fs: afero.NewOsFs()}
}

// InitMemFileSystem gives access to a memory based file system for using in tests
func NewMemFileSystem() *afero.Afero {
	out.Warn("using in-memory filesystem wrapper")
	return &afero.Afero{Fs: afero.NewMemMapFs()}
}
