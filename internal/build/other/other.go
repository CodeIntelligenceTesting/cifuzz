package other

import (
	"fmt"
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
	"code-intelligence.com/cifuzz/util/stringutil"
)

type BuilderOptions struct {
	BuildCommand string
	Engine       string
	Sanitizers   []string
	Stdout       io.Writer
	Stderr       io.Writer
}

type Builder struct {
	*BuilderOptions
	env      []string
	buildDir string
}

func NewBuilder(opts *BuilderOptions) (*Builder, error) {
	var err error
	b := &Builder{BuilderOptions: opts}

	// Create a temporary build directory
	b.buildDir, err = os.MkdirTemp("", "cifuzz-build-")
	if err != nil {
		return nil, err
	}

	b.env, err = build.CommonBuildEnv()
	if err != nil {
		return nil, err
	}

	// Set CFLAGS, CXXFLAGS, LDFLAGS, and FUZZ_TEST_LDFLAGS which must
	// be passed to the build commands by the build system.
	switch opts.Engine {
	case "libfuzzer":
		for _, sanitizer := range opts.Sanitizers {
			if sanitizer != "address" && sanitizer != "undefined" {
				panic(fmt.Sprintf("Invalid sanitizer for engine %q: %q", opts.Engine, sanitizer))
			}
		}
		err = b.setLibFuzzerEnv()
	case "replayer":
		if !stringutil.Equal(opts.Sanitizers, []string{"coverage"}) {
			panic(fmt.Sprintf("Invalid sanitizers for engine %q: %q", opts.Engine, opts.Sanitizers))
		}
		err = b.setCoverageEnv()
	default:
		panic(fmt.Sprintf("Invalid engine %q", opts.Engine))
	}
	if err != nil {
		return nil, err
	}

	return b, nil
}

// Build builds the specified fuzz test with CMake
func (b *Builder) Build() error {
	var err error
	defer fileutil.Cleanup(b.buildDir)

	if b.Engine == "replayer" {
		// Build the replayer without coverage instrumentation
		replayerSource, err := runfiles.Finder.ReplayerSourcePath()
		if err != nil {
			return err
		}
		cmd := exec.Command("clang", "-c", replayerSource, "-o", filepath.Join(b.buildDir, "replayer.o"))
		cmd.Stdout = b.Stdout
		cmd.Stderr = b.Stderr
		log.Debugf("Command: %s", cmd.String())
		err = cmd.Run()
		if err != nil {
			return errors.WithStack(err)
		}
	}

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

var commonCFlags = []string{
	// Keep debug symbols
	"-g",
	// Do optimizations which don't harm debugging
	"-Og",
	// To get good stack frames for better debugging
	"-fno-omit-frame-pointer",
	// Conventional macro to conditionally compile out fuzzer road blocks
	// See https://llvm.org/docs/LibFuzzer.html#fuzzer-friendly-build-mode
	"-DFUZZING_BUILD_MODE_UNSAFE_FOR_PRODUCTION",
}

func (b *Builder) setLibFuzzerEnv() error {
	var err error

	// Set CFLAGS and CXXFLAGS. Note that these flags must not contain
	// spaces, because the environment variables are space separated.
	//
	// Note: Keep in sync with tools/cmake/CIFuzz/share/CIFuzz/CIFuzzFunctions.cmake
	cflags := append(commonCFlags, []string{
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
	}...)
	b.env, err = envutil.Setenv(b.env, "CFLAGS", strings.Join(cflags, " "))
	if err != nil {
		return err
	}
	b.env, err = envutil.Setenv(b.env, "CXXFLAGS", strings.Join(cflags, " "))
	if err != nil {
		return err
	}

	ldflags := []string{
		// ----- Flags used to build with ASan -----
		// Link ASan and UBSan runtime
		"-fsanitize=address,undefined",
		// To avoid issues with clang (not clang++) and UBSan, see
		// https://github.com/bazelbuild/bazel/issues/11122#issuecomment-896613570
		"-fsanitize-link-c++-runtime",
	}
	b.env, err = envutil.Setenv(b.env, "LDFLAGS", strings.Join(ldflags, " "))
	if err != nil {
		return err
	}

	// Users should pass the environment variable FUZZ_TEST_CFLAGS to the
	// compiler command building the fuzz test.
	cifuzzIncludePath, err := runfiles.Finder.CIFuzzIncludePath()
	if err != nil {
		return err
	}
	b.env, err = envutil.Setenv(b.env, "FUZZ_TEST_CFLAGS", "-I"+cifuzzIncludePath)
	if err != nil {
		return err
	}

	// Users should pass the environment variable FUZZ_TEST_LDFLAGS to
	// the linker command building the fuzz test. For libfuzzer, we set
	// it to "-fsanitize=fuzzer" to build a libfuzzer binary.
	b.env, err = envutil.Setenv(b.env, "FUZZ_TEST_LDFLAGS", "-fsanitize=fuzzer")
	if err != nil {
		return err
	}

	return nil
}

func (b *Builder) setCoverageEnv() error {
	var err error

	// Set CFLAGS and CXXFLAGS. Note that these flags must not contain
	// spaces, because the environment variables are space separated.
	//
	// Note: Keep in sync with tools/cmake/CIFuzz/share/CIFuzz/CIFuzzFunctions.cmake
	cflags := append(commonCFlags, []string{
		// ----- Flags used to build with code coverage -----
		"-fprofile-instr-generate",
		"-fcoverage-mapping",
	}...)
	b.env, err = envutil.Setenv(b.env, "CFLAGS", strings.Join(cflags, " "))
	if err != nil {
		return err
	}
	b.env, err = envutil.Setenv(b.env, "CXXFLAGS", strings.Join(cflags, " "))
	if err != nil {
		return err
	}

	// Users should pass the environment variable FUZZ_TEST_LDFLAGS to
	// the linker command building the fuzz test. When building for
	// coverage, we set it to the replayer object file which we built
	// before without coverage instrumentation.
	b.env, err = envutil.Setenv(b.env, "FUZZ_TEST_LDFLAGS", filepath.Join(b.buildDir, "replayer.o"))
	if err != nil {
		return err
	}

	return nil
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
