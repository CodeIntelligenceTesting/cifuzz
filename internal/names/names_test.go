package names

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeterminsticName(t *testing.T) {
	assert.Equal(t, "stupefied_montalcini", GetDeterministicName([]byte("fc75")))
	assert.Equal(t, "gifted_dijkstra", GetDeterministicName([]byte("fc7598c04e2ffdc36c3ff70428fd98912ffb07a8")))
	assert.Equal(t, "condescending_chatterjee", GetDeterministicName([]byte("")))
}
