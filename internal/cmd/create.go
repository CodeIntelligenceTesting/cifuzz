package cmd

import (
	"fmt"
	"os"
	"strings"

	"code-intelligence.com/cifuzz/pkg/cmdutils"
	"code-intelligence.com/cifuzz/pkg/out"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
)

const (
	CPP string = "cpp"
	//JAVA string = "java"
	//GO   string = "go"
)

// map of supported types -> label:value
var types = map[string]string{
	"C/C++": CPP,
	//"Java":  JAVA,
	//"Go":    GO,
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
	rootCmd.AddCommand(createCmd)
}

func runCreateCommand(cmd *cobra.Command, args []string) (err error) {
	// get the type of the fuzz target
	var targetType string
	if len(args) == 1 {
		targetType = args[0]
	} else {
		targetType, err = out.Select("Select type of the fuzz test", types)
		if err != nil {
			fmt.Printf("%+v \n", err)
			return cmdutils.WrapSilentError(err)
		}
	}
	out.Debug("Selected fuzz test type: %s", targetType)

	cwd, err := os.Getwd()
	if err != nil {
		return errors.WithStack(err)
	}
	out.Debug("Using current working directory: %s", cwd)

  out.Warn("TODO: create stub files")

	out.Success("Fuzz test stub created at %s", cwd)
	out.Info(`
Note: Fuzz tests can be put anywhere in your repository, 
but it makes sense to keep them close to the tested code - just like regular unit tests.`)

	return
}
