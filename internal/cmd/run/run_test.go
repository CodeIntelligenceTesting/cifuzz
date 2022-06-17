package run

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"code-intelligence.com/cifuzz/internal/config"
	"code-intelligence.com/cifuzz/pkg/cmdutils"
)

func TestRunCmd(t *testing.T) {
	_, err := cmdutils.ExecuteCommand(t, New(config.NewConfig()), os.Stdin)
	assert.Error(t, err)
}
