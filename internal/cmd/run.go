package cmd

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewCmdRun() *cobra.Command {
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "#TODO",
		Long:  "#TODO",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("Not implemented")
		},
	}
	return runCmd
}
