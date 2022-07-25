package create

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"

	"code-intelligence.com/cifuzz/internal/config"
	"code-intelligence.com/cifuzz/pkg/cmdutils"
	"code-intelligence.com/cifuzz/pkg/dialog"
	"code-intelligence.com/cifuzz/pkg/log"
	"code-intelligence.com/cifuzz/pkg/stubs"
)

type cmdOpts struct {
	outputPath string
	testType   config.FuzzTestType

	config *config.Config
}

// map of supported test types -> label:value
var supportedTestTypes = map[string]string{
	"C/C++": string(config.CPP),
}

func New(config *config.Config) *cobra.Command {
	opts := &cmdOpts{config: config}

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

	createCmd.Flags().StringVarP(&opts.outputPath, "output", "o", "", "File path of new fuzz test")

	return createCmd
}

func run(cmd *cobra.Command, args []string, opts *cmdOpts) (err error) {
	// get test type
	opts.testType, err = getTestType(args)
	if err != nil {
		return err
	}
	log.Debugf("Selected fuzz test type: %s", opts.testType)

	if opts.outputPath == "" {
		opts.outputPath, err = stubs.FuzzTestFilename(opts.testType)
		if err != nil {
			return err
		}
	}
	log.Debugf("Output path: %s", opts.outputPath)

	// create stub
	err = stubs.Create(opts.outputPath, opts.testType)
	if err != nil {
		log.Errorf(err, "Failed to create fuzz test stub %s: %s", opts.outputPath, err.Error())
		return cmdutils.ErrSilent
	}

	// show success message
	log.Successf("Created fuzz test stub %s", opts.outputPath)
	log.Info(`
Note: Fuzz tests can be put anywhere in your repository, but it makes sense to keep them close to the tested code - just like regular unit tests.`)

	printBuildSystemInstructions(opts.config.BuildSystem, filepath.Base(opts.outputPath))

	return
}

// getTestType returns the test type (selected by argument or input dialog)
func getTestType(args []string) (config.FuzzTestType, error) {
	if len(args) == 1 {
		return config.FuzzTestType(args[0]), nil
	}
	userSelectedType, err := dialog.Select("Select type of the fuzz test", supportedTestTypes)
	if err != nil {
		fmt.Printf("%+v \n", err)
		return "", cmdutils.ErrSilent
	}
	return config.FuzzTestType(userSelectedType), nil
}

func printBuildSystemInstructions(buildSystem, filename string) {
	// Printing build system instructions is best-effort: Do not fail on errors.
	if buildSystem == config.BuildSystemCMake {
		log.Infof(`
Create a CMake target for the fuzz test as follows - it behaves just like a regular add_executable(...):

    add_fuzz_test(%s %s)`, strings.TrimSuffix(filename, filepath.Ext(filename)), filename)
	}
}
