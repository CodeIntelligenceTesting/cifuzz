package create

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"

	"code-intelligence.com/cifuzz/internal/config"
	"code-intelligence.com/cifuzz/pkg/cmdutils"
	"code-intelligence.com/cifuzz/pkg/dialog"
	"code-intelligence.com/cifuzz/pkg/log"
	"code-intelligence.com/cifuzz/pkg/storage"
	"code-intelligence.com/cifuzz/pkg/stubs"
)

type cmdOpts struct {
	outDir   string
	filename string
	testType config.FuzzTestType
}

// map of supported test types -> label:value
var supportedTestTypes = map[string]string{
	"C/C++": string(config.CPP),
}

func New() *cobra.Command {
	opts := &cmdOpts{}

	createCmd := &cobra.Command{
		Use:   fmt.Sprintf("create [%s]", strings.Join(maps.Values(supportedTestTypes), "|")),
		Short: "Create a new fuzz test",
		Long:  "Creates a template for a new fuzz test",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd, args, opts)
		},
		Args:      cobra.MatchAll(cobra.MaximumNArgs(1), cobra.OnlyValidArgs),
		ValidArgs: maps.Values(supportedTestTypes),
	}

	createCmd.Flags().StringVarP(&opts.outDir, "out", "o", "", "The directory where the new fuzz test should be created")
	createCmd.Flags().StringVarP(&opts.filename, "name", "n", "", "The filename of the created stub")

	return createCmd
}

func run(cmd *cobra.Command, args []string, opts *cmdOpts) (err error) {
	// get test type
	opts.testType, err = getTestType(args, cmd.InOrStdin())
	if err != nil {
		return err
	}
	log.Debugf("Selected fuzz test type: %s", opts.testType)

	// get output directory
	opts.outDir, err = storage.GetOutDir(opts.outDir)
	if errors.Is(err, os.ErrPermission) {
		log.Errorf(err, "unable to write to given out directory, permission denied: %s\n", opts.outDir)
		return cmdutils.ErrSilent
	} else if err != nil {
		return err
	}
	log.Debugf("Using output directory: %s", opts.outDir)

	opts.filename, err = determineFilename(opts, cmd.InOrStdin())
	if err != nil {
		return err
	}
	log.Debugf("Selected filename %s", opts.filename)

	// create stub
	stubPath := filepath.Join(opts.outDir, opts.filename)
	err = stubs.Create(stubPath, opts.testType)
	if err != nil {
		log.Errorf(err, "Failed to create fuzz test stub %s: %s", stubPath, err.Error())
		return cmdutils.ErrSilent
	}

	// show success message
	log.Successf("Created fuzz test stub %s", stubPath)
	log.Info(`
Note: Fuzz tests can be put anywhere in your repository, but it makes sense to keep them close to the tested code - just like regular unit tests.`)

	printBuildSystemInstructions(opts.filename)

	return
}

// getTestType returns the test type (selected by argument or input dialog)
func getTestType(args []string, stdin io.Reader) (config.FuzzTestType, error) {
	if len(args) == 1 {
		return config.FuzzTestType(args[0]), nil
	}
	userSelectedType, err := dialog.Select("Select type of the fuzz test", supportedTestTypes, stdin)
	if err != nil {
		fmt.Printf("%+v \n", err)
		return "", cmdutils.ErrSilent
	}
	return config.FuzzTestType(userSelectedType), nil
}

func determineFilename(opts *cmdOpts, stdin io.Reader) (string, error) {
	// check for the --name flag
	if opts.filename != "" {
		return opts.filename, nil
	}

	suggestedFilename, err := stubs.SuggestFilename(opts.outDir, opts.testType)
	if err != nil {
		// as this error only results in a missing filename suggestion we just show
		// it but do not stop the application
		log.Errorf(err, "unable to suggest filename for given test type %s", opts.testType)
	}

	filename, err := dialog.Input(
		"Please enter filename",
		suggestedFilename,
		stdin,
	)
	if err != nil {
		return "", err
	}
	// TODO validate filename

	return filename, nil
}

func printBuildSystemInstructions(filename string) {
	// Printing build system instructions is best-effort: Do not fail on errors.
	projectDir, err := config.FindProjectDir()
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Debug(err)
		}
		return
	}
	cfg, err := config.ReadProjectConfig(projectDir)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Debug(err)
		}
		return
	}
	if cfg.BuildSystem == config.BuildSystemCMake {
		log.Infof(`
Create a CMake target for the fuzz test as follows - it behaves just like a regular add_executable(...):

    add_fuzz_test(%s %s)`, strings.TrimSuffix(filename, filepath.Ext(filename)), filename)
	}
}
