package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateCmd(t *testing.T) {
	args := []string{
		"create",
	}
	_, err := ExecuteCommand(t, args...)
	assert.Error(t, err)
}
