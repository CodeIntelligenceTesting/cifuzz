package storage

import (
	"github.com/spf13/afero"
)

// FileSystem is just embedding the afero.Afero struct to encapsulate it
type FileSystem struct {
	afero.Afero
}

// InitFileSystem returns a wrapper for the os/host file system
func WrapFileSystem() *FileSystem {
	return &FileSystem{Afero: afero.Afero{Fs: afero.NewOsFs()}}
}

// InitMemFileSystem gives access to a memory based file system for using in tests
func NewMemFileSystem() *FileSystem {
	return &FileSystem{Afero: afero.Afero{Fs: afero.NewMemMapFs()}}
}
