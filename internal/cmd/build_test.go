package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildCmd(t *testing.T) {
	args := []string{
		"build",
	}
	_, err := ExecuteCommand(t, os.Stdin, args...)
	assert.Error(t, err)
}
