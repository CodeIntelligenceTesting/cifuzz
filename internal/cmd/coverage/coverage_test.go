package coverage

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"code-intelligence.com/cifuzz/pkg/cmdutils"
)

func TestCoverageCmd(t *testing.T) {
	_, err := cmdutils.ExecuteCommand(t, New(), os.Stdin)
	assert.Error(t, err)
}
