package main

import (
	"code-intelligence.com/cifuzz/pkg/cmdutils"
	"github.com/pkg/errors"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	installer_bundle "code-intelligence.com/cifuzz/installer-bundle"
	"code-intelligence.com/cifuzz/pkg/install"
	"code-intelligence.com/cifuzz/pkg/log"
)

func main() {
	var installDir string
	fs := &installer_bundle.Bundle

	cmd := &cobra.Command{
		Use:   "installer",
		Short: "Install cifuzz",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := install.ExtractBundle(installDir, fs)
			if err != nil {
				log.Error(err, err.Error())
				return cmdutils.ErrSilent
			}

			binDir := filepath.Join(installDir, "bin")
			install.PrintPathInstructions(binDir)

			return nil
		},
	}

	cmd.Flags().StringVarP(&installDir, "install-dir", "i", "~/cifuzz", "The directory to install cifuzz in")

	err := cmd.Execute()
	if err != nil {
		var silentErr *cmdutils.SilentError
		if !errors.As(err, &silentErr) {
			log.Error(err, err.Error())
		}
		os.Exit(1)
	}
}
