package cmd

import (
	"os"

	"code-intelligence.com/cifuzz/pkg/storage"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cifuzz",
	Short: "#tbd",
}

var fs *afero.Afero

func init() {
	fs = storage.WrapFileSystem()
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		//TODO add logging?
		os.Exit(1)
	}
}
