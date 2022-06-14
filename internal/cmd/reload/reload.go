package run

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"code-intelligence.com/cifuzz/internal/build/cmake"
	"code-intelligence.com/cifuzz/internal/config"
	"code-intelligence.com/cifuzz/pkg/cmdutils"
	"code-intelligence.com/cifuzz/pkg/log"
)

// TODO: The reload command allows to reload the fuzz test names used
//       for autocompletion from the cmake config. It's only meant as a
//       temporary solution until we find a better solution.
type reloadCmd struct {
	*cobra.Command

	projectDir string
}

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reload [flags]",
		Short: "Reload fuzz test metadata",
		// TODO: Write long description
		Long: "",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			cmd := reloadCmd{Command: c}
			return cmd.run()
		},
	}
	return cmd
}

func (c *reloadCmd) run() error {
	var err error

	c.projectDir, err = config.FindProjectDir()
	if errors.Is(err, os.ErrNotExist) {
		// The project directory doesn't exist, this is an expected
		// error, so we print it and return a silent error to avoid
		// printing a stack trace
		log.Error(err, fmt.Sprintf("%s\nUse 'cifuzz init' to set up a project for use with cifuzz.", err.Error()))
		return cmdutils.ErrSilent
	}
	if err != nil {
		return err
	}

	conf, err := config.ReadProjectConfig(c.projectDir)
	if err != nil {
		return err
	}

	if conf.BuildSystem == config.BuildSystemCMake {
		return c.reloadCMake()
	} else if conf.BuildSystem == config.BuildSystemUnknown {
		// Nothing to reload for unknown build system
		return nil
	} else {
		return errors.Errorf("Unsupported build system \"%s\"", conf.BuildSystem)
	}
}

func (c *reloadCmd) reloadCMake() error {
	// TODO: Make these configurable
	engine := "libfuzzer"
	sanitizers := []string{"address", "undefined"}

	builder, err := cmake.NewBuilder(&cmake.BuilderOptions{
		ProjectDir: c.projectDir,
		Engine:     engine,
		Sanitizers: sanitizers,
		Stdout:     c.OutOrStdout(),
		Stderr:     c.ErrOrStderr(),
	})
	if err != nil {
		return err
	}

	err = builder.Configure()
	if err != nil {
		return err
	}
	return nil
}
