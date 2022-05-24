package testutil

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"code-intelligence.com/cifuzz/util/glogutil"
)

// SetupGlog sets up glog flags from env vars and enables logging to
// stderr.
// You still have to call flag.Parse() after calling SetupGlog().
func SetupGlog() {
	// Also log to stderr
	err := os.Setenv("GLOG_alsologtostderr", "1")
	if err != nil {
		log.Fatalf("%+v", err)
	}

	// If not set already, set the GLOG_v env var to 2, to enable
	// moderately verbose logging in tests
	if os.Getenv("GLOG_v") == "" && os.Getenv("GLOG_V") == "" {
		err = os.Setenv("GLOG_v", "2")
		if err != nil {
			log.Fatalf("%+v", err)
		}
	}

	// Set up glog
	err = glogutil.SetupGlog()
	if err != nil {
		log.Fatalf("%+v", err)
	}
}

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
