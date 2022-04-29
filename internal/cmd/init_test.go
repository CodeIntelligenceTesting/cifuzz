package cmd

import (
	"os"
	"testing"

	"code-intelligence.com/cifuzz/pkg/cmdutils"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestInitCmd(t *testing.T) {
	args := []string{
		"init",
	}
	_, err := ExecuteCommand(t, os.Stdin, args...)
	assert.NoError(t, err)

	// second execution should return a ErrSilent as the config file should aready exists
	_, err = ExecuteCommand(t, os.Stdin, args...)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, cmdutils.ErrSilent))
}
