package other

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pkg/errors"

	"code-intelligence.com/cifuzz/internal/build"
	"code-intelligence.com/cifuzz/pkg/cmdutils"
	"code-intelligence.com/cifuzz/pkg/log"
	"code-intelligence.com/cifuzz/pkg/runfiles"
	"code-intelligence.com/cifuzz/util/envutil"
	"code-intelligence.com/cifuzz/util/executil"
	"code-intelligence.com/cifuzz/util/fileutil"
)

type BuilderOptions struct {
	BuildCommand string
	Stdout       io.Writer
	Stderr       io.Writer
}

type Builder struct {
	*BuilderOptions
	env []string
}

func NewBuilder(opts *BuilderOptions) (*Builder, error) {
	var err error
	b := &Builder{BuilderOptions: opts}

	b.env, err = build.CommonBuildEnv()
	if err != nil {
		return nil, err
	}

	// Set CFLAGS, CXXFLAGS, LDFLAGS, and FUZZ_TEST_LDFLAGS which must
	// be passed to the build commands by the build system.
	b.env, err = setBuildFlagsEnvVars(b.env)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// Build builds the specified fuzz test with CMake
func (b *Builder) Build() error {
	var err error

	// Run the build command
	cmd := exec.Command("/bin/sh", "-c", b.BuildCommand)
	// Redirect the build command's stdout to stderr to only have
	// reports printed to stdout
	cmd.Stdout = b.Stdout
	cmd.Stderr = b.Stderr
	cmd.Env = b.env
	log.Debugf("Command: %s", cmd.String())
	err = cmd.Run()
	if err != nil {
		// It's expected that the build command might fail, so we print
		// the error without the stack trace.
		err = cmdutils.WrapExecError(err, cmd)
		log.Error(err)
		return cmdutils.ErrSilent
	}
	return nil
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

func (b *Builder) FindFuzzTestExecutable(fuzzTest string) (string, error) {
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
