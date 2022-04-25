package cmd

import (
	"errors"
	"testing"

	"code-intelligence.com/cifuzz/pkg/cmdutils"
	"github.com/stretchr/testify/assert"
)

func TestInitCmd(t *testing.T) {
	args := []string{
		"init",
	}
	_, err := ExecuteCommand(t, args...)
	assert.NoError(t, err)

	// second execution should return a ErrSilent as the config file should aready exists
	_, err = ExecuteCommand(t, args...)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, cmdutils.ErrSilent))
}
