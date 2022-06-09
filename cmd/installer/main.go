package main

import (
	"os"

	"github.com/spf13/cobra"

	"code-intelligence.com/cifuzz/pkg/log"
	"code-intelligence.com/cifuzz/tools/install"
)

func main() {
	opts := &install.Options{}

	cmd := &cobra.Command{
		Use:   "installer",
		Short: "Install cifuzz",
		Run: func(cmd *cobra.Command, args []string) {
			installer, err := install.NewInstaller(opts)
			if err != nil {
				log.Error(err, err.Error())
				os.Exit(1)
			}
			err = installer.InstallCIFuzzAndDeps()
			if err != nil {
				log.Error(err, err.Error())
				os.Exit(1)
			}
			installer.PrintPathInstructions()
		},
	}

	cmd.Flags().StringVarP(&opts.InstallDir, "install-dir", "i", "~/cifuzz", "The directory to install cifuzz in")

	err := cmd.Execute()
	if err != nil {
		log.Error(err, err.Error())
		os.Exit(1)
	}
}
