package init

import (
	"os"

	"code-intelligence.com/cifuzz/internal/config"
	"code-intelligence.com/cifuzz/pkg/cmdutils"
	"code-intelligence.com/cifuzz/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type cmdOpts struct {
}

func New() *cobra.Command {
	opts := &cmdOpts{}
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Set up a project for use with cifuzz",
		Long: "This command sets up a project for use with cifuzz, creating a " +
			"`.cifuzz.yaml` config file.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd, args, opts)
		},
	}

	return initCmd
}

func run(cmd *cobra.Command, args []string, opts *cmdOpts) (err error) {
	cwd, err := os.Getwd()
	if err != nil {
		return errors.WithStack(err)
	}
	log.Debugf("Using current working directory: %s", cwd)

	configpath, err := config.CreateProjectConfig(cwd)
	if err != nil {
		// explicitly inform the user about an existing config file
		if os.IsExist(errors.Cause(err)) && configpath != "" {
			log.Warnf("Config already exists in %s", configpath)
			err = cmdutils.WrapSilentError(err)
		}
		log.Error(err, "Failed to create config")
		return err
	}

	log.Successf("Configuration saved in %s", configpath)
	log.Info(`
Use 'cifuzz create' to create your first fuzz test`)
	return
}
