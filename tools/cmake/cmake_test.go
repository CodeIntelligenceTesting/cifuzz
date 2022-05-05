package cmake

import (
	"context"
	"encoding/xml"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

var baseTempDir string

func TestMain(m *testing.M) {
	var err error
	baseTempDir, err = ioutil.TempDir("", "cifuzz-cmake")
	if err != nil {
		log.Fatalf("Failed to create temp dir for tests: %+v", err)
	}
	defer os.RemoveAll(baseTempDir)
	m.Run()
}

func TestIntegrationCtestDefaultSettings(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Parallel()

	buildDir := build(t, nil)
	testResults := runTests(t, buildDir)
	assert.Equal(t, map[string]bool{
		// Without sanitizers, the seed corpus entries do not crash this target.
		"parser_fuzz_test_regression_test": true,
		// The target returns a non-zero value on every input and the replayer always runs on the empty input.
		"no_seed_corpus_fuzz_test_regression_test": false,
	}, testResults)
}

func TestIntegrationCtestWithAddressSanitizer(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Parallel()

	buildDir := build(t, map[string]string{"CIFUZZ_SANITIZERS": "address"})
	testResults := runTests(t, buildDir)
	assert.Equal(t, map[string]bool{
		// Crashes on the `asan_crash` input.
		"parser_fuzz_test_regression_test": false,
		// The target returns a non-zero value on every input and the replayer always runs on the empty input.
		"no_seed_corpus_fuzz_test_regression_test": false,
	}, testResults)
}

func TestIntegrationCtestWithUndefinedBehaviorSanitizer(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	if runtime.GOOS == "windows" {
		t.Skip("MSVC does not support UndefinedBehaviorSanitizer")
	}
	t.Parallel()

	buildDir := build(t, map[string]string{"CIFUZZ_SANITIZERS": "undefined"})
	testResults := runTests(t, buildDir)
	assert.Equal(t, map[string]bool{
		// Crashes on the `ubsan_crash` input.
		"parser_fuzz_test_regression_test": false,
		// The target returns a non-zero value on every input and the replayer always runs on the empty input.
		"no_seed_corpus_fuzz_test_regression_test": false,
	}, testResults)
}

func TestIntegrationBuildWithMultipleSanitizers(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	if runtime.GOOS == "windows" {
		t.Skip("MSVC does not support UndefinedBehaviorSanitizer")
	}
	t.Parallel()

	build(t, map[string]string{"CIFUZZ_SANITIZERS": "address;undefined"})
}

func build(t *testing.T, cacheVariables map[string]string) string {
	buildDir, err := ioutil.TempDir(baseTempDir, "build")
	require.NoError(t, err)

	var cacheArgs []string
	for key, value := range cacheVariables {
		cacheArgs = append(cacheArgs, "-D", fmt.Sprintf("%s=%s", key, value))
	}

	runInDir(t, buildDir, "cmake", append(cacheArgs, testDataDir(t))...)
	runInDir(t, buildDir, "cmake", "--build", ".")

	return buildDir
}

func runTests(t *testing.T, buildDir string) (testPassed map[string]bool) {
	junitReportFile := filepath.Join(buildDir, "report.xml")
	runInDirExpectingFailure(
		t,
		buildDir,
		"ctest",
		"--verbose",
		// On Windows, ctest requires a configuration to be specified explicitly.
		"-C",
		"Debug",
		// Instead of parsing CTest's unstructured console output, we let it emit an XML report that contains
		// information on which tests passed or failed.
		"--output-junit",
		junitReportFile,
	)
	require.FileExists(t, junitReportFile)
	junitReportXml, err := ioutil.ReadFile(junitReportFile)
	require.NoError(t, err)
	var junitReport junitTestSuite
	err = xml.Unmarshal(junitReportXml, &junitReport)
	require.NoError(t, err)

	// Parse the test report in JUnit's XML format to determine which tests passed.
	testPassed = make(map[string]bool)
	for _, testCase := range junitReport.TestCases {
		if testCase.Status == "run" {
			testPassed[testCase.Name] = true
		} else {
			testPassed[testCase.Name] = false
		}
	}
	return
}

func runInDir(t *testing.T, dir, command string, args ...string) []byte {
	return runInDirInternal(t, false, dir, command, args...)
}

func runInDirExpectingFailure(t *testing.T, dir, command string, args ...string) []byte {
	return runInDirInternal(t, true, dir, command, args...)
}

func runInDirInternal(t *testing.T, expectFailure bool, dir string, command string, args ...string) []byte {
	// A timeout of 5 minutes is long enough for all current tests, but stays well under the Go test timeout of
	// 10 minutes.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	c := exec.CommandContext(ctx, command, args...)
	c.Dir = dir
	out, err := c.Output()
	if exitErr, ok := err.(*exec.ExitError); ok {
		msg := fmt.Sprintf("%q exited with %d:\nstderr:\n%s\nstdout:\n%s", c.String(), exitErr.ExitCode(), string(exitErr.Stderr), string(out))
		if !expectFailure {
			require.NoError(t, exitErr, msg)
		} else {
			// When expecting a failure, the command may still fail for an unexpected reason. Thus, in this case, always
			// log the relevant information.
			t.Log(msg)
		}
		return nil
	} else {
		msg := fmt.Sprintf("%q failed to execute with error:%v\nstdout:\n%s", c.String(), err, string(out))
		// Non-ExitErrors or context errors are never expected.
		require.NoError(t, err, msg)
		require.NoError(t, ctx.Err(), msg)
	}
	if expectFailure {
		require.Fail(t, fmt.Sprintf("%q exited with 0:\n%s", c.String(), string(out)))
	}
	return out
}

func testDataDir(t *testing.T) string {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	return filepath.Join(cwd, "testdata")
}

// JUnit XML report format
// See (unofficial source only): https://github.com/windyroad/JUnit-Schema/blob/master/JUnit.xsd
type junitTestSuite struct {
	TestCases []struct {
		Name   string `xml:"name,attr"`
		Status string `xml:"status,attr"`
	} `xml:"testcase"`
}
