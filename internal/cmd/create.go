package cmd

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
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
)

var createCmd = &cobra.Command{
	Use:       fmt.Sprintf("create [%s]", strings.Join(maps.Values(config.SupportedTypes), "|")),
	Short:     "Create a new fuzz target",
	Long:      "This commands helps creating a new fuzz target",
	RunE:      runCreateCommand,
	Args:      cobra.MatchAll(cobra.MaximumNArgs(1), cobra.OnlyValidArgs),
	ValidArgs: maps.Values(config.SupportedTypes),
}

func init() {
	createCmd.Flags().StringP("out", "o", "", "The location where the new fuzz test should be created")
	createCmd.Flags().StringP("name", "n", "", "The filename of the created stub")
	rootCmd.AddCommand(createCmd)
}

func runCreateCommand(cmd *cobra.Command, args []string) (err error) {
	// get test type
	testType, err := getTestType(cmd, args)
	if err != nil {
		return err
	}
	dialog.Debug("Selected fuzz test type: %s", targetType)

	// check for the --out flag
	outFlag, err := cmd.Flags().GetString("out")
	if err != nil {
		return errors.WithStack(err)
	}
	// get output directory
	outDir, err := storage.GetOutDir(outFlag, fs)
	if os.IsPermission(errors.Cause(err)) {
		dialog.Error(err, "unable to write to given out directory, permission denied: %s", outDir)
		return cmdutils.WrapSilentError(err)
	} else if err != nil {
		return err
	}
	dialog.Debug("Using output directory: %s", outDir)

	filename, err := getFilename(cmd, outDir, targetType)
	if err != nil {
		return err
	}
	dialog.Debug("Selected filename %s", filename)

	// create stub
	stubPath := filepath.Join(outDir, filename)
	if err := stubs.Create(stubPath, targetType, fs); err != nil {
		if os.IsExist(errors.Cause(err)) {
			dialog.Error(err, "Unable to created fuzz test, file already exists %s", stubPath)
			return cmdutils.WrapSilentError(err)
		}
	}

	// show success message
	dialog.Success("Fuzz test stub created at %s", stubPath)
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
	return
}

func getFilename(cmd *cobra.Command, outDir string, targetType config.TargetType) (string, error) {
	// check for the --name flag
	nameFlag, err := cmd.Flags().GetString("name")
	if err != nil {
		return "", errors.WithStack(err)
	}

	if nameFlag != "" {
		return nameFlag, nil
	}

	// get filename
	filename, err := dialog.Input("Please enter filename", stubs.SuggestFilename(outDir, targetType, fs))
	if err != nil {
		return "", err
	}
	// TODO validate filename

	return filename, nil
}
