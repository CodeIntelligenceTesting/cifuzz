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
	"github.com/spf13/cobra"

	"code-intelligence.com/cifuzz/internal/config"
	"code-intelligence.com/cifuzz/pkg/cmdutils"
	"code-intelligence.com/cifuzz/pkg/log"
	"code-intelligence.com/cifuzz/pkg/report"
	"code-intelligence.com/cifuzz/pkg/runner/libfuzzer"
	"code-intelligence.com/cifuzz/util/envutil"
	"code-intelligence.com/cifuzz/util/fileutil"
	"code-intelligence.com/cifuzz/util/stringutil"
)

type runOptions struct {
	buildCommand   string
	fuzzTest       string
	seedsDirs      []string
	dictionary     string
	engineArgs     []string
	fuzzTargetArgs []string
	timeout        time.Duration
	useSandbox     bool
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

	projectDir string
	buildDir   string
}

func New() *cobra.Command {
	opts := &runOptions{}

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Build and run a fuzz test",
		// TODO: Write long description (easier once we support more
		//       than just the fallback mode).
		Long: "",
		Args: cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return opts.validate()
		},
		RunE: func(c *cobra.Command, args []string) error {
			cmd := runCmd{Command: c, opts: opts}
			return cmd.run()
		},
	}

	cmd.Flags().StringVar(&opts.buildCommand, "build-command", "", "The command to build the fuzz test. Example: \"make clean && make my-fuzz-test\"")
	cmd.Flags().StringVarP(&opts.fuzzTest, "fuzz-test", "f", "", "Path where the fuzz test executable can be found after running the build command.")
	cmd.Flags().StringArrayVarP(&opts.seedsDirs, "seeds-dir", "s", nil, "Directory containing sample inputs for the code under test.\nSee https://llvm.org/docs/LibFuzzer.html#corpus and\nhttps://aflplus.plus/docs/fuzzing_in_depth/#a-collecting-inputs.")
	cmd.Flags().StringVar(&opts.dictionary, "dict", "", "A file containing input language keywords or other interesting byte sequences.\nSee https://llvm.org/docs/LibFuzzer.html#dictionaries and\nhttps://github.com/AFLplusplus/AFLplusplus/blob/stable/dictionaries/README.md.")
	cmd.Flags().StringArrayVar(&opts.engineArgs, "engine-arg", nil, "Command-line argument to pass to the fuzzing engine.\nSee https://llvm.org/docs/LibFuzzer.html#options and\nhttps://www.mankier.com/8/afl-fuzz.")
	cmd.Flags().StringArrayVar(&opts.fuzzTargetArgs, "fuzz-target-arg", nil, "Command-line argument to pass to the fuzz target.")
	cmd.Flags().DurationVar(&opts.timeout, "timeout", 0, "Maximum time in seconds to run the fuzz test. The default is to run indefinitely.")
	useMinijailDefault := strings.HasPrefix(runtime.GOOS, "linux")
	cmd.Flags().BoolVar(&opts.useSandbox, "sandbox", useMinijailDefault, "By default, fuzz tests are executed in a sandbox to prevent accidental damage to the system.\nUse --sandbox=false to run the fuzz test unsandboxed.\nOnly supported on Linux.")
	cmdutils.MarkFlagsRequired(cmd, "seeds-dir", "fuzz-test")

	return cmd
}

func (c *runCmd) run() error {
	var err error

	c.projectDir, err = config.FindProjectDir()
	if err != nil {
		return err
	}

	err = c.buildFuzzTest()
	if err != nil {
		return err
	}

	err = c.runFuzzTest()
	if err != nil {
		return err
	}

	return nil
}

func (c *runCmd) buildFuzzTest() error {
	conf, err := config.ReadProjectConfig(c.projectDir)
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

	// Create the build directory if it doesn't exist
	c.buildDir = filepath.Join(c.projectDir, ".cifuzz-build", engine, strings.Join(sanitizers, "+"))
	exists, err := fileutil.Exists(c.buildDir)
	if err != nil {
		return err
	}
	if !exists {
		err = os.MkdirAll(c.buildDir, 0755)
		if err != nil {
			return err
		}

		cacheVariables := map[string]string{
			"CIFUZZ_SANITIZERS": strings.Join(sanitizers, ";"),
			"CIFUZZ_ENGINE":     engine,
		}
		var cacheArgs []string
		for key, value := range cacheVariables {
			cacheArgs = append(cacheArgs, "-D", fmt.Sprintf("%s=%s", key, value))
		}

		// Call cmake to "Generate a project buildsystem" (that's the
		// phrasing used by the CMake man page).
		cmd := exec.Command("cmake", append(cacheArgs, c.projectDir)...)
		cmd.Stdout = c.OutOrStdout()
		cmd.Stderr = c.ErrOrStderr()
		cmd.Env = env
		cmd.Dir = c.buildDir
		log.Debugf("Working directory: %s", cmd.Dir)
		log.Debugf("Command: %s", cmd.String())
		err = cmd.Run()
		if err != nil {
			return err
		}
	}

	// Build the project with CMake
	cmd := exec.Command("cmake", "--build", c.buildDir, "--target", c.opts.fuzzTest)
	cmd.Stdout = c.OutOrStdout()
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
	cmd.Stdout = c.OutOrStdout()
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
	fuzzTestExecutable, err := c.findFuzzTestExecutable(c.opts.fuzzTest)
	if err != nil {
		return err
	}
	log.Debugf("executable: %s", fuzzTestExecutable)
	runnerOpts := &libfuzzer.RunnerOptions{
		FuzzTarget:          fuzzTestExecutable,
		SeedsDir:            c.opts.seedsDirs[0],
		AdditionalSeedsDirs: c.opts.seedsDirs[1:],
		Dictionary:          c.opts.dictionary,
		EngineArgs:          c.opts.engineArgs,
		FuzzTargetArgs:      c.opts.fuzzTargetArgs,
		ReportHandler:       &reportHandler{},
		Timeout:             c.opts.timeout,
		UseMinijail:         c.opts.useSandbox,
	}
	runner := libfuzzer.NewRunner(runnerOpts)

	// Handle cleanup (terminating the fuzzer process) when receiving
	// termination signals
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	go func() {
		s := <-sigs
		log.Infof("\nReceived %s", s.String())
		runner.Cleanup()
		os.Exit(128 + int(s.(syscall.Signal)))
	}()

	return runner.Run(context.Background())
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

	// Users should pass the environment variable FUZZ_TEST_LDFLAGS to
	// the linker command building the fuzz test. For libfuzzer, we set
	// it to "-fsanitize=fuzzer" to build a libfuzzer binary.
	env, err = envutil.Setenv(env, "FUZZ_TEST_LDFLAGS", "-fsanitize=fuzzer")
	if err != nil {
		return nil, err
	}

	return env, nil
}

type reportHandler struct{}

func (s *reportHandler) Handle(report *report.Report) error {
	jsonString, err := stringutil.ToJsonString(report)
	if err != nil {
		return err
	}
	fmt.Println(jsonString)
	return nil
}
