package init

import (
	"os"

	"code-intelligence.com/cifuzz/internal/config"
	"code-intelligence.com/cifuzz/pkg/cmdutils"
	"code-intelligence.com/cifuzz/pkg/dialog"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

func New(fs *afero.Afero) *cobra.Command {
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Set up a project for use with cifuzz",
		Long: "This command sets up a project for use with cifuzz, creating a " +
			"`.cifuzz.yaml` config file.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd, args, fs)
		},
	}

	return initCmd
}

func run(cmd *cobra.Command, args []string, fs *afero.Afero) (err error) {
	cwd, err := os.Getwd()
	if err != nil {
		return errors.WithStack(err)
	}
	dialog.DebugF("Using current working directory: %s\n", cwd)

	configpath, err := config.CreateProjectConfig(cwd, fs)
	if err != nil {
		// explicitly inform the user about an existing config file
		if os.IsExist(errors.Cause(err)) && configpath != "" {
			dialog.WarnF("Config already exists in %s\n", configpath)
			err = cmdutils.WrapSilentError(err)
		}
		dialog.Error(err, "Failed to create config")
		return err
	}

	dialog.SuccessF("Configuration saved in %s", configpath)
	dialog.Info(`
Use 'cifuzz create' to create your first fuzz test`)
	return
}
