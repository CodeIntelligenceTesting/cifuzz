package cmd

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "#TODO",
	Long:  "#TODO",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("Not implemented")
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
