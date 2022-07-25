package dialog

import (
	"bytes"
	"os"
	"testing"

	"atomicgo.dev/keyboard"
	"atomicgo.dev/keyboard/keys"
	"github.com/pterm/pterm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelect(t *testing.T) {
	var outBuf bytes.Buffer
	pterm.SetDefaultOutput(&outBuf)

	defer pterm.SetDefaultOutput(os.Stdout)

	go func() {
		require.NoError(t, keyboard.SimulateKeyPress(keys.Down))
		require.NoError(t, keyboard.SimulateKeyPress(keys.Down))
		require.NoError(t, keyboard.SimulateKeyPress(keys.Up))
		require.NoError(t, keyboard.SimulateKeyPress(keys.Enter))
	}()

	items := map[string]string{
		"Item No1": "item1",
		"Item No2": "item2",
		"Item No3": "item3",
		"Item No4": "item4",
	}
	userInput, err := Select("Test", items)
	require.NoError(t, err)
	assert.Equal(t, "item2", userInput)
}
