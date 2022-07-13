package root

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	bundleCmd "code-intelligence.com/cifuzz/internal/cmd/bundle"
	createCmd "code-intelligence.com/cifuzz/internal/cmd/create"
	initCmd "code-intelligence.com/cifuzz/internal/cmd/init"
	reloadCmd "code-intelligence.com/cifuzz/internal/cmd/reload"
	runCmd "code-intelligence.com/cifuzz/internal/cmd/run"
	"code-intelligence.com/cifuzz/internal/config"
	"code-intelligence.com/cifuzz/pkg/cmdutils"
	"code-intelligence.com/cifuzz/pkg/log"
)

func New() (*cobra.Command, error) {
	cmdConfig := config.NewConfig()

	rootCmd := &cobra.Command{
		Use:   "cifuzz",
		Short: "#tbd",
		// We are using our custom ErrSilent instead to support a more specific
		// error handling
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			err := cmdutils.Chdir()
			if err != nil {
				log.Error(err, err.Error())
				return cmdutils.ErrSilent
			}

			if !cmdutils.NeedsConfig(cmd) {
				return nil
			}

			projectDir, err := config.FindProjectDir()
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

			projectConfig, err := config.ReadProjectConfig(projectDir)
			if err != nil {
				return err
			}
			cmdConfig.ProjectDir = projectDir
			cmdConfig.ProjectConfig = projectConfig

			return nil
		},
	}

	rootCmd.PersistentFlags().BoolP("verbose", "v", false,
		"Show more verbose output, can be helpful for debugging problems")
	if err := viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose")); err != nil {
		return nil, errors.WithStack(err)
	}

	rootCmd.PersistentFlags().StringP("directory", "C", "",
		"Change the directory before performing any operations")
	if err := viper.BindPFlag("directory", rootCmd.PersistentFlags().Lookup("directory")); err != nil {
		return nil, errors.WithStack(err)
	}

	rootCmd.SetFlagErrorFunc(rootFlagErrorFunc)

	rootCmd.AddCommand(initCmd.New())
	rootCmd.AddCommand(createCmd.New(cmdConfig))
	rootCmd.AddCommand(runCmd.New())
	rootCmd.AddCommand(reloadCmd.New())
	rootCmd.AddCommand(bundleCmd.New(cmdConfig))

	return rootCmd, nil
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd, err := New()
	if err != nil {
		fmt.Printf("error while creating root command: %+v", err)
		os.Exit(1)
	}

	if cmd, err := rootCmd.ExecuteC(); err != nil {

		// Errors that are not ErrSilent are not expected and we want to show their full stacktrace
		var silentErr *cmdutils.SilentError
		if !errors.As(err, &silentErr) {
			_, _ = fmt.Fprint(cmd.ErrOrStderr(), pterm.Style{pterm.Bold, pterm.FgRed}.Sprintf("%+v\n", err))
		}

		// We only want to print the usage message if an ErrIncorrectUsage
		// was returned or it's an error produced by cobra which was
		// caused by incorrect usage
		var usageErr *cmdutils.IncorrectUsageError
		if errors.As(err, &usageErr) ||
			strings.HasPrefix(err.Error(), "unknown command") ||
			regexp.MustCompile(`(accepts|requires).*arg\(s\)`).MatchString(err.Error()) {
			// Ensure that there is an extra newline between the error
			// and the usage message
			if !strings.HasSuffix(err.Error(), "\n") {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr())
			}
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), cmd.UsageString())
		}

		var signalErr *cmdutils.SignalError
		if errors.As(err, &signalErr) {
			os.Exit(128 + int(signalErr.Signal))
		}

		os.Exit(1)
	}
}

func rootFlagErrorFunc(cmd *cobra.Command, err error) error {
	if err == pflag.ErrHelp {
		return err
	}
	return cmdutils.WrapIncorrectUsageError(err)
}
