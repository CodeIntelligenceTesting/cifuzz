package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildCmd(t *testing.T) {
	args := []string{
		"build",
	}
	_, err := ExecuteCommand(t, args...)
	assert.Error(t, err)
}
