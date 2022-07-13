package executil

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"code-intelligence.com/cifuzz/pkg/cmdutils"
	"code-intelligence.com/cifuzz/pkg/log"
)

const (
	// Duration we wait after sending a SIGTERM to the process group
	// before we send a SIGKILL. When a use case arises for configuring
	// the grace period, we can make this a configurable attribute of
	// Cmd.
	processGroupTerminationGracePeriod = 3 * time.Second
)

// HandleExecError wraps an error returned by a function on exec.Cmd, adding
// stderr to the message if the error is an exec.ExitError.
// Note: exec.ExitError instances are logged without the stack trace and
// returned as a cmdutils.SilentError.
func HandleExecError(cmd *exec.Cmd, err error) error {
	if exitErr, ok := err.(*exec.ExitError); ok {
		stderr := string(exitErr.Stderr)
		if stderr != "" && !strings.HasSuffix(stderr, "\n") {
			stderr += "\n"
		}
		err = fmt.Errorf("%s%s: %w", stderr, cmd.Args[0], err)
		log.Error(err, err.Error())
		return cmdutils.WrapSilentError(errors.WithStack(err))
	}
	return errors.WithStack(err)
}

// Cmd provides the same functionality as exec.Cmd plus some utility
// methods.
type Cmd struct {
	*exec.Cmd
	ctx                             context.Context
	waitDone                        chan struct{}
	CloseAfterWait                  []io.Closer
	pgid                            int
	getpgidError                    error
	terminatedAfterContextDone      bool
	terminatedAfterContextDoneMutex sync.Mutex
}

func Command(name string, arg ...string) *Cmd {
	return &Cmd{Cmd: exec.Command(name, arg...)}
}

// CommandContext is like Command but includes a context.
//
// The provided context is used to terminate the process group (by first
// sending SIGTERM to the process group and after a grace period of 3
// seconds SIGKILL) if the context becomes done before the command
// completes on its own.
// In that case, Cmd.TerminatedAfterContextDone() returns true.
func CommandContext(ctx context.Context, name string, arg ...string) *Cmd {
	// We don't use exec.CommandContext here to avoid a race between
	// the goroutine started by the exec package when
	// exec.CommandContext is used, which immediately sends SIGKILL to
	// the started process when the context is done, and the goroutine
	// started by this package which first sends a SIGTERM to the
	// process group and after a grace period a SIGKILL.
	return &Cmd{Cmd: exec.Command(name, arg...), ctx: ctx}
}

// StdoutTeePipe is similar to StdoutPipe, but everything written to the
// pipe is also copied to the specified writer (similar to tee(1)).
//
// In contrast to StdoutPipe, Wait will *not* automatically close the
// pipe, so it's the caller's responsibility to close the pipe. In effect,
// it is fine to call Wait before all reads from the pipe have completed.
func (c *Cmd) StdoutTeePipe(writer io.Writer) (io.ReadCloser, error) {
	if c.Stdout != nil {
		return nil, errors.New("exec: Stdout already set")
	}
	if c.Process != nil {
		return nil, errors.New("exec: StdoutTeePipe after process started")
	}
	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	c.Stdout = io.MultiWriter(writer, pw)
	c.CloseAfterWait = append(c.CloseAfterWait, pw)
	return pr, nil
}

// Same as StdoutTeePipe but for stderr.
func (c *Cmd) StderrTeePipe(writer io.Writer) (io.ReadCloser, error) {
	if c.Stderr != nil {
		return nil, errors.New("exec: Stderr already set")
	}
	if c.Process != nil {
		return nil, errors.New("exec: StderrTeePipe after process started")
	}
	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	c.Stderr = io.MultiWriter(writer, pw)
	c.CloseAfterWait = append(c.CloseAfterWait, pw)
	return pr, nil
}

// Does the same as exec.Cmd.Start(), but also closes the write ends of
// tee pipes (if there are any) on the same errors that exec.Cmd.Start()
// closes its open pipes.
func (c *Cmd) Start() error {
	if c.Process != nil {
		return errors.New("exec: already started")
	}

	// Don't start the process if the context is already done
	if c.ctx != nil {
		select {
		case <-c.ctx.Done():
			c.closeDescriptors(c.CloseAfterWait)
			return c.ctx.Err()
		default:
		}
	}

	c.prepareProcessGroupTermination()

	err := c.Cmd.Start()
	if err != nil {
		c.closeDescriptors(c.CloseAfterWait)
		return errors.WithStack(err)
	}

	// Get the process group ID which is needed when
	// c.TerminateProcessGroup() is called.
	c.pgid, err = c.getpgid()
	if err != nil {
		log.Debugf("Couldn't get process group ID: %+v", err)
		c.getpgidError = err
	}

	if c.ctx != nil {
		c.waitDone = make(chan struct{}, 1)
		go func() {
			select {
			case <-c.ctx.Done():
				c.terminatedAfterContextDoneMutex.Lock()
				// Print the reason for the context being done
				log.Infof("Terminating process: %s", c.ctx.Err().Error())
				// In contrast to exec.Cmd.Start(), we terminate the
				// whole process group here with a grace period instead
				// of calling c.Process.Kill().
				err = c.TerminateProcessGroup()
				if err != nil {
					log.Error(err, err.Error())
				}
				c.terminatedAfterContextDone = true
				c.terminatedAfterContextDoneMutex.Unlock()
				context.Background().Done()
			case <-c.waitDone:
			}
		}()
	}

	return nil
}

func (c *Cmd) TerminatedAfterContextDone() bool {
	c.terminatedAfterContextDoneMutex.Lock()
	res := c.terminatedAfterContextDone
	c.terminatedAfterContextDoneMutex.Unlock()
	return res
}

// Does the same as exec.Cmd.Wait() but also closes the write ends of
// tee pipes (if there are any).
func (c *Cmd) Wait() error {
	defer c.closeDescriptors(c.CloseAfterWait)

	err := c.Cmd.Wait()
	if c.waitDone != nil {
		close(c.waitDone)
	}
	return errors.WithStack(err)
}

// Same as exec.Cmd.Run() but uses the wrapper methods of this struct.
func (c *Cmd) Run() error {
	err := c.Start()
	if err != nil {
		return err
	}

	return c.Wait()
}

func (c *Cmd) closeDescriptors(closers []io.Closer) {
	for _, fd := range closers {
		_ = fd.Close()
	}
}
