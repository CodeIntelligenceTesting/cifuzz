package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunCmd(t *testing.T) {
	args := []string{
		"run",
	}
	_, err := ExecuteCommand(t, os.Stdin, args...)
	assert.Error(t, err)
}
