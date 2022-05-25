package dialog

import (
	"os"
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

func TestInput(t *testing.T) {
	input := []byte("my input\n")
	r, w, err := os.Pipe()
	require.NoError(t, err)

	_, err = w.Write(input)
	require.NoError(t, err)
	w.Close()

	userInput, err := Input("Test", "default", r)
	assert.NoError(t, err)
	assert.Equal(t, "my input", userInput)
}

// Should return default value if user just presses "enter"
func TestInput_Default(t *testing.T) {
	input := []byte("\n")
	r, w, err := os.Pipe()
	require.NoError(t, err)

	_, err = w.Write(input)
	require.NoError(t, err)
	w.Close()

	userInput, err := Input("Test", "default", r)
	assert.NoError(t, err)
	assert.Equal(t, "default", userInput)
}
