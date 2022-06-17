package run

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"

	"code-intelligence.com/cifuzz/internal/cmd/run/report_handler"
	"code-intelligence.com/cifuzz/internal/config"
	"code-intelligence.com/cifuzz/pkg/cmdutils"
	"code-intelligence.com/cifuzz/pkg/log"
	"code-intelligence.com/cifuzz/pkg/runfiles"
	"code-intelligence.com/cifuzz/pkg/runner/libfuzzer"
	"code-intelligence.com/cifuzz/util/envutil"
	"code-intelligence.com/cifuzz/util/fileutil"
)

// The CMake configuration (also called "build type") to use for fuzzing runs.
// See enable_fuzz_testing in tools/cmake/CIFuzz/share/CIFuzz/CIFuzzFunctions.cmake for the rationale for using this
// build type.
const cmakeBuildConfiguration = "RelWithDebInfo"

type runOptions struct {
	buildCommand   string
	fuzzTest       string
	seedsDirs      []string
	dictionary     string
	engineArgs     []string
	fuzzTargetArgs []string
	timeout        time.Duration
	useSandbox     bool
	printJSON      bool
}

func (opts *runOptions) validate() error {
	// Check if the seed dirs exist and can be accessed
	for _, d := range opts.seedsDirs {
		_, err := os.Stat(d)
		if err != nil {
			err = errors.WithStack(err)
			log.Error(err, err.Error())
			return cmdutils.ErrSilent
		}
	}

	if opts.dictionary != "" {
		// Check if the dictionary exists and can be accessed
		_, err := os.Stat(opts.dictionary)
		if err != nil {
			err = errors.WithStack(err)
			log.Error(err, err.Error())
			return cmdutils.ErrSilent
		}
	}

	return nil
}

type runCmd struct {
	*cobra.Command
	opts *runOptions

	config        *config.Config
	buildDir      string
	reportHandler *report_handler.ReportHandler
}

func New(config *config.Config) *cobra.Command {
	opts := &runOptions{}

	cmd := &cobra.Command{
		Use:   "run [flags] <fuzz test>",
		Short: "Build and run a fuzz test",
		// TODO: Write long description (easier once we support more
		//       than just the fallback mode). In particular, explain how a
		//       "fuzz test" is identified on the CLI.
		Long: "",
		Args: cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.fuzzTest = args[0]
			return opts.validate()
		},
		RunE: func(c *cobra.Command, args []string) error {
			cmd := runCmd{
				Command: c,
				opts:    opts,
				config:  config,
			}
			return cmd.run()
		},
	}

	cmd.Flags().StringVar(&opts.buildCommand, "build-command", "", "The command to build the fuzz test. Example: \"make clean && make my-fuzz-test\"")
	cmd.Flags().StringArrayVarP(&opts.seedsDirs, "seeds-dir", "s", nil, "Directory containing sample inputs for the code under test.\nSee https://llvm.org/docs/LibFuzzer.html#corpus and\nhttps://aflplus.plus/docs/fuzzing_in_depth/#a-collecting-inputs.")
	cmd.Flags().StringVar(&opts.dictionary, "dict", "", "A file containing input language keywords or other interesting byte sequences.\nSee https://llvm.org/docs/LibFuzzer.html#dictionaries and\nhttps://github.com/AFLplusplus/AFLplusplus/blob/stable/dictionaries/README.md.")
	cmd.Flags().StringArrayVar(&opts.engineArgs, "engine-arg", nil, "Command-line argument to pass to the fuzzing engine.\nSee https://llvm.org/docs/LibFuzzer.html#options and\nhttps://www.mankier.com/8/afl-fuzz.")
	cmd.Flags().StringArrayVar(&opts.fuzzTargetArgs, "fuzz-target-arg", nil, "Command-line argument to pass to the fuzz target.")
	cmd.Flags().DurationVar(&opts.timeout, "timeout", 0, "Maximum time in seconds to run the fuzz test. The default is to run indefinitely.")
	useMinijailDefault := runtime.GOOS == "linux"
	cmd.Flags().BoolVar(&opts.useSandbox, "sandbox", useMinijailDefault, "By default, fuzz tests are executed in a sandbox to prevent accidental damage to the system.\nUse --sandbox=false to run the fuzz test unsandboxed.\nOnly supported on Linux.")
	cmd.Flags().BoolVar(&opts.printJSON, "json", false, "Print output as JSON")

	return cmd
}

func (c *runCmd) run() error {
	var err error

	err = c.buildFuzzTest()
	if err != nil {
		return err
	}

	// Initialize the report handler. Only do this right before we start
	// the fuzz test, because this is storing a timestamp which is used
	// to figure out how long the fuzzing run is running.
	c.reportHandler, err = report_handler.NewReportHandler(c.opts.printJSON, viper.GetBool("verbose"))
	if err != nil {
		return err
	}

	err = c.runFuzzTest()
	if err != nil {
		return err
	}

	err = c.printFinalMetrics()
	if err != nil {
		return err
	}

	return nil
}

func (c *runCmd) buildFuzzTest() error {
	conf, err := config.ReadProjectConfig(c.config.ProjectDir)
	if err != nil {
		return err
	}

	if conf.BuildSystem == config.BuildSystemCMake {
		return c.buildWithCMake()
	} else if conf.BuildSystem == config.BuildSystemUnknown {
		return c.buildWithUnknownBuildSystem()
	} else {
		return errors.Errorf("Unsupported build system \"%s\"", conf.BuildSystem)
	}
}

func (c *runCmd) buildWithCMake() error {
	// TODO: Make these configurable
	engine := "libfuzzer"
	sanitizers := []string{"address", "undefined"}

	// Prepare the environment
	env, err := commonBuildEnv()
	if err != nil {
		return err
	}

	// Ensure that the build directory exists.
	// Note: Invoking CMake on the same build directory with different cache
	// variables is a no-op. For this reason, we have to encode all choices made
	// for the cache variables below in the path to the build directory.
	// Currently, this includes the fuzzing engine and the choice of sanitizers.
	c.buildDir = filepath.Join(c.config.ProjectDir, ".cifuzz-build", engine, strings.Join(sanitizers, "+"))
	err = os.MkdirAll(c.buildDir, 0755)
	if err != nil {
		return err
	}

	cacheVariables := map[string]string{
		"CMAKE_BUILD_TYPE":    cmakeBuildConfiguration,
		"CIFUZZ_ENGINE":       engine,
		"CIFUZZ_SANITIZERS":   strings.Join(sanitizers, ";"),
		"CIFUZZ_TESTING:BOOL": "ON",
	}
	var cacheArgs []string
	for key, value := range cacheVariables {
		cacheArgs = append(cacheArgs, "-D", fmt.Sprintf("%s=%s", key, value))
	}

	// Call cmake to "Generate a project buildsystem" (that's the
	// phrasing used by the CMake man page).
	// Note: This is usually a no-op after the directory has been created once,
	// even if cache variables change. However, if a previous invocation of this
	// command failed during CMake generation and the command is run again, the
	// build step would only result in a very unhelpful error message about
	// missing Makefiles. By reinvoking CMake's configuration explicitly here,
	// we either get a helpful error message or the build step will succeed if
	// the user fixed the issue in the meantime.
	cmd := exec.Command("cmake", append(cacheArgs, c.config.ProjectDir)...)
	// Redirect the build command's stdout to stderr to only have
	// reports printed to stdout
	cmd.Stdout = c.ErrOrStderr()
	cmd.Stderr = c.ErrOrStderr()
	cmd.Env = env
	cmd.Dir = c.buildDir
	log.Debugf("Working directory: %s", cmd.Dir)
	log.Debugf("Command: %s", cmd.String())
	err = cmd.Run()
	if err != nil {
		return err
	}

	// Build the project with CMake
	cmd = exec.Command(
		"cmake",
		"--build", c.buildDir,
		"--config", cmakeBuildConfiguration,
		"--target", c.opts.fuzzTest,
	)
	// Redirect the build command's stdout to stderr to only have
	// reports printed to stdout
	cmd.Stdout = c.ErrOrStderr()
	cmd.Stderr = c.ErrOrStderr()
	cmd.Env = env
	log.Debugf("Command: %s", cmd.String())
	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func (c *runCmd) buildWithUnknownBuildSystem() error {
	// Prepare the environment
	env, err := commonBuildEnv()
	if err != nil {
		return err
	}
	// Set CFLAGS, CXXFLAGS, LDFLAGS, and FUZZ_TEST_LDFLAGS which must
	// be passed to the build commands by the build system.
	env, err = setBuildFlagsEnvVars(env)
	if err != nil {
		return err
	}

	// To build with an unknown build system, a build command must be
	// provided
	if c.opts.buildCommand == "" {
		return cmdutils.WrapIncorrectUsageError(errors.Errorf("Flag \"build-command\" must be set to build" +
			" with an unknown build system"))
	}

	// Run the build command
	cmd := exec.Command("/bin/sh", "-c", c.opts.buildCommand)
	// Redirect the build command's stdout to stderr to only have
	// reports printed to stdout
	cmd.Stdout = c.ErrOrStderr()
	cmd.Stderr = c.ErrOrStderr()
	cmd.Env = env
	log.Debugf("Command: %s", cmd.String())
	err = cmd.Run()
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (c *runCmd) runFuzzTest() error {
	log.Infof("Running %s", pterm.Style{pterm.Reset, pterm.FgLightBlue}.Sprintf(c.opts.fuzzTest))
	fuzzTestExecutable, err := c.findFuzzTestExecutable(c.opts.fuzzTest)
	if err != nil {
		return err
	}
	log.Debugf("Executable: %s", fuzzTestExecutable)

	if len(c.opts.seedsDirs) == 0 {
		// If no seeds directory is specified, use a single persistent corpus
		// directory per fuzz test in a hidden subdirectory. In most cases,
		// corpora are not checked into source control, so a hidden directory
		// is an appropriate default that can always be overridden via the
		// --seeds-dir flag.
		defaultCorpusDir := filepath.Join(c.config.ProjectDir, ".cifuzz-corpus", c.opts.fuzzTest)
		err := os.MkdirAll(defaultCorpusDir, 0755)
		if err != nil {
			return errors.WithStack(err)
		}
		log.Infof("Storing corpus in %s", fileutil.PrettifyPath(defaultCorpusDir))
		c.opts.seedsDirs = []string{defaultCorpusDir}
	}

	runnerOpts := &libfuzzer.RunnerOptions{
		FuzzTarget:          fuzzTestExecutable,
		SeedsDir:            c.opts.seedsDirs[0],
		AdditionalSeedsDirs: c.opts.seedsDirs[1:],
		Dictionary:          c.opts.dictionary,
		EngineArgs:          c.opts.engineArgs,
		FuzzTargetArgs:      c.opts.fuzzTargetArgs,
		ReportHandler:       c.reportHandler,
		Timeout:             c.opts.timeout,
		UseMinijail:         c.opts.useSandbox,
		Verbose:             viper.GetBool("verbose"),
		KeepColor:           !c.opts.printJSON,
	}
	runner := libfuzzer.NewRunner(runnerOpts)

	// Handle cleanup (terminating the fuzzer process) when receiving
	// termination signals
	signalHandlerCtx, cancelSignalHandler := context.WithCancel(context.Background())
	routines, routinesCtx := errgroup.WithContext(signalHandlerCtx)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	routines.Go(func() error {
		select {
		case <-signalHandlerCtx.Done():
			return nil
		case s := <-sigs:
			log.Warnf("Received %s", s.String())
			runner.Cleanup()
			err := cmdutils.NewSignalError(s.(syscall.Signal))
			log.Error(err, err.Error())
			return cmdutils.WrapSilentError(err)
		}
	})

	// Run the fuzzer
	routines.Go(func() error {
		defer cancelSignalHandler()
		return runner.Run(routinesCtx)
	})

	return routines.Wait()
}

func (c *runCmd) findFuzzTestExecutable(fuzzTest string) (string, error) {
	if exists, _ := fileutil.Exists(fuzzTest); exists {
		return fuzzTest, nil
	}
	var executable string
	err := filepath.Walk(c.buildDir, func(path string, info os.FileInfo, err error) error {
		if info.Name() == fuzzTest {
			executable = path
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if executable == "" {
		return "", errors.Errorf("Could not find executable for fuzz test %s", fuzzTest)
	}
	return executable, nil
}

func (c *runCmd) printFinalMetrics() error {
	numSeeds, err := countSeeds(c.opts.seedsDirs)
	if err != nil {
		return err
	}

	return c.reportHandler.PrintFinalMetrics(numSeeds)
}

func commonBuildEnv() ([]string, error) {
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

func setBuildFlagsEnvVars(env []string) ([]string, error) {
	// Set CFLAGS and CXXFLAGS. Note that these flags must not contain
	// spaces, because the environment variables are space separated.
	//
	// Note: Keep in sync with tools/cmake/CIFuzz/share/CIFuzz/CIFuzzFunctions.cmake
	cflags := []string{
		// ----- Common flags -----
		// Keep debug symbols
		"-g",
		// Do optimizations which don't harm debugging
		"-Og",
		// To get good stack frames for better debugging
		"-fno-omit-frame-pointer",
		// Conventional macro to conditionally compile out fuzzer road blocks
		// See https://llvm.org/docs/LibFuzzer.html#fuzzer-friendly-build-mode
		"-DFUZZING_BUILD_MODE_UNSAFE_FOR_PRODUCTION",

		// ----- Flags used to build with libFuzzer -----
		// Compile with edge coverage and compare instrumentation. We
		// use fuzzer-no-link here instead of -fsanitize=fuzzer because
		// CFLAGS are often also passed to the linker, which would cause
		// errors if the build includes tools which have a main function.
		"-fsanitize=fuzzer-no-link",

		// ----- Flags used to build with ASan -----
		// Build with instrumentation for ASan and UBSan and link in
		// their runtime
		"-fsanitize=address,undefined",
		// To support recovering from ASan findings
		"-fsanitize-recover=address",
		// Use additional error detectors for use-after-scope bugs
		// TODO: Evaluate the slow down caused by this flag
		// TODO: Check if there are other additional error detectors
		//       which we want to use
		"-fsanitize-address-use-after-scope",
	}
	env, err := envutil.Setenv(env, "CFLAGS", strings.Join(cflags, " "))
	if err != nil {
		return nil, err
	}
	env, err = envutil.Setenv(env, "CXXFLAGS", strings.Join(cflags, " "))
	if err != nil {
		return nil, err
	}

	ldflags := []string{
		// ----- Flags used to build with ASan -----
		// Link ASan and UBSan runtime
		"-fsanitize=address,undefined",
		// To avoid issues with clang (not clang++) and UBSan, see
		// https://github.com/bazelbuild/bazel/issues/11122#issuecomment-896613570
		"-fsanitize-link-c++-runtime",
	}
	env, err = envutil.Setenv(env, "LDFLAGS", strings.Join(ldflags, " "))
	if err != nil {
		return nil, err
	}

	// Users should pass the environment variable FUZZ_TEST_CFLAGS to the
	// compiler command building the fuzz test.
	cifuzzIncludePath, err := runfiles.Finder.CifuzzIncludePath()
	if err != nil {
		return nil, err
	}
	env, err = envutil.Setenv(env, "FUZZ_TEST_CFLAGS", "-I"+cifuzzIncludePath)
	if err != nil {
		return nil, err
	}

	// Users should pass the environment variable FUZZ_TEST_LDFLAGS to
	// the linker command building the fuzz test. For libfuzzer, we set
	// it to "-fsanitize=fuzzer" to build a libfuzzer binary.
	env, err = envutil.Setenv(env, "FUZZ_TEST_LDFLAGS", "-fsanitize=fuzzer")
	if err != nil {
		return nil, err
	}

	return env, nil
}

func countSeeds(seedDirs []string) (numSeeds uint, err error) {
	for _, dir := range seedDirs {
		var seedsInDir uint
		err = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return err
			}
			// Don't count empty files, same as libFuzzer
			if info.Size() != 0 {
				seedsInDir += 1
			}
			return nil
		})
		if err != nil {
			return 0, errors.WithStack(err)
		}
		numSeeds += seedsInDir
	}
	return numSeeds, nil
}
