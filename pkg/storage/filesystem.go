package storage

import (
	"github.com/spf13/afero"
)

// InitFileSystem returns a wrapper for the os/host file system
func WrapFileSystem() *afero.Afero {
	return &afero.Afero{Fs: afero.NewOsFs()}
}

// InitMemFileSystem gives access to a memory based file system for using in tests
func NewMemFileSystem() *afero.Afero {
	return &afero.Afero{Fs: afero.NewMemMapFs()}
}
