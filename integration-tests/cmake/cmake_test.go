package cmake

import (
	"bufio"
	"context"
	"io"
	"io/ioutil"
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

	"code-intelligence.com/cifuzz/tools/install"
	"code-intelligence.com/cifuzz/util/executil"
	"code-intelligence.com/cifuzz/util/fileutil"
)

func TestIntegration_InitCreateRun(t *testing.T) {
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

	//execute root command
	cifuzz := installer.CIFuzzExecutablePath()
	cmd := executil.Command(cifuzz)
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	t.Logf("Command: %s", cmd.String())
	err = cmd.Run()
	require.NoError(t, err)

	// execute init command
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

	// execute create command
	outputPath := filepath.Join("src/parser/parser_fuzz_test.cpp")
	cmd = executil.Command(cifuzz, "create", "-C", dir, "cpp", "--output", outputPath)
	cmd.Dir = dir
	stderrPipe, err = cmd.StderrTeePipe(os.Stderr)
	require.NoError(t, err)
	cmd.Stdout = os.Stdout
	require.NoError(t, err)
	t.Logf("Command: %s", cmd.String())
	err = cmd.Run()
	require.NoError(t, err)

	// check that the fuzz test was created in the correct directory
	fuzzTestPath := filepath.Join(dir, outputPath)
	require.FileExists(t, fuzzTestPath)
	followStepsPrintedByCreateCommand(t, stderrPipe, filepath.Join(filepath.Dir(fuzzTestPath), "CMakeLists.txt"))
	err = stderrPipe.Close()
	require.NoError(t, err)

	// run the (empty) fuzz test
	runFuzzer(t, cifuzz, dir, regexp.MustCompile(`^paths: \d+`), true)

	// make the fuzz test call a function
	modifyFuzzTestToCallFunction(t, fuzzTestPath)
	// run the fuzz test
	runFuzzer(t, cifuzz, dir, regexp.MustCompile(`^SUMMARY: UndefinedBehaviorSanitizer`), false)
}

func copyTestdataDir(t *testing.T) string {
	dir, err := ioutil.TempDir("", "cifuzz-cmake-testdata-")
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

	const timeout = 1 * time.Minute
	runCtx, closeRunCtx := context.WithTimeout(context.Background(), timeout)
	defer closeRunCtx()
	cmd := executil.CommandContext(runCtx, cifuzz, "run", "parser_fuzz_test")
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
	// add to CMakeLists.txt
	scanner := bufio.NewScanner(initOutput)
	var linesToAdd []string
	var isLineToAdd bool
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "Use 'cifuzz create' to create your first fuzz test") {
			break
		}
		if strings.HasPrefix(scanner.Text(), "Enable fuzz testing in your CMake project by adding the following lines") {
			isLineToAdd = true
			continue
		}
		if isLineToAdd {
			line := strings.TrimSpace(scanner.Text()) + "\n"
			linesToAdd = append(linesToAdd, line)
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
	_, err = f.Write([]byte(strings.Join(lines, "\n")))
	require.NoError(t, err)
}

func followStepsPrintedByCreateCommand(t *testing.T, initOutput io.Reader, cmakeLists string) {
	t.Helper()

	// Create a CMake target for the fuzz test by adding to
	// CMakeLists.txt the lines which `cifuzz create` tells us to add

	// Parse the `cifuzz create` output to find the lines we should add
	// to CMakeLists.txt
	scanner := bufio.NewScanner(initOutput)
	var toAdd []byte
	var isLineToAdd bool
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "Create a CMake target for the fuzz test as follows ") {
			isLineToAdd = true
			continue
		}
		if isLineToAdd {
			line := strings.TrimSpace(scanner.Text()) + "\n"
			toAdd = append(toAdd, []byte(line)...)
		}
	}
	if len(toAdd) == 0 {
		require.FailNow(t, "`cictl create` didn't print the lines which should be added to CMakeLists.txt")
	}

	// Append the lines to CMakeLists.txt
	f, err := os.OpenFile(cmakeLists, os.O_APPEND|os.O_WRONLY, 0700)
	require.NoError(t, err)
	defer f.Close()
	_, err = f.Write(toAdd)
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
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "FUZZ_TEST(") {
			seenBeginningOfFuzzTestFunc = true
		}
		// Insert the function call at the end of the FUZZ_TEST
		// function, right above the "}".
		if seenBeginningOfFuzzTestFunc && strings.HasPrefix(scanner.Text(), "}") {
			lines = append(lines, "  parse(std::string(reinterpret_cast<const char *>(data), size));")
		}
		lines = append(lines, scanner.Text())
	}

	// Write the new content of the fuzz test back to file
	_, err = f.Seek(0, io.SeekStart)
	require.NoError(t, err)
	_, err = f.Write([]byte(strings.Join(lines, "\n")))
	require.NoError(t, err)

	// Add dependency on parser lib to CMakeLists.txt
	cmakeLists := filepath.Join(filepath.Dir(fuzzTestPath), "CMakeLists.txt")
	f, err = os.OpenFile(cmakeLists, os.O_APPEND|os.O_WRONLY, 0700)
	require.NoError(t, err)
	defer f.Close()
	_, err = f.Write([]byte("target_link_libraries(parser_fuzz_test PRIVATE parser)\n"))
	require.NoError(t, err)
}
