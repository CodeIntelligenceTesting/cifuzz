package dialog

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
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
	assert.NoError(t, err)

	_, err = w.Write(input)
	assert.NoError(t, err)
	w.Close()

	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	os.Stdin = r

	userInput, err := Input("Test", "default")
	assert.NoError(t, err)
	assert.Equal(t, "my input", userInput)
}

// Should return default value if user just presses "enter"
func TestInput_Default(t *testing.T) {
	input := []byte("\n")
	r, w, err := os.Pipe()
	assert.NoError(t, err)

	_, err = w.Write(input)
	assert.NoError(t, err)
	w.Close()

	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	os.Stdin = r

	userInput, err := Input("Test", "default")
	assert.NoError(t, err)
	assert.Equal(t, "default", userInput)
}
