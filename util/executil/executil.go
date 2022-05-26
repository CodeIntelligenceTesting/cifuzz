package executil

import (
	"context"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/pkg/errors"

	"code-intelligence.com/cifuzz/pkg/log"
)

const (
	// Duration we wait after sending a SIGTERM to the process group
	// before we send a SIGKILL. When a use case arises for configuring
	// the grace period, we can make this a configurable attribute of
	// Cmd.
	processGroupTerminationGracePeriod = 3 * time.Second
)

// Cmd provides the same functionality as exec.Cmd plus some utility
// methods.
type Cmd struct {
	*exec.Cmd
	ctx            context.Context
	waitDone       chan struct{}
	CloseAfterWait []io.Closer
	pgid           int
	// When TerminateProcessGroupWhenContextDone is set to true,
	// Cmd.Start() will terminate the process group when the command did
	// not complete before the context is done. In that case,
	// TerminatedAfterContextDone() will return true.
	TerminateProcessGroupWhenContextDone bool
	terminatedAfterContextDone           bool
	terminatedAfterContextDoneMutex      sync.Mutex
}

func Command(name string, arg ...string) *Cmd {
	return &Cmd{Cmd: exec.Command(name, arg...)}
}

func CommandContext(ctx context.Context, name string, arg ...string) *Cmd {
	return &Cmd{Cmd: exec.CommandContext(ctx, name, arg...), ctx: ctx}
}

// StdoutTeePipe is similar to StdoutPipe, but everything written to the
// pipe is also copied to stdout (like tee(1) does).
//
// In contrast to StdoutPipe, Wait will *not* automatically close the
// pipe, so it's the caller's responsibility to close the pipe. In effect,
// it is fine to call Wait before all reads from the pipe have completed.
func (c *Cmd) StdoutTeePipe() (io.ReadCloser, error) {
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
	c.Stdout = io.MultiWriter(os.Stdout, pw)
	c.CloseAfterWait = append(c.CloseAfterWait, pw)
	return pr, nil
}

// Same as StdoutTeePipe but for stderr.
func (c *Cmd) StderrTeePipe() (io.ReadCloser, error) {
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
	c.Stderr = io.MultiWriter(os.Stderr, pw)
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
		return err
	}

	if c.TerminateProcessGroupWhenContextDone && c.ctx != nil {
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
				c.TerminateProcessGroup()
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
