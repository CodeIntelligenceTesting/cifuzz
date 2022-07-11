package build

import (
	"io"
	"os"
	"runtime"

	"github.com/pkg/errors"

	"code-intelligence.com/cifuzz/util/envutil"
)

type BuilderOptions struct {
	ProjectDir string
	Engine     string
	Sanitizers []string
	Stdout     io.Writer
	Stderr     io.Writer
}

func (opts *BuilderOptions) Validate() error {
	// Check that the project dir is set
	if opts.ProjectDir == "" {
		return errors.New("ProjectDir is not set")
	}
	// Check that the project dir exists and can be accessed
	_, err := os.Stat(opts.ProjectDir)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

type Builder interface {
	Opts() *BuilderOptions
	BuildDir() string

	Build(fuzzTest string) error
	Configure() error
	FindFuzzTestExecutable(fuzzTest string) (string, error)
	FindFuzzTestSeedCorpus(fuzzTest string) (string, error)
	GetRuntimeDeps(fuzzTest string) ([]string, error)
}

func CommonBuildEnv() ([]string, error) {
	var err error
	env := os.Environ()

	// On Windows, our preferred compiler is MSVC, which can't easily be run
	// from an arbitrary terminal as it requires about a dozen environment
	// variables to be set correctly. Thus, we assume users to run cifuzz from
	// a developer command prompt anyway and thus don't need to set the
	// compiler explicitly.
	if runtime.GOOS != "windows" {
		// Set the C/C++ compiler to clang/clang++, which is needed to build a
		// binary with fuzzing instrumentation (gcc doesn't have
		// -fsanitize=fuzzer).
		env, err = envutil.Setenv(env, "CC", "clang")
		if err != nil {
			return nil, err
		}
		env, err = envutil.Setenv(env, "CXX", "clang++")
		if err != nil {
			return nil, err
		}
	}

	// We don't want to fail if ASan is set up incorrectly for tools
	// built and executed during the build or they contain leaks.
	env, err = envutil.Setenv(env, "ASAN_OPTIONS", "detect_leaks=0:verify_asan_link_order=0")
	if err != nil {
		return nil, err
	}

	return env, nil
}
