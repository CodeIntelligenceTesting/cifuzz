package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunCmd(t *testing.T) {
	args := []string{
		"run",
	}
	_, err := ExecuteCommand(t, args...)
	assert.Error(t, err)
}
