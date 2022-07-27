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
	"code-intelligence.com/cifuzz/util/executil"
	"code-intelligence.com/cifuzz/util/fileutil"
)

type runOptions struct {
	BuildSystem    string        `mapstructure:"build-system"`
	BuildCommand   string        `mapstructure:"build-command"`
	SeedCorpusDirs []string      `mapstructure:"seed-corpus-dirs"`
	Dictionary     string        `mapstructure:"dict"`
	EngineArgs     []string      `mapstructure:"engine-args"`
	FuzzTestArgs   []string      `mapstructure:"fuzz-test-args"`
	Timeout        time.Duration `mapstructure:"timeout"`
	UseSandbox     bool          `mapstructure:"use-sandbox"`
	PrintJSON      bool          `mapstructure:"print-json"`

	ProjectDir string
	fuzzTest   string
}

func (opts *runOptions) validate() error {
	var err error

	// Check if the seed dirs exist and can be accessed and ensure that
	// the paths are absolute
	for i, d := range opts.SeedCorpusDirs {
		_, err := os.Stat(d)
		if err != nil {
			err = errors.WithStack(err)
			log.Error(err, err.Error())
			return cmdutils.ErrSilent
		}
		opts.SeedCorpusDirs[i], err = filepath.Abs(d)
		if err != nil {
			err = errors.WithStack(err)
			log.Error(err, err.Error())
			return cmdutils.ErrSilent
		}
	}

	if opts.Dictionary != "" {
		// Check if the dictionary exists and can be accessed
		_, err := os.Stat(opts.Dictionary)
		if err != nil {
			err = errors.WithStack(err)
			log.Error(err, err.Error())
			return cmdutils.ErrSilent
		}
	}

	if opts.BuildSystem == "" {
		opts.BuildSystem, err = config.DetermineBuildSystem(opts.ProjectDir)
		if err != nil {
			return err
		}
	} else {
		err = config.ValidateBuildSystem(opts.BuildSystem)
		if err != nil {
			return err
		}
	}

	return nil
}

type runCmd struct {
	*cobra.Command
	opts *runOptions

	config        *config.Config
	reportHandler *report_handler.ReportHandler
}

func New() *cobra.Command {
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
			err := config.ParseProjectConfig(opts)
			if err != nil {
				return err
			}

			opts.fuzzTest = args[0]
			return opts.validate()
		},
		RunE: func(c *cobra.Command, args []string) error {
			cmd := runCmd{Command: c, opts: opts}
			return cmd.run()
		},
	}

	cmd.Flags().String("build-command", "", "The command to build the fuzz test. Example: \"make clean && make my-fuzz-test\"")
	cmdutils.ViperMustBindPFlag("build-command", cmd.Flags().Lookup("build-command"))

	cmd.Flags().StringArrayP("seed-corpus", "s", nil, "Directory containing sample inputs for the code under test.\nSee https://llvm.org/docs/LibFuzzer.html#corpus and\nhttps://aflplus.plus/docs/fuzzing_in_depth/#a-collecting-inputs.")
	cmdutils.ViperMustBindPFlag("seed-corpus-dirs", cmd.Flags().Lookup("seed-corpus"))

	cmd.Flags().String("dict", "", "A file containing input language keywords or other interesting byte sequences.\nSee https://llvm.org/docs/LibFuzzer.html#dictionaries and\nhttps://github.com/AFLplusplus/AFLplusplus/blob/stable/dictionaries/README.md.")
	cmdutils.ViperMustBindPFlag("dict", cmd.Flags().Lookup("dict"))

	cmd.Flags().StringArray("engine-arg", nil, "Command-line argument to pass to the fuzzing engine.\nSee https://llvm.org/docs/LibFuzzer.html#options and\nhttps://www.mankier.com/8/afl-fuzz.")
	cmdutils.ViperMustBindPFlag("engine-args", cmd.Flags().Lookup("engine-arg"))

	cmd.Flags().StringArray("fuzz-test-arg", nil, "Command-line argument to pass to the fuzz test.")
	cmdutils.ViperMustBindPFlag("fuzz-test-args", cmd.Flags().Lookup("fuzz-test-arg"))

	cmd.Flags().Duration("timeout", 0, "Maximum time in seconds to run the fuzz test. The default is to run indefinitely.")
	cmdutils.ViperMustBindPFlag("timeout", cmd.Flags().Lookup("timeout"))

	cmd.Flags().Bool("use-sandbox", false, "By default, fuzz tests are executed in a sandbox to prevent accidental damage to the system.\nUse --use-sandbox=false to run the fuzz test unsandboxed.\nOnly supported on Linux.")
	viper.SetDefault("use-sandbox", runtime.GOOS == "linux")
	cmdutils.ViperMustBindPFlag("use-sandbox", cmd.Flags().Lookup("use-sandbox"))

	cmd.Flags().BoolVar(&opts.PrintJSON, "json", false, "Print output as JSON")
	cmdutils.ViperMustBindPFlag("print-json", cmd.Flags().Lookup("json"))

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
	c.reportHandler, err = report_handler.NewReportHandler(defaultSeedCorpusDir, c.opts.PrintJSON, viper.GetBool("verbose"))
	if err != nil {
		return err
	}

	err = c.runFuzzTest(fuzzTestExecutable)
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && c.opts.UseSandbox {
			return cmdutils.WrapCouldBeSandboxError(err)
		}
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

	if c.opts.BuildSystem == config.BuildSystemCMake {
		builder, err := cmake.NewBuilder(&cmake.BuilderOptions{
			ProjectDir: c.opts.ProjectDir,
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
	} else if c.opts.BuildSystem == config.BuildSystemOther {
		if runtime.GOOS == "windows" {
			return "", errors.New("CMake is the only supported build system on Windows")
		}
		return c.buildWithOtherBuildSystem()
	} else {
		return "", errors.Errorf("Unsupported build system \"%s\"", c.opts.BuildSystem)
	}
}

func (c *runCmd) buildWithOtherBuildSystem() (string, error) {
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

	// To build with other build systems, a build command must be provided.
	if c.opts.BuildCommand == "" {
		return "", cmdutils.WrapIncorrectUsageError(errors.Errorf("Flag \"build-command\" must be set to build" +
			" when using build system type \"other\""))
	}

	// Run the build command
	cmd := exec.Command("/bin/sh", "-c", c.opts.BuildCommand)
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

	generatedCorpusDir := c.generatedCorpusPath()
	err := os.MkdirAll(generatedCorpusDir, 0755)
	if err != nil {
		return errors.WithStack(err)
	}
	log.Infof("Storing generated corpus in %s", fileutil.PrettifyPath(generatedCorpusDir))

	runnerOpts := &libfuzzer.RunnerOptions{
		FuzzTarget:         fuzzTestExecutable,
		GeneratedCorpusDir: generatedCorpusDir,
		SeedCorpusDirs:     c.opts.SeedCorpusDirs,
		Dictionary:         c.opts.Dictionary,
		EngineArgs:         c.opts.EngineArgs,
		FuzzTestArgs:       c.opts.FuzzTestArgs,
		ReportHandler:      c.reportHandler,
		Timeout:            c.opts.Timeout,
		UseMinijail:        c.opts.UseSandbox,
		Verbose:            viper.GetBool("verbose"),
		KeepColor:          !c.opts.PrintJSON,
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

	var execErr *cmdutils.ExecError
	if errors.As(err, &execErr) {
		// It's expected that libFuzzer might fail due to user
		// configuration, so we print the error without the stack trace.
		log.Error(err)
		return cmdutils.ErrSilent
	}

	return err
}

func (c *runCmd) findFuzzTestExecutable(fuzzTest string) (string, error) {
	if exists, _ := fileutil.Exists(fuzzTest); exists {
		return executil.CallablePath(fuzzTest), nil
	}
	var executable string
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.WithStack(err)
		}
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
	return executil.CallablePath(executable), nil
}

func (c *runCmd) printFinalMetrics() error {
	numSeeds, err := countSeeds(append(c.opts.SeedCorpusDirs, c.generatedCorpusPath()))
	if err != nil {
		return err
	}

	return c.reportHandler.PrintFinalMetrics(numSeeds)
}

func (c *runCmd) generatedCorpusPath() string {
	// Store the generated corpus in a single persistent directory per
	// fuzz test in a hidden subdirectory.
	return filepath.Join(c.opts.ProjectDir, ".cifuzz-corpus", c.opts.fuzzTest)
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
