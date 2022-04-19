package cmd

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new fuzz target",
	Long:  "This commands helps creating a new fuzz target",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("Not implemented")
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
}
