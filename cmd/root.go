package cmd

import (
	"errors"
	"fmt"
	"os"

	"code-intelligence.com/cifuzz/pkg/cmdutils"
	"code-intelligence.com/cifuzz/pkg/storage"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cifuzz",
	Short: "#tbd",
	// We are using our the custom ErrSilent instead to support a more specific
	// error handling
	SilenceErrors: true,
	SilenceUsage:  true,
}

var fs *afero.Afero

func init() {
	fs = storage.WrapFileSystem()
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {

		if !errors.Is(err, cmdutils.ErrSilent) {
			fmt.Println(rootCmd.UsageString())
			fmt.Printf("%+v \n", err)
		}

		os.Exit(1)
	}
}
