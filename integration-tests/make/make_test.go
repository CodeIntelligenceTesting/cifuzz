package cmake

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"

	"github.com/otiai10/copy"
	"github.com/stretchr/testify/require"

	"code-intelligence.com/cifuzz/tools/install"
	"code-intelligence.com/cifuzz/util/executil"
	"code-intelligence.com/cifuzz/util/fileutil"
)

func TestIntegration_Make_Run(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	if runtime.GOOS == "windows" {
		t.Skip("Make support is only available on Unix")
	}

	installer, err := install.NewInstaller(nil)
	require.NoError(t, err)
	err = installer.InstallCIFuzzAndDeps()
	require.NoError(t, err)
	defer installer.Cleanup()
	cifuzz := installer.CIFuzzExecutablePath()

	dir := copyMakeExampleDir(t)
	defer fileutil.Cleanup(dir)
	t.Logf("executing make integration test in %s", dir)

	// Run the two fuzz tests and verify that they crash with the expected finding.
	runFuzzer(t, cifuzz, dir, "my_fuzz_test_1", regexp.MustCompile(`^==\d*==ERROR: AddressSanitizer: heap-buffer-overflow`))
	runFuzzer(t, cifuzz, dir, "my_fuzz_test_2", regexp.MustCompile(`^==\d*==ERROR: AddressSanitizer: heap-buffer-overflow`))
}

func copyMakeExampleDir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "cifuzz-make-example-")
	require.NoError(t, err)

	// Get the path to the testdata dir
	cwd, err := os.Getwd()
	require.NoError(t, err)
	exampleDir := filepath.Join(cwd, "..", "..", "examples", "make")

	// Copy the example dir to the temporary directory
	err = copy.Copy(exampleDir, dir)
	require.NoError(t, err)

	return dir
}

func runFuzzer(t *testing.T, cifuzz string, dir string, fuzzTest string, expectedOutput *regexp.Regexp) {
	t.Helper()

	cmd := executil.Command(
		cifuzz,
		"run", "-v", fuzzTest,
		"--build-command", "make clean && make "+fuzzTest,
		// The crashes are expected to be found quickly.
		"--engine-arg=-run=1000",
		"--engine-arg=-seed=1",
	)
	cmd.Dir = dir
	stdoutPipe, err := cmd.StdoutTeePipe(os.Stdout)
	require.NoError(t, err)
	stderrPipe, err := cmd.StderrTeePipe(os.Stderr)
	require.NoError(t, err)

	t.Logf("Command: %s", cmd.String())
	err = cmd.Run()
	require.NoError(t, err)

	// Check that the output contains the expected output
	var seenExpectedOutput bool
	// cifuzz progress messages go to stdout.
	scanner := bufio.NewScanner(stdoutPipe)
	for scanner.Scan() {
		if expectedOutput.MatchString(scanner.Text()) {
			seenExpectedOutput = true
		}
	}
	// Fuzzer output goes to stderr.
	scanner = bufio.NewScanner(stderrPipe)
	for scanner.Scan() {
		if expectedOutput.MatchString(scanner.Text()) {
			seenExpectedOutput = true
		}
	}
	require.True(t, seenExpectedOutput, "Did not see %q in fuzzer output", expectedOutput.String())
}
