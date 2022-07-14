package run

import (
	"context"
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

	"code-intelligence.com/cifuzz/internal/build"
	"code-intelligence.com/cifuzz/internal/build/cmake"
	"code-intelligence.com/cifuzz/internal/cmd/run/report_handler"
	"code-intelligence.com/cifuzz/internal/completion"
	"code-intelligence.com/cifuzz/internal/config"
	"code-intelligence.com/cifuzz/pkg/cmdutils"
	"code-intelligence.com/cifuzz/pkg/log"
	"code-intelligence.com/cifuzz/pkg/runfiles"
	"code-intelligence.com/cifuzz/pkg/runner/libfuzzer"
	"code-intelligence.com/cifuzz/util/envutil"
	"code-intelligence.com/cifuzz/util/fileutil"
)

type runOptions struct {
	buildCommand   string
	fuzzTest       string
	seedCorpusDirs []string
	dictionary     string
	engineArgs     []string
	fuzzTargetArgs []string
	timeout        time.Duration
	useSandbox     bool
	printJSON      bool
}

func (opts *runOptions) validate() error {
	// Check if the seed dirs exist and can be accessed and ensure that
	// the paths are absolute
	for i, d := range opts.seedCorpusDirs {
		_, err := os.Stat(d)
		if err != nil {
			err = errors.WithStack(err)
			log.Error(err, err.Error())
			return cmdutils.ErrSilent
		}
		opts.seedCorpusDirs[i], err = filepath.Abs(d)
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
		Long:              "",
		ValidArgsFunction: completion.ValidFuzzTests,
		Args:              cobra.ExactArgs(1),
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
	cmd.Flags().StringArrayVarP(&opts.seedCorpusDirs, "seed-corpus", "s", nil, "Directory containing sample inputs for the code under test.\nSee https://llvm.org/docs/LibFuzzer.html#corpus and\nhttps://aflplus.plus/docs/fuzzing_in_depth/#a-collecting-inputs.")
	cmd.Flags().StringVar(&opts.dictionary, "dict", "", "A file containing input language keywords or other interesting byte sequences.\nSee https://llvm.org/docs/LibFuzzer.html#dictionaries and\nhttps://github.com/AFLplusplus/AFLplusplus/blob/stable/dictionaries/README.md.")
	cmd.Flags().StringArrayVar(&opts.engineArgs, "engine-arg", nil, "Command-line argument to pass to the fuzzing engine.\nSee https://llvm.org/docs/LibFuzzer.html#options and\nhttps://www.mankier.com/8/afl-fuzz.")
	cmd.Flags().StringArrayVar(&opts.fuzzTargetArgs, "fuzz-test-arg", nil, "Command-line argument to pass to the fuzz test.")
	cmd.Flags().DurationVar(&opts.timeout, "timeout", 0, "Maximum time in seconds to run the fuzz test. The default is to run indefinitely.")
	useMinijailDefault := runtime.GOOS == "linux"
	cmd.Flags().BoolVar(&opts.useSandbox, "use-sandbox", useMinijailDefault, "By default, fuzz tests are executed in a sandbox to prevent accidental damage to the system.\nUse --sandbox=false to run the fuzz test unsandboxed.\nOnly supported on Linux.")
	cmd.Flags().BoolVar(&opts.printJSON, "json", false, "Print output as JSON")

	return cmd
}

func (c *runCmd) run() error {
	var err error

	fuzzTestExecutable, err := c.buildFuzzTest()
	if err != nil {
		return err
	}

	// Initialize the report handler. Only do this right before we start
	// the fuzz test, because this is storing a timestamp which is used
	// to figure out how long the fuzzing run is running.
	defaultSeedCorpusDir := c.opts.fuzzTest + "_seed_corpus"
	c.reportHandler, err = report_handler.NewReportHandler(defaultSeedCorpusDir, c.opts.printJSON, viper.GetBool("verbose"))
	if err != nil {
		return err
	}

	err = c.runFuzzTest(fuzzTestExecutable)
	if err != nil {
		return err
	}

	err = c.printFinalMetrics()
	if err != nil {
		return err
	}

	return nil
}

func (c *runCmd) buildFuzzTest() (string, error) {
	// TODO: Do not hardcode these values.
	sanitizers := []string{"address"}
	// UBSan is not supported by MSVC
	// TODO: Not needed anymore when sanitizers are configurable,
	//       then we do want to fail if the user explicitly asked for
	//       UBSan.
	if runtime.GOOS != "windows" {
		sanitizers = append(sanitizers, "undefined")
	}

	if c.config.BuildSystem == config.BuildSystemCMake {
		builder, err := cmake.NewBuilder(&cmake.BuilderOptions{
			ProjectDir: c.config.ProjectDir,
			// TODO: Do not hardcode this values.
			Engine:     "libfuzzer",
			Sanitizers: sanitizers,
			Stdout:     c.OutOrStdout(),
			Stderr:     c.ErrOrStderr(),
		})
		if err != nil {
			return "", err
		}
		err = builder.Configure()
		if err != nil {
			return "", err
		}
		err = builder.Build([]string{c.opts.fuzzTest})
		if err != nil {
			return "", err
		}
		return builder.FindFuzzTestExecutable(c.opts.fuzzTest)
	} else if c.config.BuildSystem == config.BuildSystemUnknown {
		return c.buildWithUnknownBuildSystem()
	} else {
		return "", errors.Errorf("Unsupported build system \"%s\"", c.config.BuildSystem)
	}
}

func (c *runCmd) buildWithUnknownBuildSystem() (string, error) {
	// Prepare the environment
	env, err := build.CommonBuildEnv()
	if err != nil {
		return "", err
	}
	// Set CFLAGS, CXXFLAGS, LDFLAGS, and FUZZ_TEST_LDFLAGS which must
	// be passed to the build commands by the build system.
	env, err = setBuildFlagsEnvVars(env)
	if err != nil {
		return "", err
	}

	// To build with an unknown build system, a build command must be
	// provided
	if c.opts.buildCommand == "" {
		return "", cmdutils.WrapIncorrectUsageError(errors.Errorf("Flag \"build-command\" must be set to build" +
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
		return "", errors.WithStack(err)
	}
	return c.findFuzzTestExecutable(c.opts.fuzzTest)
}

func (c *runCmd) runFuzzTest(fuzzTestExecutable string) error {
	log.Infof("Running %s", pterm.Style{pterm.Reset, pterm.FgLightBlue}.Sprintf(c.opts.fuzzTest))
	log.Debugf("Executable: %s", fuzzTestExecutable)

	// Store the generated corpus in a single persistent directory per
	// fuzz test in a hidden subdirectory.
	generatedCorpusDir := filepath.Join(c.config.ProjectDir, ".cifuzz-corpus", c.opts.fuzzTest)
	err := os.MkdirAll(generatedCorpusDir, 0755)
	if err != nil {
		return errors.WithStack(err)
	}
	log.Infof("Storing generated corpus in %s", fileutil.PrettifyPath(generatedCorpusDir))

	runnerOpts := &libfuzzer.RunnerOptions{
		FuzzTarget:         fuzzTestExecutable,
		GeneratedCorpusDir: generatedCorpusDir,
		SeedCorpusDirs:     c.opts.seedCorpusDirs,
		Dictionary:         c.opts.dictionary,
		EngineArgs:         c.opts.engineArgs,
		FuzzTargetArgs:     c.opts.fuzzTargetArgs,
		ReportHandler:      c.reportHandler,
		Timeout:            c.opts.timeout,
		UseMinijail:        c.opts.useSandbox,
		Verbose:            viper.GetBool("verbose"),
		KeepColor:          !c.opts.printJSON,
	}
	runner := libfuzzer.NewRunner(runnerOpts)

	// Handle cleanup (terminating the fuzzer process) when receiving
	// termination signals
	signalHandlerCtx, cancelSignalHandler := context.WithCancel(context.Background())
	routines, routinesCtx := errgroup.WithContext(signalHandlerCtx)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	var signalErr error
	routines.Go(func() error {
		select {
		case <-signalHandlerCtx.Done():
			return nil
		case s := <-sigs:
			log.Warnf("Received %s", s.String())
			signalErr = cmdutils.NewSignalError(s.(syscall.Signal))
			runner.Cleanup()
			return signalErr
		}
	})

	// Run the fuzzer
	routines.Go(func() error {
		defer cancelSignalHandler()
		return runner.Run(routinesCtx)
	})

	err = routines.Wait()
	// We use a separate variable to pass signal errors, because when
	// a signal was received, the first goroutine terminates the second
	// one, resulting in a race of which returns an error first. In that
	// case, we always want to print the signal error, not the
	// "Unexpected exit code" error from the runner.
	if signalErr != nil {
		log.Error(signalErr, signalErr.Error())
		return cmdutils.WrapSilentError(signalErr)
	}
	return err
}

func (c *runCmd) findFuzzTestExecutable(fuzzTest string) (string, error) {
	if exists, _ := fileutil.Exists(fuzzTest); exists {
		return fuzzTest, nil
	}
	var executable string
	err := filepath.Walk(c.buildDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if runtime.GOOS == "windows" {
			if info.Name() == fuzzTest+".exe" {
				executable = path
			}
		} else {
			// As a heuristic, verify that the executable candidate has some
			// executable bit set - it may not be sufficient to actually execute
			// it as the current user.
			if info.Name() == fuzzTest && (info.Mode()&0111 != 0) {
				executable = path
			}
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
	numSeeds, err := countSeeds(c.opts.seedCorpusDirs)
	if err != nil {
		return err
	}

	return c.reportHandler.PrintFinalMetrics(numSeeds)
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
	cifuzzIncludePath, err := runfiles.Finder.CIFuzzIncludePath()
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

func countSeeds(seedCorpusDirs []string) (numSeeds uint, err error) {
	for _, dir := range seedCorpusDirs {
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
