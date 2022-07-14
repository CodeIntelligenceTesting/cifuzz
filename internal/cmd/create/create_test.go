package create

import (
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"code-intelligence.com/cifuzz/internal/config"
	"code-intelligence.com/cifuzz/pkg/cmdutils"
	"code-intelligence.com/cifuzz/util/fileutil"
)

var baseTempDir string

func TestMain(m *testing.M) {
	var err error
	baseTempDir, err = os.MkdirTemp("", "create-cmd-test-")
	if err != nil {
		log.Fatalf("Failed to create temp dir for tests: %+v", err)
	}
	defer fileutil.Cleanup(baseTempDir)
	m.Run()
}

func TestCreateCmd(t *testing.T) {
	args := []string{
		"cpp",
		"--output",
		filepath.Join(baseTempDir, "fuzz-test.cpp"),
	}
	_, err := cmdutils.ExecuteCommand(t, New(config.NewConfig()), os.Stdin, args...)
	assert.NoError(t, err)
}

func TestCreateCmd_InvalidType(t *testing.T) {
	args := []string{
		"foo",
	}
	_, err := cmdutils.ExecuteCommand(t, New(config.NewConfig()), os.Stdin, args...)
	assert.Error(t, err)
}

func TestCreateCmd_OutDir(t *testing.T) {
	t.Skip()
}
