package root

import (
	"os"
	"testing"

	"code-intelligence.com/cifuzz/pkg/cmdutils"
	"code-intelligence.com/cifuzz/pkg/storage"
	"github.com/stretchr/testify/assert"
)

func TestRootCmd(t *testing.T) {
	fs := storage.NewMemFileSystem()
	_, err := cmdutils.ExecuteCommand(t, New(fs), os.Stdin)
	assert.NoError(t, err)
}
