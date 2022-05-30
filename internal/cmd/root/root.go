package root

import (
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	buildCmd "code-intelligence.com/cifuzz/internal/cmd/build"
	createCmd "code-intelligence.com/cifuzz/internal/cmd/create"
	initCmd "code-intelligence.com/cifuzz/internal/cmd/init"
	runCmd "code-intelligence.com/cifuzz/internal/cmd/run"
	"code-intelligence.com/cifuzz/pkg/cmdutils"
	"code-intelligence.com/cifuzz/pkg/log"
)

func New() *cobra.Command {
	var workdir string

	rootCmd := &cobra.Command{
		Use:   "cifuzz",
		Short: "#tbd",
		// We are using our custom ErrSilent instead to support a more specific
		// error handling
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if workdir != "" {
				err := os.Chdir(workdir)
				if err != nil {
					err = errors.WithStack(err)
					log.Error(err, err.Error())
					return cmdutils.ErrSilent
				}
			}
			return nil
		},
	}

	rootCmd.PersistentFlags().BoolP("verbose", "v", false,
		"Show more verbose output, can be helpful for debugging problems")
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))

	rootCmd.PersistentFlags().StringVarP(&workdir, "directory", "C", "",
		"Change the directory before performing any operations")
	viper.BindPFlag("directory", rootCmd.PersistentFlags().Lookup("directory"))

	rootCmd.AddCommand(initCmd.New())
	rootCmd.AddCommand(createCmd.New())
	rootCmd.AddCommand(buildCmd.New())
	rootCmd.AddCommand(runCmd.New())

	return rootCmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd := New()
	if cmd, err := rootCmd.ExecuteC(); err != nil {

		// Errors that are not ErrSilent are not expected and we want to show their full stacktrace
		if !errors.Is(err, cmdutils.ErrSilent) {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "%+v\n", err)
		}

		// We only want to print the usage message if an ErrIncorrectUsage
		// was returned or it's an error produced by cobra which was
		// caused by incorrect usage
		var usageErr *cmdutils.IncorrectUsageError
		if errors.As(err, &usageErr) ||
			strings.HasPrefix(err.Error(), "required flag") ||
			strings.HasPrefix(err.Error(), "unknown command") {
			// Ensure that there is an extra newline between the error
			// and the usage message
			if !strings.HasSuffix(err.Error(), "\n") {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr())
			}
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), cmd.UsageString())
		}

		os.Exit(1)
	}
}
