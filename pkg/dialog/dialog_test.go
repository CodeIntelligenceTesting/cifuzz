package dialog

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelect(t *testing.T) {
	input := []byte("\n")

	r, w, err := os.Pipe()
	require.NoError(t, err)

	_, err = w.Write(input)
	require.NoError(t, err)
	w.Close()

	items := map[string]string{
		"Item No1": "item1",
	}

	userInput, err := Select("Test", items, r)
	assert.NoError(t, err)
	assert.Equal(t, "item1", userInput)
}

func TestInputFilename(t *testing.T) {
	input := []byte("my input\n")
	r, w, err := os.Pipe()
	require.NoError(t, err)

	_, err = w.Write(input)
	require.NoError(t, err)
	w.Close()

	// The zsh vared command we're using when the current shell is zsh
	// uses /dev/tty for reading instead of stdin, which causes this
	// test to block. To avoid that, we're unsetting the SHELL variable
	// in that case to make InputFilename fall back to just reading a
	// line from stdin.
	if filepath.Base(os.Getenv("SHELL")) == "zsh" {
		_ = os.Unsetenv("SHELL")
	}

	userInput, err := InputFilename(r, "Test", "default")
	assert.NoError(t, err)
	assert.Equal(t, "my input", userInput)
}

// Should return default value if user just presses "enter"
func TestInputFilename_Default(t *testing.T) {
	input := []byte("\n")
	r, w, err := os.Pipe()
	require.NoError(t, err)

	_, err = w.Write(input)
	require.NoError(t, err)
	w.Close()

	// The zsh vared command we're using when the current shell is zsh
	// uses /dev/tty for reading instead of stdin, which causes this
	// test to block. To avoid that, we're unsetting the SHELL variable
	// in that case to make InputFilename fall back to just reading a
	// line from stdin.
	if filepath.Base(os.Getenv("SHELL")) == "zsh" {
		_ = os.Unsetenv("SHELL")
	}

	userInput, err := InputFilename(r, "Test", "default")
	assert.NoError(t, err)
	assert.Equal(t, "default", userInput)
}
