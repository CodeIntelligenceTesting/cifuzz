package completion

import (
	"path/filepath"

	"github.com/mattn/go-zglob"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"code-intelligence.com/cifuzz/internal/config"
	"code-intelligence.com/cifuzz/pkg/cmdutils"
	"code-intelligence.com/cifuzz/pkg/log"
)

// ValidFuzzTests can be used as a cobra ValidArgsFunction that completes fuzz test names.
func ValidFuzzTests(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Change the directory if the `--directory` flag was set
	err := cmdutils.Chdir()
	if err != nil {
		log.Error(err, err.Error())
		return nil, cobra.ShellCompDirectiveError
	}

	// Read the project config to figure out the build system
	projectDir, err := config.FindProjectDir()
	if err != nil {
		log.Error(err, err.Error())
		return nil, cobra.ShellCompDirectiveError
	}
	conf, err := config.ReadProjectConfig(projectDir)
	if err != nil {
		log.Error(err, err.Error())
		return nil, cobra.ShellCompDirectiveError
	}

	if conf.BuildSystem == config.BuildSystemCMake {
		return validCMakeFuzzTests(cmd, args, toComplete)
	} else if conf.BuildSystem == config.BuildSystemOther {
		// For other build systems, the <fuzz test> argument must be
		// the path to the fuzz test executable, so we use file
		// completion here (which is only useful if the executable has
		// been built before, but that's still better than no completion
		// support)
		return nil, cobra.ShellCompDirectiveDefault
	} else {
		err := errors.Errorf("Unsupported build system \"%s\"", conf.BuildSystem)
		log.Error(err, err.Error())
		return nil, cobra.ShellCompDirectiveError
	}
}

func validCMakeFuzzTests(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	matches, err := zglob.Glob(".cifuzz-build/**/.cifuzz/fuzz_tests/*")
	if err != nil {
		log.Error(err, err.Error())
		return nil, cobra.ShellCompDirectiveError
	}
	var res []string
	for _, match := range matches {
		res = append(res, filepath.Base(match))
	}
	return res, cobra.ShellCompDirectiveNoFileComp
}
