package cmake

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"code-intelligence.com/cifuzz/internal/build"
	"code-intelligence.com/cifuzz/pkg/log"
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
	cacheVariables := map[string]string{
		"CMAKE_BUILD_TYPE":    cmakeBuildConfiguration,
		"CIFUZZ_ENGINE":       b.Engine,
		"CIFUZZ_SANITIZERS":   strings.Join(b.Sanitizers, ";"),
		"CIFUZZ_TESTING:BOOL": "ON",
	}
	var cacheArgs []string
	for key, value := range cacheVariables {
		cacheArgs = append(cacheArgs, "-D", fmt.Sprintf("%s=%s", key, value))
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
