package run

import (
	"os"
	"testing"

	"code-intelligence.com/cifuzz/pkg/cmdutils"
	"github.com/stretchr/testify/assert"
)

func TestRunCmd(t *testing.T) {
	_, err := cmdutils.ExecuteCommand(t, NewCmdRun(), os.Stdin)
	assert.Error(t, err)
}
