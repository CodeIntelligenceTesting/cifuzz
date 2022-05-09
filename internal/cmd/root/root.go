package root

import (
	"fmt"
	"os"

	buildCmd "code-intelligence.com/cifuzz/internal/cmd/build"
	createCmd "code-intelligence.com/cifuzz/internal/cmd/create"
	initCmd "code-intelligence.com/cifuzz/internal/cmd/init"
	runCmd "code-intelligence.com/cifuzz/internal/cmd/run"
	"code-intelligence.com/cifuzz/pkg/cmdutils"
	"code-intelligence.com/cifuzz/pkg/storage"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewCmdRoot(fs *afero.Afero) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "cifuzz",
		Short: "#tbd",
		// We are using our custom ErrSilent instead to support a more specific
		// error handling
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	rootCmd.PersistentFlags().BoolP("verbose", "v", false,
		"Show more verbose output, can be helpful for debugging problems")
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))

	rootCmd.AddCommand(initCmd.NewCmdInit(fs))
	rootCmd.AddCommand(createCmd.NewCmdCreate(fs))
	rootCmd.AddCommand(buildCmd.NewCmdBuild())
	rootCmd.AddCommand(runCmd.NewCmdRun())

	return rootCmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	fs := storage.WrapFileSystem()
	rootCmd := NewCmdRoot(fs)
	if cmd, err := rootCmd.ExecuteC(); err != nil {

		// Errors that are not ErrSilent are not expected and we want to show their full stacktrace
		if !errors.Is(err, cmdutils.ErrSilent) {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "%+v\n", err)
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), cmd.UsageString())
		}

		os.Exit(1)
	}
}
