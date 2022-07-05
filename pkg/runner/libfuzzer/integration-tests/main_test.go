package integration_tests

import (
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"

	"code-intelligence.com/cifuzz/tools/install"
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

	installer, err = install.NewInstaller(&install.Options{InstallDir: filepath.Join(baseTempDir, "install-dir")})
	if err != nil {
		fileutil.Cleanup(baseTempDir)
		log.Fatalf("Failed to create install dir for tests: %+v", err)
	}

	defer fileutil.Cleanup(baseTempDir)
	m.Run()
}
