package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateCmd(t *testing.T) {
	args := []string{
		"create",
		"cpp",
	}
	_, err := ExecuteCommand(t, args...)
	assert.NoError(t, err)
}

func TestCreateCmd_InvalidType(t *testing.T) {
	args := []string{
		"create",
		"foo",
	}
	_, err := ExecuteCommand(t, args...)
	assert.Error(t, err)
}

func TestCreateCmd_DefaultDir(t *testing.T) {
	t.Skip()
}

func TestCreateCmd_OutDir(t *testing.T) {
	t.Skip()
}
