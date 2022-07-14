package libfuzzer

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"code-intelligence.com/cifuzz/pkg/log"
	"code-intelligence.com/cifuzz/pkg/minijail"
	libfuzzer_parser "code-intelligence.com/cifuzz/pkg/parser/libfuzzer"
	"code-intelligence.com/cifuzz/pkg/report"
	fuzzer_runner "code-intelligence.com/cifuzz/pkg/runner"
	"code-intelligence.com/cifuzz/util/envutil"
	"code-intelligence.com/cifuzz/util/executil"
	"code-intelligence.com/cifuzz/util/sliceutil"
	"code-intelligence.com/cifuzz/util/stringutil"
)

const (
	MaxBufferedReports = 10
	sendTimeout        = time.Second * 10
	// ExitGracePeriod is time we give libfuzzer to exit after
	// max_total_time was exceeded.
	// Must be more than 2 seconds, because in the CI it happened that
	// libfuzzer did not exit within 2 seconds.
	ExitGracePeriod = time.Second * 5
)

type RunnerOptions struct {
	FuzzTarget         string
	GeneratedCorpusDir string
	SeedCorpusDirs     []string
	Dictionary         string
	LibraryDirs        []string
	EnvVars            []string
	EngineArgs         []string
	FuzzTestArgs       []string
	ReportHandler      report.Handler
	Timeout            time.Duration
	UseMinijail        bool
	Verbose            bool
	KeepColor          bool
	LogOutput          io.Writer
}

func (options *RunnerOptions) ValidateOptions() error {
	if options.UseMinijail {
		if runtime.GOOS != "linux" {
			return errors.Errorf("Minijail is only supported on Linux")
		}

		// To be able to make the fuzz target accessible to minijail,
		// its path must be absolute and all symlinks must be resolved.
		var err error
		options.FuzzTarget, err = filepath.EvalSymlinks(options.FuzzTarget)
		if err != nil {
			return errors.WithStack(err)
		}
		options.FuzzTarget, err = filepath.Abs(options.FuzzTarget)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	if options.LogOutput == nil {
		options.LogOutput = os.Stderr
	}

	return nil
}

type Runner struct {
	*RunnerOptions
	SupportJazzer bool

	cmd *executil.Cmd
}

func NewRunner(options *RunnerOptions) *Runner {
	return &Runner{RunnerOptions: options, SupportJazzer: false}
}

func (r *Runner) Run(ctx context.Context) error {
	err := r.ValidateOptions()
	if err != nil {
		return err
	}

	args := []string{r.FuzzTarget}

	// Tell libfuzzer to exit after the timeout
	timeoutSeconds := strconv.FormatInt(int64(r.Timeout.Seconds()), 10)
	args = append(args, "-max_total_time="+timeoutSeconds)

	// Tell libfuzzer which dictionary it should use
	if r.Dictionary != "" {
		args = append(args, "-dict="+r.Dictionary)
	}

	// Add user-specified libfuzzer options
	args = append(args, r.EngineArgs...)

	// Tell libfuzzer which corpus directory it should use
	args = append(args, r.GeneratedCorpusDir)

	// Add any seed corpus directories as further positional arguments
	args = append(args, r.SeedCorpusDirs...)

	if len(r.FuzzTestArgs) > 0 {
		// separate the libfuzzer and fuzz test arguments with a "--"
		args = append(args, "--")
		args = append(args, r.FuzzTestArgs...)
	}

	// The environment to run libfuzzer in
	fuzzerEnv, err := r.FuzzerEnvironment()
	if err != nil {
		return err
	}

	// The environment we run minijail in
	wrapperEnv := os.Environ()

	if r.UseMinijail {
		libfuzzerArgs := args

		// Make libfuzzer create artifacts (e.g. crash files) in the
		// minijail output directory.
		libfuzzerArgs = append(libfuzzerArgs, "-artifact_prefix="+minijail.OutputDir+"/")

		bindings := []*minijail.Binding{
			// The fuzz target must be accessible
			{Source: r.FuzzTarget},
			// The first corpus directory must be writable, because
			// libfuzzer writes new test inputs to it
			{Source: r.GeneratedCorpusDir, Writable: minijail.ReadWrite},
		}

		for _, dir := range r.SeedCorpusDirs {
			bindings = append(bindings, &minijail.Binding{Source: dir})
		}

		// Set up Minijail
		mj, err := minijail.NewMinijail(&minijail.Options{
			Args:     libfuzzerArgs,
			Bindings: bindings,
			Env:      fuzzerEnv,
		})
		if err != nil {
			return err
		}
		defer mj.Cleanup()

		// Use the command which runs libfuzzer via minijail
		args = mj.Args
	} else {
		// We don't use minijail, so we can set the environment
		// variables for the fuzzer in the wrapper environment
		for key, value := range envutil.ToMap(fuzzerEnv) {
			wrapperEnv, err = envutil.Setenv(wrapperEnv, key, value)
			if err != nil {
				return err
			}
		}
	}

	return r.RunLibfuzzerAndReport(ctx, args, wrapperEnv)
}

func (r *Runner) RunLibfuzzerAndReport(ctx context.Context, args []string, env []string) error {
	var err error

	// Ideally, libfuzzer exits on its own after the timeout, because we
	// specified `-max_total_time` above. For the case that it does not,
	// we still set up a timeout handler here which sends a SIGTERM and
	// later a SIGKILL if it still didn't exit.
	// To give libfuzzer some time to exit on its own, we send the
	// SIGTERM a bit later than the timeout specified via `-max_total_time`.
	var cmdCtx context.Context
	var cancelCmdCtx context.CancelFunc
	if r.Timeout > 0 {
		terminateTimeout := r.Timeout + ExitGracePeriod
		cmdCtx, cancelCmdCtx = context.WithTimeout(ctx, terminateTimeout)
	} else {
		// No timeout
		cmdCtx, cancelCmdCtx = context.WithCancel(ctx)
	}
	defer cancelCmdCtx()
	r.cmd = executil.CommandContext(cmdCtx, args[0], args[1:]...)
	r.cmd.Env = env

	var stderrPipe io.ReadCloser
	if r.Verbose {
		// Print the command's stdout and stderr via pterm to avoid that
		// the output messes with the pterm output or gets overwritten
		// by it.
		// Note that this causes the command's stdout to be printed to
		// stderr, which is what we want, because we only want reports
		// printed to stdout.
		ptermWriter := log.NewPTermWriter(r.LogOutput)
		r.cmd.Stdout = ptermWriter

		// Write the command's stderr to both a pipe and the pterm
		// writer which prints it to stderr, so that we can parse the
		// output but still allow the caller to observe the status and
		// progress in realtime.
		stderrPipe, err = r.cmd.StderrTeePipe(ptermWriter)
		if err != nil {
			return err
		}
	} else {
		stderrPipe, err = r.cmd.StderrPipe()
		if err != nil {
			return err
		}
	}

	log.Debugf("Command: %s", strings.Join(stringutil.QuotedStrings(r.cmd.Args), " "))
	err = r.cmd.Start()
	if err != nil {
		return err
	}

	reporter := libfuzzer_parser.NewLibfuzzerOutputParser(&libfuzzer_parser.Options{
		SupportJazzer: r.SupportJazzer,
		KeepColor:     r.KeepColor,
	})
	reportsCh := make(chan *report.Report, MaxBufferedReports)

	// Start a go routine which waits for the command to exit and
	// continuously parses the output
	routines, routinesCtx := errgroup.WithContext(ctx)
	routines.Go(func() error {
		waitErrCh := make(chan error)

		// Wait for the command to exit in a go routine, so that below
		// we can cancel waiting when the context is done
		go func() {
			waitErrCh <- r.cmd.Wait()
		}()

		// Wait until the reporter has finished parsing stderr, so that
		// we can check below whether the reporter has found something
		err := reporter.Parse(routinesCtx, stderrPipe, reportsCh)
		if err != nil {
			return err
		}

		select {
		case err := <-waitErrCh:
			if r.cmd.TerminatedAfterContextDone() {
				// The command was terminated because the timeout exceeded. We
				// don't return an error in that case.
				return nil
			}

			// If err is not an ExitError, something unexpected happened
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) {
				return err
			}

			if !IsExpectedExitError(err) {
				// Print the stderr output of the fuzzer up to the point where
				// it has been successfully initialized to provide users with
				// the context of this abnormal exit even without verbose mode.
				if !r.Verbose {
					log.Print(reporter.StartupOutput())
				}
				return errors.WithMessagef(err, "Unexpected exit code %d", exitErr.ExitCode())
			}

			if !reporter.FindingReported {
				return errors.WithMessagef(err, "libFuzzer exited with expected exit code %d but no finding was reported", exitErr.ExitCode())
			}

			// libFuzzer found an error and exited with an expected error
			// code. We don't want to return an error in that case.
			return nil
		case <-routinesCtx.Done():
			return routinesCtx.Err()
		}
	})

	// Continuously send reports from the reports channel to the
	// receiver. By doing this in a separate go routine and using a
	// buffered reports channel, this allows the LibfuzzerOutputParser
	// function above to not block when creating a report (as long as
	// the buffer is not full).
	routines.Go(func() error {
		senderErrCh := make(chan error, 1)

		go func() {
			senderErrCh <- sendReports(r.ReportHandler, reportsCh)
		}()

		select {
		case err := <-senderErrCh:
			return err
		case <-routinesCtx.Done():
			// The routines context got cancelled, so either the
			// command failed or the reporter encountered an error.
			// We give the sender a few seconds to send pending reports.
			select {
			case err := <-senderErrCh:
				return err
			case <-time.After(sendTimeout):
				return errors.Errorf("Sending reports timed out (%s)", sendTimeout)
			}
		}
	})

	return routines.Wait()
}

func (r *Runner) FuzzerEnvironment() ([]string, error) {
	env, err := fuzzer_runner.FuzzerEnvironment()
	if err != nil {
		return nil, err
	}

	env, err = fuzzer_runner.SetLDLibraryPath(env, r.LibraryDirs)
	if err != nil {
		return nil, err
	}

	// Add the user-specified environment variables. We do this after
	// setting our defaults but before setting sanitizer options,
	// because there we  take care of overriding options which we need
	// to override and keeping other options.
	env, err = fuzzer_runner.AddEnvFlags(env, r.EnvVars)
	if err != nil {
		return nil, err
	}

	env, err = fuzzer_runner.SetCommonUBSANOptions(env)
	if err != nil {
		return nil, err
	}

	env, err = fuzzer_runner.SetCommonASANOptions(env)
	if err != nil {
		return nil, err
	}

	overrideOptions := map[string]string{
		// Per default this is set to false, except for darwin.
		// To have consistent behaviour on all supported operating systems
		// we are setting this explicitly to false
		"abort_on_error": "0",
	}
	env, err = fuzzer_runner.SetASANOptions(env, nil, overrideOptions)
	if err != nil {
		return nil, err
	}

	return env, nil
}

func (r *Runner) Cleanup() {
	if r.cmd != nil {
		err := r.cmd.TerminateProcessGroup()
		if err != nil {
			log.Error(err, err.Error())
		}
	}
}

func sendReports(handler report.Handler, reportsCh <-chan *report.Report) error {
	for r := range reportsCh {
		err := handler.Handle(r)
		if err != nil {
			return err
		}
	}
	return nil
}

func IsExpectedExitError(err error) bool {
	expectedExitCodes := []int{
		fuzzer_runner.SanitizerErrorExitCode,
		fuzzer_runner.LibFuzzerErrorExitCode,
		fuzzer_runner.LibFuzzerOOMExitCode,
		fuzzer_runner.LibFuzzerTimeoutExitCode,
	}

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return false
	}
	return sliceutil.Contains(expectedExitCodes, exitErr.ExitCode())
}
