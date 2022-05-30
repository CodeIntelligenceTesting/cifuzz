package runner

import (
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"golang.org/x/exp/maps"
	"golang.org/x/term"

	"code-intelligence.com/cifuzz/pkg/report"
	"code-intelligence.com/cifuzz/pkg/runfiles"
	"code-intelligence.com/cifuzz/util/envutil"
	"code-intelligence.com/cifuzz/util/stringutil"
)

// Exit code when a sanitizer reports a bug
const SanitizerErrorExitCode = 78

// Exit code used by fuzzers which we didn't configure to return
// SanitizerErrorExitCode instead
// TODO(adrian): Use SanitizerErrorExitCode in all runners
const DeprecatedSanitizerErrorExitCode = 1

// Exit code used by libFuzzer when libFuzzer itself reports a bug. See
//
//   https://github.com/llvm/llvm-project/blob/75e33f71c2dae584b13a7d1186ae0a038ba98838/compiler-rt/lib/fuzzer/FuzzerOptions.h#L26
//
// This is configurable via the `-error_exitcode` libFuzzer flag, see
//
//   https://llvm.org/docs/LibFuzzer.html
//
// This is also used by jazzer to indicate that an issue was found.
const LibFuzzerErrorExitCode = 77

// Exit code used by libFuzzer on OOM errors. See
//
//   https://github.com/llvm/llvm-project/blob/75e33f71c2dae584b13a7d1186ae0a038ba98838/compiler-rt/lib/fuzzer/FuzzerOptions.h#L24
//
// This is (currently) not configurable via any libFuzzer flag.
const LibFuzzerOOMExitCode = 71

// Exit code used by libFuzzer on timeout errors. See
//
//   https://github.com/llvm/llvm-project/blob/75e33f71c2dae584b13a7d1186ae0a038ba98838/compiler-rt/lib/fuzzer/FuzzerOptions.h#L23
//
// This is configurable via the `-timeout_exitcode` libFuzzer flag, see
//
//   https://llvm.org/docs/LibFuzzer.html
//
const LibFuzzerTimeoutExitCode = 70

var defaultSanitizerOptions = map[string]string{
	// color defaults to "auto", which lets sanitizers detects whether colorized
	// output is supported by determining whether their stdout is a tty. Since
	// the sanitizers run in a subprocess that has its output redirected to a
	// pipe, this will never detect a tty even if cifuzz is running in one.
	// Instead, enable colored reports if and only if cifuzz itself runs in a
	// tty.
	"color": sanitizerOptionsColorValue(),
}

// Sender is an interface to something that can send.
type Sender interface {
	Send(report *report.Report) error
}

func SetSanitizerOptions(existingOptionsStr string, defaults map[string]string, overrides map[string]string) string {
	options := strings.Split(existingOptionsStr, ":")

	for key, val := range defaults {
		options = setDefaultIfNotSetAlready(options, key, val)
	}

	for key, val := range overrides {
		options = overrideOption(options, key, val)
	}

	return stringutil.JoinNonEmpty(options, ":")
}

func setDefaultIfNotSetAlready(options []string, key, value string) []string {
	for _, option := range options {
		if strings.HasPrefix(option, key+"=") {
			// The option is already set
			return options
		}
	}
	// The option doesn't exist yet so we append it
	return append(options, key+"="+value)
}

func overrideOption(options []string, key, value string) []string {
	for i, option := range options {
		if strings.HasPrefix(option, key+"=") {
			// Replace the option
			options[i] = key + "=" + value
			return options
		}
	}
	// The option doesn't exist yet so we append it
	return append(options, key+"="+value)
}

func SetASANOptions(env []string, defaults map[string]string, overrides map[string]string) ([]string, error) {
	options := envutil.Getenv(env, "ASAN_OPTIONS")
	options = SetSanitizerOptions(options, defaults, overrides)
	return envutil.Setenv(env, "ASAN_OPTIONS", options)
}

func SetCommonASANOptions(env []string) ([]string, error) {
	defaultOptions := maps.Clone(defaultSanitizerOptions)
	overrideOptions := map[string]string{
		// The default exit code when a sanitizer finds something is 1,
		// which makes it hard to differentiate between the case that
		// the sanitizer found something and unexpected errors, so we
		// set it to some less commonly used exit code here.
		// Documentation of this option:
		//
		//   https://github.com/google/sanitizers/wiki/SanitizerCommonFlags
		//
		"exitcode": strconv.Itoa(SanitizerErrorExitCode),
		// Per default this is set to false, except for darwin.
		// To have consistent behaviour on all supported operating systems
		// we are setting this explicitly to false
		"abort_on_error": "0",
	}
	return SetASANOptions(env, defaultOptions, overrideOptions)
}

func SetCommonUBSANOptions(env []string) ([]string, error) {
	defaultOptions := maps.Clone(defaultSanitizerOptions)
	maps.Copy(defaultOptions, map[string]string{
		// Instruct UBSAN (enabled for all sanitizers) to print full stack traces
		// instead of only the top stack frame with a relative file path.
		// This allows us to set more breakpoints while debugging undefined behaviour
		// findings and also ensures absolute file paths in the stack trace which can
		// be mapped to the project build directory.
		"print_stacktrace": "1",
	})
	options := envutil.Getenv(env, "UBSAN_OPTIONS")
	options = SetSanitizerOptions(options, defaultOptions, nil)
	return envutil.Setenv(env, "UBSAN_OPTIONS", options)
}

func AddEnvFlags(env []string, envVars []string) ([]string, error) {
	var err error
	for _, e := range envVars {
		split := strings.SplitN(e, "=", 2)
		if len(split) < 2 {
			return nil, errors.Errorf("Invalid environment variable, must be of the form KEY=VAL: %s", e)
		}
		key, val := split[0], split[1]
		env, err = envutil.Setenv(env, key, val)
		if err != nil {
			return nil, err
		}
	}
	return env, nil
}

func FuzzerEnvironment() ([]string, error) {
	var err error
	var env []string

	// Tell the address sanitizer where it can find llvm-symbolizer.
	// See https://clang.llvm.org/docs/AddressSanitizer.html#symbolizing-the-reports
	llvmSymbolizer, err := runfiles.Finder.LLVMSymbolizerPath()
	if err != nil {
		return nil, err
	}
	env, err = envutil.Setenv(env, "ASAN_SYMBOLIZER_PATH", llvmSymbolizer)
	if err != nil {
		return nil, err
	}

	// Tell llvm-symbolizer to strip the build dir from paths, to have
	// stack traces printed in the logs with relative paths, which are
	// less confusing when fuzzer-runner is executed on a different
	// system than the fuzzers were built on. Note that llvm-symbolizer
	// only gives us relative paths if the compiler command-line also
	// contained relative paths.
	// See https://llvm.org/docs/CommandGuide/llvm-symbolizer.html#cmdoption-llvm-symbolizer-relativenames
	env, err = envutil.Setenv(env, "LLVM_SYMBOLIZER_OPTS", "--relativenames")
	if err != nil {
		return nil, err
	}

	return env, nil
}

func SetLDLibraryPath(env []string, libraryDirs []string) ([]string, error) {
	// Add directories from the library dir flags to LD_LIBRARY_PATH
	var libDirsList string
	libDirsList = envutil.AddToColonSeparatedList(libDirsList, libraryDirs...)
	return envutil.Setenv(env, "LD_LIBRARY_PATH", libDirsList)
}

func sanitizerOptionsColorValue() string {
	// Colorize sanitizer reports if cifuzz itself is running in an interactive
	// terminal. Since we redirect unstructured output from fuzzer binaries to
	// stderr, that is the file descriptor we should check for tty-ness.
	if term.IsTerminal(int(os.Stderr.Fd())) {
		return "always"
	} else {
		return "never"
	}
}
