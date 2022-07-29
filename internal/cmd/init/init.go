package init

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"code-intelligence.com/cifuzz/internal/config"
	"code-intelligence.com/cifuzz/pkg/cmdutils"
	"code-intelligence.com/cifuzz/pkg/log"
)

type cmdOpts struct {
}

func New() *cobra.Command {
	opts := &cmdOpts{}
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Set up a project for use with cifuzz",
		Long: "This command sets up a project for use with cifuzz, creating a " +
			"`cifuzz.yaml` config file.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd, args, opts)
		},
	}

	cmdutils.DisableConfigCheck(initCmd)

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
		if errors.Is(err, os.ErrExist) && configpath != "" {
			log.Warnf("Config already exists in %s", configpath)
			err = cmdutils.ErrSilent
		}
		log.Error(err, "Failed to create config")
		return err
	}
	log.Successf("Configuration saved in %s", configpath)

	printBuildSystemInstructions(cwd)

	log.Print(`
Use 'cifuzz create' to create your first fuzz test.`)
	return
}

func printBuildSystemInstructions(cwd string) {
	// Printing build system instructions is best-effort: Do not fail on errors.
	cfg, err := config.ReadProjectConfig(cwd)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Debug(err)
		}
		return
	}
	if cfg.BuildSystem == config.BuildSystemCMake {
		log.Print(`
Enable fuzz testing in your CMake project by adding the following lines
to the top-level CMakeLists.txt before any add_subdirectory(...),
add_library(...) or add_executable(...) calls:

    find_package(cifuzz)
    enable_fuzz_testing()`)
	}

}
