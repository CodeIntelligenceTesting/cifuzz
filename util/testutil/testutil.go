package testutil

import (
	"io/fs"
	"os"
	"path/filepath"
)

// RegisterTestDeps ensures that the test calling this function is rerun (despite caching) if any of the files and
// directories (and their recursive contents) under the provided paths change.
func RegisterTestDeps(path ...string) {
	// Workaround for https://github.com/golang/go/issues/53053
	// Explicitly stat all data dirs and files so that the Go test runner picks up the data dependency and knows how to
	// rerun the test if the data dir contents change. Without this explicit recursive walk, changes to files in
	// subdirectories aren't picked up automatically.
	for _, p := range path {
		err := filepath.Walk(p, func(path string, info fs.FileInfo, _ error) error {
			_, err := os.Stat(path)
			if err != nil {
				// Fail hard if the declared test dep does not exist.
				panic(err)
			}
			return nil
		})
		if err != nil {
			panic(err)
		}
	}
}
