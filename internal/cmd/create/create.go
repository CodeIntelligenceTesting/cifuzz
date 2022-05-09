package create

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"code-intelligence.com/cifuzz/internal/config"
	"code-intelligence.com/cifuzz/pkg/cmdutils"
	"code-intelligence.com/cifuzz/pkg/dialog"
	"code-intelligence.com/cifuzz/pkg/storage"
	"code-intelligence.com/cifuzz/pkg/stubs"
	"code-intelligence.com/cifuzz/pkg/workarounds"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
)

// map of supported test types -> label:value
var supportedTestTypes = map[string]string{
	"C/C++": string(config.CPP),
}

func New(fs *afero.Afero) *cobra.Command {

	createCmd := &cobra.Command{
		Use:   fmt.Sprintf("create [%s]", strings.Join(maps.Values(supportedTestTypes), "|")),
		Short: "Create a new fuzz test",
		Long:  "Creates a template for a new fuzz test",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd, args, fs)
		},
		Args:      cobra.MatchAll(cobra.MaximumNArgs(1), cobra.OnlyValidArgs),
		ValidArgs: maps.Values(supportedTestTypes),
	}

	createCmd.Flags().StringP("out", "o", "", "The directory where the new fuzz test should be created")
	createCmd.Flags().StringP("name", "n", "", "The filename of the created stub")

	return createCmd
}

func run(cmd *cobra.Command, args []string, fs *afero.Afero) (err error) {
	// get test type
	testType, err := getTestType(cmd, args)
	if err != nil {
		return err
	}
	dialog.DebugF("Selected fuzz test type: %s\n", testType)

	// check for the --out flag
	outFlag, err := cmd.Flags().GetString("out")
	if err != nil {
		return errors.WithStack(err)
	}
	// get output directory
	outDir, err := storage.GetOutDir(outFlag, fs)
	if workarounds.IsPermission(errors.Cause(err)) {
		dialog.ErrorF(err, "unable to write to given out directory, permission denied: %s\n", outDir)
		return cmdutils.WrapSilentError(err)
	} else if err != nil {
		return err
	}
	dialog.DebugF("Using output directory: %s\n", outDir)

	filename, err := determineFilename(cmd, outDir, testType, fs)
	if err != nil {
		return err
	}
	dialog.DebugF("Selected filename %s\n", filename)

	// create stub
	stubPath := filepath.Join(outDir, filename)
	if err := stubs.Create(stubPath, testType, fs); err != nil {
		if os.IsExist(errors.Cause(err)) {
			dialog.ErrorF(err, "Unable to created fuzz test, file already exists %s\n", stubPath)
			return cmdutils.WrapSilentError(err)
		}
	}

	// show success message
	dialog.SuccessF("Fuzz test stub created at %s\n", stubPath)
	dialog.Info(`
Note: Fuzz tests can be put anywhere in your repository, but it makes sense to keep them close to the tested code - just like regular unit tests.`)

	return
}

// getTestType returns the test type (selected by argument or input dialog)
func getTestType(cmd *cobra.Command, args []string) (config.FuzzTestType, error) {
	if len(args) == 1 {
		return config.FuzzTestType(args[0]), nil
	}
	userSelectedType, err := dialog.Select("Select type of the fuzz test", supportedTestTypes, cmd.InOrStdin())
	if err != nil {
		fmt.Printf("%+v \n", err)
		return "", cmdutils.WrapSilentError(err)
	}
	return config.FuzzTestType(userSelectedType), nil
}

func determineFilename(cmd *cobra.Command, outDir string, testType config.FuzzTestType, fs *afero.Afero) (string, error) {
	// check for the --name flag
	nameFlag, err := cmd.Flags().GetString("name")
	if err != nil {
		return "", errors.WithStack(err)
	}

	if nameFlag != "" {
		return nameFlag, nil
	}

	suggestedFilename, err := stubs.SuggestFilename(outDir, testType, fs)
	if err != nil {
		// as this error only results in a missing filename suggestion we just show
		// it but do not stop the application
		dialog.ErrorF(err, "unable to suggest filename for given test type %s", testType)
	}

	filename, err := dialog.Input(
		"Please enter filename",
		suggestedFilename,
		cmd.InOrStdin(),
	)
	if err != nil {
		return "", err
	}
	// TODO validate filename

	return filename, nil
}
