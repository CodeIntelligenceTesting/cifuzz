package cmd

import (
	"os"

	"code-intelligence.com/cifuzz/internal/config"
	"code-intelligence.com/cifuzz/pkg/cmdutils"
	"code-intelligence.com/cifuzz/pkg/out"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Set up a project for use with cifuzz",
	Long: "This command sets up a project for use with cifuzz, creating a " +
		"`.cifuzz.yaml` config file.",
	Args: cobra.NoArgs,
	RunE: runInitCommand,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInitCommand(cmd *cobra.Command, args []string) (err error) {
	cwd, err := os.Getwd()
	if err != nil {
		return errors.WithStack(err)
	}
	out.Debug("Using current working directory: %s", cwd)

	configpath, err := config.CreateProjectConfig(cwd, fs)
	if err != nil {
		// explicitly inform the user about an existing config file
		if os.IsExist(errors.Cause(err)) && configpath != "" {
			out.Warn("Config already exists in %s", configpath)
			err = cmdutils.WrapSilentError(err)
		}
		out.Error(err, "Failed to create config")
		return err
	}

	out.Success("Configuration saved in %s", configpath)
	out.Info("\nUse 'cifuzz create' to create your first fuzz test")
	return
}
