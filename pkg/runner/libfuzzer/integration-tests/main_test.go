package integration_tests

import (
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"

	"code-intelligence.com/cifuzz/pkg/install"
	"code-intelligence.com/cifuzz/util/fileutil"
)

func TestMain(m *testing.M) {
	var err error

	// Print debug output for easier debugging
	viper.Set("verbose", true)

	baseTempDir, err = os.MkdirTemp("", "libfuzzer-runner-integration-test-")
	if err != nil {
		log.Fatalf("Failed to create temp dir for tests: %+v", err)
	}

	opts := install.Options{
		Version:   "dev",
		TargetDir: filepath.Join(baseTempDir, "install-dir"),
	}
	bundler, err = install.NewInstallationBundler(opts)
	if err != nil {
		fileutil.Cleanup(baseTempDir)
		log.Fatalf("Failed to create install dir for tests: %+v", err)
	}

	defer fileutil.Cleanup(baseTempDir)
	m.Run()
}
