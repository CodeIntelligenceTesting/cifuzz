package artifact

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"

	"github.com/pkg/errors"
)

// WriteArchive writes a GZip-compressed TAR to out containing the files and empty directories given in manifest.
// The keys in manifest correspond to the path within the archive, the corresponding value is expected to be the
// absolute path of the file or directory on disk.
func WriteArchive(out io.Writer, manifest map[string]string) error {
	gw := gzip.NewWriter(out)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	for archivePath, absPath := range manifest {
		err := addToArchive(tw, archivePath, absPath)
		if err != nil {
			return err
		}
	}

	return nil
}

// addToArchive adds the file absPath to the archive under the path archivePath.
func addToArchive(tw *tar.Writer, archivePath, absPath string) error {
	fileOrDir, err := os.Open(absPath)
	if err != nil {
		return errors.WithStack(err)
	}
	defer fileOrDir.Close()
	info, err := fileOrDir.Stat()
	if err != nil {
		return errors.WithStack(err)
	}

	// Since fileOrDir.Stat() follows symlinks, info will not be of type symlink
	// at this point - no need to pass in a non-empty value for link.
	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return errors.WithStack(err)
	}
	header.Name = archivePath
	err = tw.WriteHeader(header)
	if err != nil {
		return errors.WithStack(err)
	}

	if !info.Mode().IsRegular() {
		return nil
	}
	_, err = io.Copy(tw, fileOrDir)
	if err != nil {
		return errors.Wrapf(err, "failed to compress file: %s", absPath)
	}

	return nil
}
