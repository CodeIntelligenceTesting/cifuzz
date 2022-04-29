package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"code-intelligence.com/cifuzz/pkg/storage"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	fs = storage.NewMemFileSystem()
	m.Run()
}

func ExecuteCommand(t *testing.T, in io.Reader, args ...string) (string, error) {
	t.Helper()

	rootCmd.SetIn(in)
	rootCmd.SetArgs(args)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	// overwrite the pre run function to avoid initilization the command settings/environment
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if fs == nil {
			fs = storage.NewMemFileSystem()
		}
	}
	err := rootCmd.Execute()
	return strings.TrimSpace(buf.String()), err
}

func TestRootCmd(t *testing.T) {
	args := []string{""}
	_, err := ExecuteCommand(t, os.Stdin, args...)
	assert.NoError(t, err)
}
