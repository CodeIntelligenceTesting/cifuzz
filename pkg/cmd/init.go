package cmd

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Set up a project for use with cifuzz",
	Long: "This command sets up a project for use with cifuzz, creating a " +
		"`.cifuzz.yaml` config file.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("Not implemented")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
