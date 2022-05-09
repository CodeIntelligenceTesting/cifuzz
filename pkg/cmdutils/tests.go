package cmdutils

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func ExecuteCommand(t *testing.T, cmd *cobra.Command, in io.Reader, args ...string) (string, error) {

	cmd.SetIn(in)
	cmd.SetArgs(args)

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	err := cmd.Execute()
	return strings.TrimSpace(buf.String()), err
}
