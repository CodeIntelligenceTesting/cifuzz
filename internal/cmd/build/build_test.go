package build

import (
	"os"
	"testing"

	"code-intelligence.com/cifuzz/pkg/cmdutils"
	"github.com/stretchr/testify/assert"
)

func TestBuildCmd(t *testing.T) {
	_, err := cmdutils.ExecuteCommand(t, NewCmdBuild(), os.Stdin)
	assert.Error(t, err)
}
