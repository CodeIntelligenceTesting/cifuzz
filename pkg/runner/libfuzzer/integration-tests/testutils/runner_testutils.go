package testutils

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"code-intelligence.com/cifuzz/internal/config"
	"code-intelligence.com/cifuzz/pkg/report"
	"code-intelligence.com/cifuzz/pkg/runner/libfuzzer"
	"code-intelligence.com/cifuzz/util/stringutil"
)

// ChannelPassthrough pipes the reports from the runner package to a report channel
// and also prints them to stdout
type ChannelPassthrough struct {
	ch chan *report.Report
}

func (cp *ChannelPassthrough) Handle(report *report.Report) error {
	select {
	case cp.ch <- report:
		jsonString, err := stringutil.ToJsonString(report)
		if err != nil {
			return err
		}
		fmt.Println(jsonString)
	default:
	}
	return nil
}

// RunnerTest helps to execute tests for the runner package
type RunnerTest struct {
	FuzzTarget      string
	ExecutionDir    string
	Engine          config.Engine
	SeedCorpusDir   string
	Timeout         time.Duration
	EngineArgs      []string
	FuzzTargetArgs  []string
	FuzzerEnv       []string
	DisableMinijail bool
	RunsLimit       int
}

func NewLibfuzzerTest(t *testing.T, fuzzTarget string, disableMinijail bool) *RunnerTest {
	return &RunnerTest{
		ExecutionDir: GetFuzzTargetBuildDir(t),
		FuzzTarget:   GetFuzzTargetPath(t, fuzzTarget),
		Engine:       config.LIBFUZZER,
		// Use a deterministic random seed
		EngineArgs: []string{
			"-seed=1",
		},
		DisableMinijail: disableMinijail,
		// For those tests which don't set a custom runs limit, the
		// expected errors are found within 3000 runs.
		RunsLimit: 3000,
	}

}

// Start selects the needed runner and execute it with the given options
func (test *RunnerTest) Start(t *testing.T, reportCh chan *report.Report) error {
	var err error

	if test.SeedCorpusDir == "" {
		test.SeedCorpusDir, err = ioutil.TempDir("", "seeds")
		require.NoError(t, err)
	}

	additionalSeedDir, err := ioutil.TempDir("", "additional_seeds")
	require.NoError(t, err)

	if test.RunsLimit != -1 {
		// Limit the number of runs
		test.EngineArgs = append(test.EngineArgs, fmt.Sprintf("-runs=%d", test.RunsLimit))
	}

	libfuzzerOptions := &libfuzzer.RunnerOptions{
		FuzzTarget:          test.FuzzTarget,
		SeedsDir:            test.SeedCorpusDir,
		AdditionalSeedsDirs: []string{additionalSeedDir},
		Timeout:             test.Timeout,
		EngineArgs:          test.EngineArgs,
		FuzzTargetArgs:      test.FuzzTargetArgs,
		EnvVars:             test.FuzzerEnv,
		UseMinijail:         !test.DisableMinijail,
		ReportHandler:       &ChannelPassthrough{ch: reportCh},
		Verbose:             true,
	}
	defer close(reportCh)

	if test.Engine == config.LIBFUZZER {
		libfuzzerRunner := libfuzzer.NewRunner(libfuzzerOptions)
		return libfuzzerRunner.Run(context.Background())
	}

	return fmt.Errorf("unknown fuzzing engine for test execution")
}

// Run makes sure that all the test output gets captured
func (test *RunnerTest) Run(t *testing.T) (string, string, []*report.Report) {

	// change working directory to keep a clean state
	err := os.Chdir(test.ExecutionDir)
	require.NoError(t, err)

	rStdout, wStdout, err := os.Pipe()
	require.NoError(t, err)
	rStderr, wStderr, err := os.Pipe()
	require.NoError(t, err)

	origStdout := os.Stdout
	origStderr := os.Stderr
	os.Stdout = wStdout
	os.Stderr = wStderr

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	stdoutClosed := make(chan struct{})
	stderrClosed := make(chan struct{})

	go func() {
		_, _ = io.Copy(io.MultiWriter(origStdout, stdout), rStdout)
		stdoutClosed <- struct{}{}
	}()

	go func() {
		_, _ = io.Copy(io.MultiWriter(origStderr, stderr), rStderr)
		stderrClosed <- struct{}{}
	}()

	// create buffered channel for receiving the reports
	reportCh := make(chan *report.Report, 1024)

	err = test.Start(t, reportCh)
	require.NoError(t, err)

	// collecting reports
	reports := []*report.Report{}
	for report := range reportCh {
		reports = append(reports, report)
	}

	os.Stdout = origStdout
	os.Stderr = origStderr
	wStdout.Close()
	wStderr.Close()

	rStdout.Close()
	rStderr.Close()

	<-stdoutClosed
	<-stderrClosed

	return stdout.String(), stderr.String(), reports
}

func (test *RunnerTest) RequireSeedCorpusNotEmpty(t *testing.T) {
	seeds, err := os.ReadDir(test.SeedCorpusDir)
	require.NoError(t, err)
	require.NotEmpty(t, seeds, "corpus directory is empty: %s", test.SeedCorpusDir)
}
