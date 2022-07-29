package cmake

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/otiai10/copy"
	"github.com/stretchr/testify/require"

	"code-intelligence.com/cifuzz/pkg/install"
	"code-intelligence.com/cifuzz/util/executil"
	"code-intelligence.com/cifuzz/util/fileutil"
	"code-intelligence.com/cifuzz/util/stringutil"
)

var expectedFinding = regexp.MustCompile(`^==\d*==ERROR: AddressSanitizer: heap-buffer-overflow`)
var filteredLine = regexp.MustCompile(`child process \d+ exited`)

func TestIntegration_Make_RunCoverage(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	if runtime.GOOS == "windows" {
		t.Skip("Make support is only available on Unix")
	}

	// Create installation bundle
	projectDir, err := install.FindProjectDir()
	require.NoError(t, err)
	targetDir := filepath.Join(projectDir, "installer-bundle", "bundle")
	err = os.RemoveAll(targetDir)
	require.NoError(t, err)

	opts := install.Options{Version: "dev", TargetDir: targetDir}
	bundler, err := install.NewInstallationBundler(opts)
	require.NoError(t, err)
	err = bundler.BuildCIFuzzAndDeps()
	require.NoError(t, err)
	defer bundler.Cleanup()

	// Install CIFuzz in temp folder
	installDir, err := os.MkdirTemp("", "cifuzz-")
	require.NoError(t, err)
	installDir = filepath.Join(installDir, "cifuzz")
	installCmd := exec.Command("go", "run", "-tags", "installer", "cmd/installer/main.go", "-i", installDir)
	installCmd.Dir = projectDir
	t.Logf("Command: %s", installCmd.String())
	err = installCmd.Run()
	require.NoError(t, err)

	cifuzz := install.CIFuzzExecutablePath(filepath.Join(installDir, "bin"))

	dir := copyMakeExampleDir(t)
	defer fileutil.Cleanup(dir)
	t.Logf("executing make integration test in %s", dir)

	// Run the two fuzz tests and verify that they crash with the expected finding.
	runFuzzer(t, cifuzz, dir, "my_fuzz_test_1", expectedFinding)
	runFuzzer(t, cifuzz, dir, filepath.Join(dir, "my_fuzz_test_2"), expectedFinding)

	createCoverageReport(t, cifuzz, dir)
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
		"--build-command", "make clean && make "+filepath.Base(fuzzTest),
		// The crashes are expected to be found quickly.
		"--engine-arg=-runs=1000000",
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
		if filteredLine.MatchString(scanner.Text()) {
			require.FailNow(t, "Found line in output which should have been filtered", scanner.Text())
		}
	}
	require.True(t, seenExpectedOutput, "Did not see %q in fuzzer output", expectedOutput.String())
}

func createCoverageReport(t *testing.T, cifuzz string, dir string) {
	t.Helper()

	cmd := executil.Command(cifuzz, "coverage", "-v",
		"--build-command=make clean && make my_fuzz_test_1",
		"my_fuzz_test_1")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	t.Logf("Command: %s", strings.Join(stringutil.QuotedStrings(cmd.Args), " "))
	err := cmd.Run()
	require.NoError(t, err)

	// Check that the coverage report was created
	reportPath := filepath.Join(dir, "my_fuzz_test_1.coverage.html")
	require.FileExists(t, reportPath)

	// Check that the coverage report contains coverage for the api.cpp
	// source file
	bytes, err := os.ReadFile(reportPath)
	require.NoError(t, err)
	require.Contains(t, string(bytes), "api.cpp")
}
