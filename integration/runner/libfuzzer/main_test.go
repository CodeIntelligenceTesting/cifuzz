package libfuzzer

import (
	"io/ioutil"
	"log"
	"testing"

	"github.com/spf13/viper"

	"code-intelligence.com/cifuzz/util/fileutil"
)

var baseTempDir string

func TestMain(m *testing.M) {
	var err error

	// Print debug output for easier debugging
	viper.Set("verbose", true)

	baseTempDir, err = ioutil.TempDir("", "libfuzzer-runner-integration-test-")
	if err != nil {
		log.Fatalf("Failed to create temp dir for tests: %+v", err)
	}
	defer fileutil.Cleanup(baseTempDir)

	m.Run()
}
