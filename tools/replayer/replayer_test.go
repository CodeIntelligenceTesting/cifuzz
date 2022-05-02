package replayer

import (
	_ "embed"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed src/replayer.c
var replayerSrc []byte

//go:embed testdata/fuzz_target.c
var fuzzTargetSrc []byte

type compilerCase struct {
	compiler                    string
	flags                       []string
	outputFlag                  string
	disableFuzzerInitializeFlag string
}

// clang-cl is almost fully compatible with MSVC (cl.exe), but doesn't seem to support ASan and the /Za flag.
var clangCl = compilerCase{
	"clang-cl",
	[]string{
		"/W4",
		// Treat warnings as errors.
		"/WX",
		// Enable additional security warnings.
		"/sdl",
	},
	"/Fe%s",
	"/DDISABLE_FUZZER_INITIALIZE",
}

var msvc = compilerCase{
	"cl",
	append([]string{
		// Disable Microsoft extensions to the C90 standard.
		"/Za",
		"/fsanitize=address",
		// Sanitizer runtimes have to be linked manually on Windows:
		// https://devblogs.microsoft.com/cppblog/addresssanitizer-asan-for-windows-with-msvc/
		"/wholearchive:clang_rt.asan-x86_64.lib",
		// Using ASan without debug symbols may result in a warning, which would be fatal due to /WX.
		"/Zi",
	}, clangCl.flags...),
	clangCl.outputFlag,
	clangCl.disableFuzzerInitializeFlag,
}

var clang = compilerCase{
	"clang",
	[]string{
		"-Wall",
		"-Wextra",
		"-Werror",
		// Add debug info to make ASan and UBSan findings more useful.
		"-g",
		"-fsanitize=address,undefined",
		// Make UBSan findings assertable by aborting.
		"-fsanitize-undefined-trap-on-error",
		// Disable compiler-specific extensions and use the C90 standard.
		"-ansi",
		"-o",
	},
	"%s",
	"-DDISABLE_FUZZER_INITIALIZE",
}

// MinGW lacks ASan and UBSan, but is otherwise compatible with Unix gcc.
var mingw = compilerCase{
	"gcc",
	[]string{
		"-Wall",
		"-Wextra",
		"-pedantic",
		"-pedantic-errors",
		"-Werror",
		"-ansi",
		"-o",
	},
	"%s",
	"-DDISABLE_FUZZER_INITIALIZE",
}

// On Unix, gcc supports ASan and UBSan.
var gcc = compilerCase{
	"gcc",
	append([]string{
		// Add debug info to make ASan and UBSan findings more useful.
		"-g",
		"-fsanitize=address,undefined",
		// Make UBSan findings assertable by aborting.
		"-fsanitize-undefined-trap-on-error",
	}, mingw.flags...),
	mingw.outputFlag,
	mingw.disableFuzzerInitializeFlag,
}

type runCase struct {
	inputs     []string
	normalExit bool
}

var baseRunCases = []runCase{
	{
		[]string{},
		true,
	},
	{
		[]string{"foo"},
		true,
	},
	{
		[]string{"foo", "bar"},
		true,
	},
	{
		[]string{"foo", "assert", "bar"},
		false,
	},
	{
		[]string{"foo", "return", "bar"},
		false,
	},
}

var asanRunCase = runCase{
	[]string{"foo", "asan", "bar"},
	false,
}

var ubsanRunCase = runCase{
	[]string{"foo", "ubsan", "bar"},
	false,
}

var baseTempDir string

func TestMain(m *testing.M) {
	var err error
	baseTempDir, err = ioutil.TempDir("", "cifuzz-replayer")
	if err != nil {
		log.Fatalf("Failed to create temp dir for tests: %+v", err)
	}
	defer os.RemoveAll(baseTempDir)
	m.Run()
}

func TestIntegrationReplayerWithMsvc(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	if runtime.GOOS != "windows" {
		t.Skip("MSVC is only available on Windows")
	}
	t.Parallel()

	// MSVC (cl.exe) does not support UBSan.
	subtestCompileAndRunWithFuzzerInitialize(t, msvc, append(baseRunCases, asanRunCase))
	subtestCompileAndRunWithoutFuzzerInitialize(t, msvc)
}

func TestIntegrationReplayerWithClangCl(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	if runtime.GOOS != "windows" {
		t.Skip("clang-cl is only available on Windows")
	}
	t.Parallel()

	// clang-cl does not seem to support sanitizers.
	// CI runs fail with the error referenced in https://github.com/llvm/llvm-project/issues/52728.
	subtestCompileAndRunWithFuzzerInitialize(t, clangCl, baseRunCases)
	subtestCompileAndRunWithoutFuzzerInitialize(t, clangCl)
}

func TestIntegrationReplayerWithClang(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	if runtime.GOOS == "windows" {
		t.Skip("clang on Windows is covered by the clang-cl test")
	}
	t.Parallel()

	subtestCompileAndRunWithFuzzerInitialize(t, clang, append(baseRunCases, asanRunCase, ubsanRunCase))
	subtestCompileAndRunWithoutFuzzerInitialize(t, clang)
}

func TestIntegrationReplayerWithGcc(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	if runtime.GOOS == "windows" {
		t.Skip("gcc on Windows is covered by the MinGW test")
	}
	t.Parallel()

	subtestCompileAndRunWithFuzzerInitialize(t, gcc, append(baseRunCases, asanRunCase, ubsanRunCase))
	subtestCompileAndRunWithoutFuzzerInitialize(t, gcc)
}

func TestIntegrationReplayerWithMingw(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	if runtime.GOOS != "windows" {
		t.Skip("MinGW is only available on Windows")
	}
	t.Parallel()

	// MinGW does not support sanitizers.
	subtestCompileAndRunWithFuzzerInitialize(t, mingw, baseRunCases)
	subtestCompileAndRunWithoutFuzzerInitialize(t, mingw)
}

func TestIntegrationReplayerWithNoAsserts(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Parallel()

	tempDir, err := ioutil.TempDir(baseTempDir, "")
	require.NoError(t, err)

	if runtime.GOOS == "windows" {
		compileReplayer(t, tempDir, clangCl.compiler, clangCl.outputFlag, append([]string{"/DNDEBUG"}, clangCl.flags...)...)
	} else {
		compileReplayer(t, tempDir, clang.compiler, clang.outputFlag, append([]string{"-DNDEBUG"}, clang.flags...)...)
	}
}

func subtestCompileAndRunWithFuzzerInitialize(t *testing.T, cc compilerCase, rcs []runCase) {
	t.Run("WithFuzzerInitialize", func(t *testing.T) {
		t.Parallel()

		tempDir, err := ioutil.TempDir(baseTempDir, "")
		require.NoError(t, err)

		replayer := compileReplayer(t, tempDir, cc.compiler, cc.outputFlag, cc.flags...)

		for _, rc := range rcs {
			// Capture loop variable in goroutine, see https://gist.github.com/posener/92a55c4cd441fc5e5e85f27bca008721.
			rc := rc
			t.Run(strings.Join(rc.inputs, ","), func(t *testing.T) {
				t.Parallel()

				expectedOut := []string{
					// Assert that LLVMFuzzerInitialize has been executed.
					fmt.Sprintf("init(%d,%s)", len(rc.inputs)+1 /* argc */, replayer /* argv[0] */),
				}
				for _, input := range rc.inputs {
					// The output of the fuzz target ends with the first crash, so don't include the magic
					// inputs and any subsequent ones in the expected output.
					if input == "asan" || input == "ubsan" || input == "assert" || input == "return" {
						break
					}
					expectedOut = append(expectedOut, fmt.Sprintf("'%s'", input))
				}

				out, err := runReplayer(t, tempDir, replayer, rc.inputs...)
				if rc.normalExit {
					if exitErr, ok := err.(*exec.ExitError); ok {
						require.NoError(t, err, string(exitErr.Stderr))
					} else {
						require.NoError(t, err)
					}
				} else {
					require.Error(t, err)
					require.IsType(t, &exec.ExitError{}, err, err.Error())
				}
				assert.Equal(t, expectedOut, out)
			})
		}
	})
}

func subtestCompileAndRunWithoutFuzzerInitialize(t *testing.T, cc compilerCase) {
	t.Run("WithoutFuzzerInitialize", func(t *testing.T) {
		t.Parallel()

		tempDir, err := ioutil.TempDir(baseTempDir, "")
		require.NoError(t, err)

		replayer := compileReplayer(t, tempDir, cc.compiler, cc.outputFlag, append(
			[]string{cc.disableFuzzerInitializeFlag},
			cc.flags...,
		)...)

		out, err := runReplayer(t, tempDir, replayer, "foo", "bar")
		if exitErr, ok := err.(*exec.ExitError); ok {
			require.NoError(t, err, string(exitErr.Stderr))
		} else {
			require.NoError(t, err)
		}
		assert.Equal(t, []string{"'foo'", "'bar'"}, out)
	})
}

// compileReplayer expects the last flag to be the compiler's equivalent of '-o' (if necessary) and returns the path to
// the resulting executable.
func compileReplayer(t *testing.T, tempDir string, compiler string, outputFlag string, flags ...string) string {
	tempDir, err := ioutil.TempDir(tempDir, "")
	require.NoError(t, err)

	reproducerSrcFile := filepath.Join(tempDir, "replayer.c")
	err = ioutil.WriteFile(reproducerSrcFile, replayerSrc, 0700)
	require.NoError(t, err)
	fuzzTargetSrcFile := filepath.Join(tempDir, "fuzz_target.c")
	err = ioutil.WriteFile(fuzzTargetSrcFile, fuzzTargetSrc, 0700)
	require.NoError(t, err)

	outBasename := "replayer"
	if runtime.GOOS == "windows" {
		outBasename += ".exe"
	}
	outFile := filepath.Join(tempDir, outBasename)
	var args []string
	args = append(args, flags...)
	args = append(args, fmt.Sprintf(outputFlag, outFile))
	args = append(args, reproducerSrcFile, fuzzTargetSrcFile)
	c := exec.Command(compiler, args...)
	// Emit additional compiler outputs into a test-exclusive directory.
	c.Dir = tempDir
	out, err := c.CombinedOutput()
	require.NoErrorf(t, err, "Failed to execute %q: %+v\n%s", c.String(), err, string(out))

	return outFile
}

func runReplayer(t *testing.T, tempDir string, replayerPath string, inputs ...string) ([]string, error) {
	var inputPaths []string
	for _, input := range inputs {
		tempFile, err := ioutil.TempFile(tempDir, "input")
		require.NoError(t, err)
		_, err = tempFile.WriteString(input)
		require.NoError(t, err)
		inputPaths = append(inputPaths, tempFile.Name())
		err = tempFile.Close()
		require.NoError(t, err)
	}

	c := exec.Command(replayerPath, inputPaths...)
	out, err := c.Output()
	// Split on both \r\n (Windows) and \n (Unix) after removing trailing newlines.
	outLines := strings.Split(strings.ReplaceAll(strings.TrimSpace(string(out)), "\r\n", "\n"), "\n")
	return outLines, err
}
