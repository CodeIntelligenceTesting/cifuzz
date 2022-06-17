package cmdutils

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestNeedsConfig(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	require.True(t, NeedsConfig(cmd))

	DisableConfigCheck(cmd)
	require.False(t, NeedsConfig(cmd))
}
