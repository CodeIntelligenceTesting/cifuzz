package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitCmd(t *testing.T) {
	args := []string{
		"init",
	}
	_, err := ExecuteCommand(t, args...)
	assert.NoError(t, err)
}
