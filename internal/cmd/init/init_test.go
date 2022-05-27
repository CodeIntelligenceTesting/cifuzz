package init

import (
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"code-intelligence.com/cifuzz/pkg/cmdutils"
	"code-intelligence.com/cifuzz/util/fileutil"
)

var baseTempDir string

func TestMain(m *testing.M) {
	var err error
	baseTempDir, err = ioutil.TempDir("", "init-cmd-test-")
	if err != nil {
		log.Fatalf("Failed to create temp dir for tests: %+v", err)
	}
	defer fileutil.Cleanup(baseTempDir)

	err = os.Chdir(baseTempDir)
	if err != nil {
		log.Fatalf("Failed to change the working directory to %s", baseTempDir)
	}

	m.Run()
}

func TestInitCmd(t *testing.T) {
	_, err := cmdutils.ExecuteCommand(t, New(), os.Stdin)
	assert.NoError(t, err)

	// second execution should return a ErrSilent as the config file should aready exists
	_, err = cmdutils.ExecuteCommand(t, New(), os.Stdin)
	assert.Error(t, err)
	assert.ErrorIs(t, err, cmdutils.ErrSilent)
}
