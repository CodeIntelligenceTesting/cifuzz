package cmd

import (
	"fmt"
	"os"
	"strings"

	"code-intelligence.com/cifuzz/pkg/cmdutils"
	"code-intelligence.com/cifuzz/pkg/out"
	"code-intelligence.com/cifuzz/pkg/storage"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
)

const (
	CPP  string = "cpp"
	JAVA string = "java"
	GO   string = "go"
)

// map of supported types -> label:value
var types = map[string]string{
	"C/C++": CPP,
	"Java":  JAVA,
	"Go":    GO,
}

var createCmd = &cobra.Command{
	Use:       fmt.Sprintf("create [%s]", strings.Join(maps.Values(types), "|")),
	Short:     "Create a new fuzz target",
	Long:      "This commands helps creating a new fuzz target",
	RunE:      runCreateCommand,
	Args:      cobra.MatchAll(cobra.MaximumNArgs(1), cobra.OnlyValidArgs),
	ValidArgs: maps.Values(types),
}

func init() {
	createCmd.Flags().StringP("out", "o", "", "The location where the new fuzz test should be created")
	rootCmd.AddCommand(createCmd)
}

func runCreateCommand(cmd *cobra.Command, args []string) (err error) {
	// get target type
	targetType, err := getTargetType(cmd, args)
	if err != nil {
		return err
	}
	out.Debug("Selected fuzz test type: %s", targetType)

	// check for the --out flag
	outFlag, err := cmd.Flags().GetString("out")
	if err != nil {
		return errors.WithStack(err)
	}
	// get output directory
	outDir, err := storage.GetOutDir(outFlag, fs)
	if os.IsPermission(errors.Cause(err)) {
		out.Error(err, "unable to write to given out directory, permission denied: %s", outDir)
		return cmdutils.WrapSilentError(err)
	} else if err != nil {
		return err
	}

	out.Debug("Using output directory: %s", outDir)

	out.Warn("TODO: create stub files")

	// show success message
	out.Success("Fuzz test stub created at %s", outDir)
	out.Info(`
Note: Fuzz tests can be put anywhere in your repository, but it makes sense to keep them close to the tested code - just like regular unit tests.`)

	return
}

// getTargetType returns the target type (selected by argument or input dialog)
func getTargetType(cmd *cobra.Command, args []string) (targetType string, err error) {
	// get the type of the fuzz target
	if len(args) == 1 {
		targetType = args[0]
	} else {
		targetType, err = out.Select("Select type of the fuzz test", types)
		if err != nil {
			fmt.Printf("%+v \n", err)
			return "", cmdutils.WrapSilentError(err)
		}
	}
	return
}
