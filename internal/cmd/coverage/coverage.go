package coverage

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"code-intelligence.com/cifuzz/internal/build/cmake"
	"code-intelligence.com/cifuzz/internal/build/other"
	"code-intelligence.com/cifuzz/internal/completion"
	"code-intelligence.com/cifuzz/internal/config"
	"code-intelligence.com/cifuzz/pkg/cmdutils"
	"code-intelligence.com/cifuzz/pkg/log"
	"code-intelligence.com/cifuzz/pkg/minijail"
	"code-intelligence.com/cifuzz/pkg/runfiles"
	"code-intelligence.com/cifuzz/util/envutil"
	"code-intelligence.com/cifuzz/util/fileutil"
	"code-intelligence.com/cifuzz/util/stringutil"
)

var sharedLibraryRegex = regexp.MustCompile(`^.+\.((so)|(dylib))(\.\d\w*)*$`)

type coverageOptions struct {
	BuildSystem    string   `mapstructure:"build-system"`
	BuildCommand   string   `mapstructure:"build-command"`
	SeedCorpusDirs []string `mapstructure:"seed-corpus-dirs"`
	FuzzTestArgs   []string `mapstructure:"fuzz-test-args"`
	UseSandbox     bool     `mapstructure:"use-sandbox"`

	ProjectDir string
	fuzzTest   string
}

func (opts *coverageOptions) validate() error {
	var err error

	opts.SeedCorpusDirs, err = cmdutils.ValidateSeedCorpusDirs(opts.SeedCorpusDirs)
	if err != nil {
		log.Error(err, err.Error())
		return cmdutils.ErrSilent
	}

	// If the default seed corpus dir exists, add it to the list of
	// seed corpus dirs
	defaultSeedCorpusDir := cmdutils.DefaultSeedCorpusDir(opts.fuzzTest)
	defaultSeedCorpusDir, err = filepath.Abs(defaultSeedCorpusDir)
	if err != nil {
		return errors.WithStack(err)
	}
	exists, err := fileutil.Exists(defaultSeedCorpusDir)
	if err != nil {
		return err
	}
	if exists && !stringutil.Contains(opts.SeedCorpusDirs, defaultSeedCorpusDir) {
		opts.SeedCorpusDirs = append(opts.SeedCorpusDirs, defaultSeedCorpusDir)
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

	// To build with other build systems, a build command must be provided
	if opts.BuildSystem == config.BuildSystemOther && opts.BuildCommand == "" {
		msg := `Flag "build-command" must be set when using the build system type "other"`
		return cmdutils.WrapIncorrectUsageError(errors.New(msg))
	}

	return nil
}

type coverageCmd struct {
	*cobra.Command
	opts   *coverageOptions
	tmpDir string
}

func New() *cobra.Command {
	opts := &coverageOptions{}

	cmd := &cobra.Command{
		Use:   "coverage [flags] <fuzz test>",
		Short: "Generate a coverage report for a fuzz test",
		// TODO: Write long description
		Long:              "",
		ValidArgsFunction: completion.ValidFuzzTests,
		Args:              cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Bind viper keys to flags. We can't do this in the New
			// function, because that would re-bind viper keys which
			// were bound to the flags of other commands before.
			cmdutils.ViperMustBindPFlag("build-command", cmd.Flags().Lookup("build-command"))
			cmdutils.ViperMustBindPFlag("seed-corpus-dirs", cmd.Flags().Lookup("seed-corpus"))
			cmdutils.ViperMustBindPFlag("fuzz-test-args", cmd.Flags().Lookup("fuzz-test-arg"))
			cmdutils.ViperMustBindPFlag("use-sandbox", cmd.Flags().Lookup("use-sandbox"))

			projectDir, err := config.ParseProjectConfig(opts)
			if err != nil {
				return err
			}
			opts.ProjectDir = projectDir

			opts.fuzzTest = args[0]
			return opts.validate()
		},
		RunE: func(c *cobra.Command, args []string) error {
			cmd := coverageCmd{Command: c, opts: opts}
			return cmd.run()
		},
	}

	// Note: If a flag should be configurable via cifuzz.yaml as well,
	// bind it to viper in the PreRunE function.
	cmd.Flags().String("build-command", "", `The command to build the fuzz test. Example: "make clean && make my-fuzz-test"`)
	cmd.Flags().StringArrayP("seed-corpus", "s", nil, "Directory containing sample inputs for the code under test.\nSee https://llvm.org/docs/LibFuzzer.html#corpus and\nhttps://aflplus.plus/docs/fuzzing_in_depth/#a-collecting-inputs.")
	cmd.Flags().StringArray("fuzz-test-arg", nil, "Command-line argument to pass to the fuzz test.")
	cmd.Flags().Bool("use-sandbox", false, "By default, fuzz tests are executed in a sandbox to prevent accidental damage to the system.\nUse --use-sandbox=false to run the fuzz test unsandboxed.\nOnly supported on Linux.")
	viper.SetDefault("use-sandbox", runtime.GOOS == "linux")

	return cmd
}

func (c *coverageCmd) run() error {
	var err error

	var baseTmpDir string
	if c.opts.UseSandbox {
		baseTmpDir = minijail.OutputDir
		err = os.MkdirAll(baseTmpDir, 0700)
		if err != nil {
			return errors.WithStack(err)
		}
	}
	c.tmpDir, err = os.MkdirTemp(baseTmpDir, "coverage-")
	if err != nil {
		return errors.WithStack(err)
	}
	defer fileutil.Cleanup(c.tmpDir)

	fuzzTestExecutable, err := c.buildFuzzTest()
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

	err = c.indexRawProfile(fuzzTestExecutable)
	if err != nil {
		return err
	}

	err = c.generateHTMLReport(fuzzTestExecutable)
	if err != nil {
		return err
	}

	log.Successf("Created coverage report %s", c.htmlReportPath(fuzzTestExecutable))

	return nil
}

func (c *coverageCmd) buildFuzzTest() (string, error) {
	log.Infof("Building %s", pterm.Style{pterm.Reset, pterm.FgLightBlue}.Sprintf(c.opts.fuzzTest))

	if c.opts.BuildSystem == config.BuildSystemCMake {
		builder, err := cmake.NewBuilder(&cmake.BuilderOptions{
			ProjectDir: c.opts.ProjectDir,
			Engine:     "replayer",
			Sanitizers: []string{"coverage"},
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
		builder, err := other.NewBuilder(&other.BuilderOptions{
			BuildCommand: c.opts.BuildCommand,
			Engine:       "replayer",
			Sanitizers:   []string{"coverage"},
			Stdout:       c.OutOrStdout(),
			Stderr:       c.ErrOrStderr(),
		})
		if err != nil {
			return "", err
		}
		err = builder.Build()
		if err != nil {
			return "", err
		}
		return builder.FindFuzzTestExecutable(c.opts.fuzzTest)
	} else {
		return "", errors.Errorf("Unsupported build system \"%s\"", c.opts.BuildSystem)
	}
}

func (c *coverageCmd) runFuzzTest(fuzzTestExecutable string) error {
	log.Infof("Running %s on corpus", pterm.Style{pterm.Reset, pterm.FgLightBlue}.Sprintf(c.opts.fuzzTest))
	log.Debugf("Executable: %s", fuzzTestExecutable)

	seedDirs := c.opts.SeedCorpusDirs

	generatedCorpusDir := cmdutils.GeneratedCorpusDir(c.opts.ProjectDir, c.opts.fuzzTest)
	exists, err := fileutil.Exists(generatedCorpusDir)
	if err != nil {
		return err
	}
	if exists {
		seedDirs = append(seedDirs, generatedCorpusDir)
	}

	// The environment we run the binary in
	var binaryEnv []string
	binaryEnv, err = envutil.Setenv(binaryEnv, "LLVM_PROFILE_FILE", c.rawProfilePattern())
	if err != nil {
		return err
	}

	// Use the absolute path to the fuzz test executable to be able to
	// add a Minijail binding for it
	fuzzTestExecutable, err = filepath.Abs(fuzzTestExecutable)
	if err != nil {
		return errors.WithStack(err)
	}

	// The environment we run minijail in
	wrapperEnv := os.Environ()

	args := append([]string{fuzzTestExecutable}, seedDirs...)
	if len(c.opts.FuzzTestArgs) > 0 {
		args = append(append(args, "--"), c.opts.FuzzTestArgs...)
	}

	if c.opts.UseSandbox {
		bindings := []*minijail.Binding{
			// The fuzz target must be accessible
			{Source: fuzzTestExecutable},
		}

		for _, dir := range seedDirs {
			bindings = append(bindings, &minijail.Binding{Source: dir})
		}

		// Set up Minijail
		mj, err := minijail.NewMinijail(&minijail.Options{
			Args:     args,
			Bindings: bindings,
			Env:      binaryEnv,
		})
		if err != nil {
			return err
		}
		defer mj.Cleanup()

		// Use the command which runs the fuzz test via minijail
		args = mj.Args
	} else {
		// We don't use minijail, so we can merge the binary and wrapper
		// environment
		for key, value := range envutil.ToMap(binaryEnv) {
			wrapperEnv, err = envutil.Setenv(wrapperEnv, key, value)
			if err != nil {
				return err
			}
		}
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = wrapperEnv

	if viper.GetBool("verbose") {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	log.Debugf("Command: %s", strings.Join(stringutil.QuotedStrings(cmd.Args), " "))
	err = cmd.Run()
	if err != nil {
		// It's expected that the fuzz test executable might fail, so we
		// print the error without the stack trace.
		err = cmdutils.WrapExecError(err, cmd)
		log.Error(err)
		return cmdutils.ErrSilent
	}
	return nil
}

func (c *coverageCmd) indexRawProfile(fuzzTestExecutable string) error {
	rawProfileFiles, err := c.rawProfileFiles()
	if err != nil {
		return err
	}

	llvmProfData, err := runfiles.Finder.LLVMProfDataPath()
	if err != nil {
		return err
	}

	args := append([]string{"merge", "-sparse", "-o", c.indexedProfilePath(fuzzTestExecutable)}, rawProfileFiles...)
	cmd := exec.Command(llvmProfData, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Debugf("Command: %s", strings.Join(stringutil.QuotedStrings(cmd.Args), " "))
	err = cmd.Run()
	if err != nil {
		return cmdutils.WrapExecError(errors.WithStack(err), cmd)
	}
	return nil
}

func (c *coverageCmd) generateHTMLReport(fuzzTestExecutable string) error {
	llvmCov, err := runfiles.Finder.LLVMCovPath()
	if err != nil {
		return err
	}

	// Add all shared objects in the project directory as arguments, for
	// the case that the executable uses those and we therefore want
	// them to be included in the coverage report.
	// TODO: Only add those objects which are actually used, and which
	//       might live outside of the project directory, by parsing the
	//       shared object dependencies of the executable (we could use
	//       cmake for that or do it ourselves in Go).
	var sharedObjects []string
	err = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.WithStack(err)
		}
		if info.IsDir() {
			return nil
		}
		// Ignore shared objects in .dSYM directories, to avoid llvm-cov
		// failing with:
		//
		//    Failed to load coverage: Unsupported coverage format version
		//
		if strings.Contains(path, "dSYM") {
			return nil
		}
		if sharedLibraryRegex.MatchString(info.Name()) {
			sharedObjects = append(sharedObjects, path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	args := []string{"show", "-instr-profile=" + c.indexedProfilePath(fuzzTestExecutable), "-format=html",
		fuzzTestExecutable}
	for _, path := range sharedObjects {
		args = append(args, "-object="+path)
	}

	cmd := exec.Command(llvmCov, args...)
	cmd.Stderr = os.Stderr
	log.Debugf("Command: %s", strings.Join(stringutil.QuotedStrings(cmd.Args), " "))
	output, err := cmd.Output()
	if err != nil {
		return cmdutils.WrapExecError(errors.WithStack(err), cmd)
	}

	// Write the HTML output to file
	err = os.WriteFile(c.htmlReportPath(fuzzTestExecutable), output, 0644)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (c *coverageCmd) rawProfilePattern() string {
	// TODO: According to the documentation [1], "%c" should be useful
	//       here, but for an unclear reason that results in no .profraw
	//       files being generated.
	//       [1] https://clang.llvm.org/docs/SourceBasedCodeCoverage.html#running-the-instrumented-program
	return filepath.Join(c.tmpDir, "%m.profraw")
}

func (c *coverageCmd) rawProfileFiles() ([]string, error) {
	files, err := filepath.Glob(filepath.Join(c.tmpDir, "*.profraw"))
	return files, errors.WithStack(err)
}

func (c *coverageCmd) indexedProfilePath(fuzzTestExecutable string) string {
	return filepath.Join(c.tmpDir, filepath.Base(fuzzTestExecutable)+".profdata")
}

func (c *coverageCmd) htmlReportPath(fuzzTestExecutable string) string {
	return filepath.Base(fuzzTestExecutable) + ".coverage.html"
}
