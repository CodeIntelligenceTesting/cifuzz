package cmake

import (
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pkg/errors"

	"code-intelligence.com/cifuzz/internal/build"
	"code-intelligence.com/cifuzz/pkg/log"
	"code-intelligence.com/cifuzz/util/fileutil"
)

// The CMake configuration (also called "build type") to use for fuzzing runs.
// See enable_fuzz_testing in tools/cmake/CIFuzz/share/CIFuzz/CIFuzzFunctions.cmake for the rationale for using this
// build type.
const cmakeBuildConfiguration = "RelWithDebInfo"

type BuilderOptions struct {
	ProjectDir string
	Engine     string
	Sanitizers []string
	Stdout     io.Writer
	Stderr     io.Writer
}

func (opts *BuilderOptions) validate() error {
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

type Builder struct {
	*BuilderOptions
	BuildDir string
	env      []string
}

func NewBuilder(opts *BuilderOptions) (*Builder, error) {
	err := opts.validate()
	if err != nil {
		return nil, err
	}

	b := &Builder{BuilderOptions: opts}

	// Ensure that the build directory exists.
	// Note: Invoking CMake on the same build directory with different cache
	// variables is a no-op. For this reason, we have to encode all choices made
	// for the cache variables below in the path to the build directory.
	// Currently, this includes the fuzzing engine and the choice of sanitizers.
	b.BuildDir = filepath.Join(opts.ProjectDir, ".cifuzz-build", opts.Engine, strings.Join(opts.Sanitizers, "+"))
	err = os.MkdirAll(b.BuildDir, 0755)
	if err != nil {
		return nil, err
	}

	b.env, err = build.CommonBuildEnv()
	if err != nil {
		return nil, err
	}

	return b, nil
}

// Configure calls cmake to "Generate a project buildsystem" (that's the
// phrasing used by the CMake man page).
// Note: This is usually a no-op after the directory has been created once,
// even if cache variables change. However, if a previous invocation of this
// command failed during CMake generation and the command is run again, the
// build step would only result in a very unhelpful error message about
// missing Makefiles. By reinvoking CMake's configuration explicitly here,
// we either get a helpful error message or the build step will succeed if
// the user fixed the issue in the meantime.
func (b *Builder) Configure() error {
	cacheArgs := []string{
		"-DCMAKE_BUILD_TYPE=" + cmakeBuildConfiguration,
		"-DCIFUZZ_ENGINE=" + b.Engine,
		"-DCIFUZZ_SANITIZERS=" + strings.Join(b.Sanitizers, ";"),
		"-DCIFUZZ_TESTING:BOOL=ON",
	}
	if runtime.GOOS != "windows" {
		// Use relative paths in RPATH/RUNPATH so that binaries from the
		// build directory can find their shared libraries even when
		// packaged into an artifact.
		// On Windows, where there is no RPATH, there are two ways the user or
		// we can handle this:
		// 1. Use the TARGET_RUNTIME_DLLS generator expression introduced in
		//    CMake 3.21 to copy all DLLs into the directory of the executable
		//    in a post-build action.
		// 2. Add all library directories to PATH.
		cacheArgs = append(cacheArgs, "-DCMAKE_BUILD_RPATH_USE_ORIGIN:BOOL=ON")
	}

	cmd := exec.Command("cmake", append(cacheArgs, b.ProjectDir)...)
	// Redirect the build command's stdout to stderr to only have
	// reports printed to stdout
	cmd.Stdout = b.Stderr
	cmd.Stderr = b.Stderr
	cmd.Env = b.env
	cmd.Dir = b.BuildDir
	log.Debugf("Working directory: %s", cmd.Dir)
	log.Debugf("Command: %s", cmd.String())
	err := cmd.Run()
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// Build builds the specified fuzz test with CMake
func (b *Builder) Build(fuzzTest string) error {
	cmd := exec.Command(
		"cmake",
		"--build", b.BuildDir,
		"--config", cmakeBuildConfiguration,
		"--target", fuzzTest,
	)
	// Redirect the build command's stdout to stderr to only have
	// reports printed to stdout
	cmd.Stdout = b.Stderr
	cmd.Stderr = b.Stderr
	cmd.Env = b.env
	log.Debugf("Command: %s", cmd.String())
	err := cmd.Run()
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// FindFuzzTestExecutable uses the info files emitted by the CMake integration
// in the configure step to look up the absolute path of a fuzz test's
// executable.
func (b *Builder) FindFuzzTestExecutable(fuzzTest string) (string, error) {
	// The path to the info file for single-configuration CMake generators (e.g.
	// Makefiles).
	infoFileCandidate := filepath.Join(b.BuildDir, ".cifuzz", "fuzz_tests", fuzzTest)
	exists, err := fileutil.Exists(infoFileCandidate)
	if err != nil || !exists {
		// The path to the info file for multi-configuration CMake generators
		// (e.g. MSBuild).
		infoFileCandidate = filepath.Join(b.BuildDir, cmakeBuildConfiguration, ".cifuzz", "fuzz_tests", fuzzTest)
		exists, err = fileutil.Exists(infoFileCandidate)
	}
	if err != nil {
		return "", err
	}
	if !exists {
		return "", errors.Errorf("failed to find executable for fuzz test %q", fuzzTest)
	}
	fuzzTestExecutable, err := ioutil.ReadFile(infoFileCandidate)
	if err != nil {
		return "", errors.WithStack(err)
	}
	return string(fuzzTestExecutable), nil
}
