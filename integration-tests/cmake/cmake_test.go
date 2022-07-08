package cmake

import (
	"bufio"
	"context"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/otiai10/copy"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"code-intelligence.com/cifuzz/pkg/artifact"
	"code-intelligence.com/cifuzz/tools/install"
	"code-intelligence.com/cifuzz/util/executil"
	"code-intelligence.com/cifuzz/util/fileutil"
)

func TestIntegration_InitCreateRunBundle(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	if runtime.GOOS == "windows" {
		t.Skip("This test doesn't work on Windows yet")
	}

	installer, err := install.NewInstaller(nil)
	require.NoError(t, err)
	err = installer.InstallCIFuzzAndDeps()
	require.NoError(t, err)
	defer installer.Cleanup()
	err = os.Setenv("CMAKE_PREFIX_PATH", installer.InstallDir)
	require.NoError(t, err)

	dir := copyTestdataDir(t)
	defer fileutil.Cleanup(dir)
	t.Logf("executing cmake integration test in %s", dir)

	// Execute the root command
	cifuzz := installer.CIFuzzExecutablePath()
	cmd := executil.Command(cifuzz)
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	t.Logf("Command: %s", cmd.String())
	err = cmd.Run()
	require.NoError(t, err)

	// Execute the init command
	cmd = executil.Command(cifuzz, "init", "-C", dir)
	cmd.Dir = dir
	stderrPipe, err := cmd.StderrTeePipe(os.Stderr)
	require.NoError(t, err)
	t.Logf("Command: %s", cmd.String())
	err = cmd.Run()
	require.NoError(t, err)
	followStepsPrintedByInitCommand(t, stderrPipe, filepath.Join(dir, "CMakeLists.txt"))
	err = stderrPipe.Close()
	require.NoError(t, err)

	// Execute the create command
	outputPath := filepath.Join("src", "parser", "parser_fuzz_test.cpp")
	cmd = executil.Command(cifuzz, "create", "-C", dir, "cpp", "--output", outputPath)
	cmd.Dir = dir
	stderrPipe, err = cmd.StderrTeePipe(os.Stderr)
	require.NoError(t, err)
	cmd.Stdout = os.Stdout
	require.NoError(t, err)
	t.Logf("Command: %s", cmd.String())
	err = cmd.Run()
	require.NoError(t, err)

	// Check that the fuzz test was created in the correct directory
	fuzzTestPath := filepath.Join(dir, outputPath)
	require.FileExists(t, fuzzTestPath)
	followStepsPrintedByCreateCommand(t, stderrPipe, filepath.Join(filepath.Dir(fuzzTestPath), "CMakeLists.txt"))
	err = stderrPipe.Close()
	require.NoError(t, err)

	// Run the (empty) fuzz test
	runFuzzer(t, cifuzz, dir, regexp.MustCompile(`^paths: \d+`), true)

	// Make the fuzz test call a function. Before we do that, we sleep
	// for one second, to avoid make implementations which only look at
	// the full seconds of the timestamp to not rebuild the target, see
	// https://www.gnu.org/software/autoconf/manual/autoconf-2.63/html_node/Timestamps-and-Make.html
	time.Sleep(time.Second)
	modifyFuzzTestToCallFunction(t, fuzzTestPath)
	// Run the fuzz test
	runFuzzer(t, cifuzz, dir, regexp.MustCompile(`^SUMMARY: UndefinedBehaviorSanitizer`), false)

	// Bundle the fuzz into an archive.
	archivePath := filepath.Join(dir, "parser_fuzz_test.tar.gz")
	cmd = executil.Command(cifuzz, "bundle", "parser_fuzz_test", "-o", archivePath)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	require.NoError(t, err)
	require.FileExists(t, archivePath)

	// Extract the archive into a temporary directory.
	archiveDir, err := os.MkdirTemp("", "cifuzz-fuzzing-archive-*")
	require.NoError(t, err)
	defer fileutil.Cleanup(archiveDir)
	archiveFile, err := os.Open(archivePath)
	require.NoError(t, err)
	err = artifact.ExtractArchiveForTestsOnly(archiveFile, archiveDir)
	require.NoError(t, err)

	// Read the fuzzer path from the YAML.
	metadataPath := filepath.Join(archiveDir, "cifuzz.yaml")
	require.FileExists(t, metadataPath)
	metadataYaml, err := os.ReadFile(metadataPath)
	require.NoError(t, err)
	// We use a simple regex here instead of duplicating knowledge of our metadata YAML schema.
	fuzzerPathPattern := regexp.MustCompile(`\W*path: (.*)`)
	fuzzerPath := filepath.Join(archiveDir, string(fuzzerPathPattern.FindSubmatch(metadataYaml)[1]))
	require.FileExists(t, fuzzerPath)

	// Run the fuzzer on the empty input to verify that it finds all its runtime dependencies.
	cmd = executil.Command(fuzzerPath, "-runs=0")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	require.NoError(t, err)
}

func copyTestdataDir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "cifuzz-cmake-testdata-")
	require.NoError(t, err)

	// Get the path to the testdata dir
	cwd, err := os.Getwd()
	require.NoError(t, err)
	testDataDir := filepath.Join(cwd, "testdata")

	// Copy the testdata dir to the temporary directory
	err = copy.Copy(testDataDir, dir)
	require.NoError(t, err)

	return dir
}

func runFuzzer(t *testing.T, cifuzz string, dir string, expectedOutput *regexp.Regexp, terminate bool) {
	t.Helper()

	const timeout = 2 * time.Minute
	runCtx, closeRunCtx := context.WithTimeout(context.Background(), timeout)
	defer closeRunCtx()
	cmd := executil.CommandContext(runCtx, cifuzz, "run", "-v", "parser_fuzz_test", "--engine-arg=-seed=1")
	cmd.Dir = dir
	stdoutPipe, err := cmd.StdoutTeePipe(os.Stdout)
	require.NoError(t, err)
	cmd.Stderr = os.Stderr

	// Terminate the cifuzz process when we receive a termination signal
	// (else the test won't stop). An alternative would be to run the
	// command in the foreground, via syscall.SysProcAttr.Foreground,
	// but that's not supported on Windows.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	go func() {
		s := <-sigs
		t.Logf("Received %s", s.String())
		err = cmd.TerminateProcessGroup()
		require.NoError(t, err)
	}()

	t.Logf("Command: %s", cmd.String())
	err = cmd.Start()
	require.NoError(t, err)

	waitErrCh := make(chan error)
	// Wait for the command to exit in a go routine, so that below
	// we can cancel waiting when the context is done
	go func() {
		waitErrCh <- cmd.Wait()
	}()

	// Check that the output contains the expected output
	var seenExpectedOutput bool
	scanner := bufio.NewScanner(stdoutPipe)
	for scanner.Scan() {
		if expectedOutput.MatchString(scanner.Text()) {
			seenExpectedOutput = true
			if terminate {
				err = cmd.TerminateProcessGroup()
				require.NoError(t, err)
			}
		}
	}

	select {
	case err := <-waitErrCh:
		var exitErr *exec.ExitError
		// The expected exit code when the process was terminated via
		// SIGTERM is 128 + 15 = 143
		if seenExpectedOutput && terminate && errors.As(err, &exitErr) && exitErr.ExitCode() == 143 {
			return
		}
		require.NoError(t, err)
	case <-runCtx.Done():
		require.NoError(t, runCtx.Err())
	}
}

func followStepsPrintedByInitCommand(t *testing.T, initOutput io.Reader, cmakeLists string) {
	t.Helper()

	// Enable fuzz testing by adding to CMakeLists.txt the lines which
	// `cifuzz init` tells us to add

	// First, parse the `cifuzz init` output to find the lines we should
	// add to CMakeLists.txt - the ones that are indented.
	scanner := bufio.NewScanner(initOutput)
	var linesToAdd []string
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "    ") {
			linesToAdd = append(linesToAdd, strings.TrimSpace(scanner.Text()))
		}
	}
	if len(linesToAdd) == 0 {
		require.FailNow(t, "`cictl init` didn't print the lines which should be added to CMakeLists.txt")
	}
	f, err := os.OpenFile(cmakeLists, os.O_RDWR, 0700)
	require.NoError(t, err)
	defer f.Close()

	// Now find the correct line in CMakeLists.txt where the lines
	// should be added. It must be before any "add_subdirectory"
	// functions which add directories that might use functions from
	// the cifuzz CMake package.
	scanner = bufio.NewScanner(f)
	var lines []string
	var addedLines bool
	for scanner.Scan() {
		if !addedLines && strings.HasPrefix(scanner.Text(), "add_subdirectory") {
			// Found an "add_subdirectory" call, insert the lines before
			// that line
			lines = append(lines, linesToAdd...)
			addedLines = true
		}
		lines = append(lines, scanner.Text())
	}
	if !addedLines {
		require.FailNow(t, "Didn't find a \"add_subdirectory\" line in %v", cmakeLists)
	}

	// Write the new content of CMakeLists.txt back to file
	_, err = f.Seek(0, io.SeekStart)
	require.NoError(t, err)
	_, err = f.WriteString(strings.Join(lines, "\n"))
	require.NoError(t, err)
}

func followStepsPrintedByCreateCommand(t *testing.T, initOutput io.Reader, cmakeLists string) {
	t.Helper()

	// Create a CMake target for the fuzz test by adding to
	// CMakeLists.txt the lines which `cifuzz create` tells us to add

	// Parse the `cifuzz create` output to find the lines we should add
	// to CMakeLists.txt
	scanner := bufio.NewScanner(initOutput)
	var cMakeListsSuffix []byte
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "    ") {
			line := strings.TrimSpace(scanner.Text()) + "\n"
			cMakeListsSuffix = append(cMakeListsSuffix, []byte(line)...)
		}
	}
	if len(cMakeListsSuffix) == 0 {
		require.FailNow(t, "`cictl create` didn't print the lines which should be added to CMakeLists.txt")
	}

	// Append the lines to CMakeLists.txt
	f, err := os.OpenFile(cmakeLists, os.O_APPEND|os.O_WRONLY, 0700)
	require.NoError(t, err)
	defer f.Close()
	_, err = f.Write(cMakeListsSuffix)
	require.NoError(t, err)
}

func modifyFuzzTestToCallFunction(t *testing.T, fuzzTestPath string) {
	// Modify the fuzz test stub created by `cifuzz create` to actually
	// call a function.

	f, err := os.OpenFile(fuzzTestPath, os.O_RDWR, 0700)
	require.NoError(t, err)
	defer f.Close()
	scanner := bufio.NewScanner(f)
	// At the top of the file we add the required headers
	lines := []string{`#include "parser.h"`}
	var seenBeginningOfFuzzTestFunc bool
	var addedFunctionCall bool
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "FUZZ_TEST(") {
			seenBeginningOfFuzzTestFunc = true
		}
		// Insert the function call at the end of the FUZZ_TEST
		// function, right above the "}".
		if seenBeginningOfFuzzTestFunc && strings.HasPrefix(scanner.Text(), "}") {
			lines = append(lines, "  parse(std::string(reinterpret_cast<const char *>(data), size));")
			addedFunctionCall = true
		}
		lines = append(lines, scanner.Text())
	}
	require.NoError(t, scanner.Err())
	require.True(t, addedFunctionCall)

	// Write the new content of the fuzz test back to file
	_, err = f.Seek(0, io.SeekStart)
	require.NoError(t, err)
	_, err = f.WriteString(strings.Join(lines, "\n"))
	require.NoError(t, err)
	err = f.Close()
	require.NoError(t, err)

	// Add dependency on parser lib to CMakeLists.txt
	cmakeLists := filepath.Join(filepath.Dir(fuzzTestPath), "CMakeLists.txt")
	f, err = os.OpenFile(cmakeLists, os.O_APPEND|os.O_WRONLY, 0700)
	require.NoError(t, err)
	defer f.Close()
	_, err = f.WriteString("target_link_libraries(parser_fuzz_test PRIVATE parser)\n")
	require.NoError(t, err)
	err = f.Close()
	require.NoError(t, err)
}
