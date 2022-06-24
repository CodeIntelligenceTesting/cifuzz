//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package executil

import (
	"syscall"
	"time"

	"github.com/pkg/errors"

	"code-intelligence.com/cifuzz/pkg/log"
)

func (c *Cmd) TerminateProcessGroup() error {
	if c.getpgidError != nil {
		return errors.WithMessage(c.getpgidError, "Can't terminate process group")
	}

	log.Infof("Sending SIGTERM to process group %d", c.pgid)
	// We ignore errors here because the process group might not exist
	// anymore at this point.
	err := syscall.Kill(-c.pgid, syscall.SIGTERM) // note the minus sign
	if err != nil {
		return errors.WithStack(err)
	}

	// Give the process group a few seconds to exit
	select {
	case <-time.After(processGroupTerminationGracePeriod):
		// The process group didn't exit within the grace period, so we
		// send it a SIGKILL now
		log.Infof("Sending SIGKILL to process group %d", c.pgid)
		// We ignore errors here because the process group might not exist
		// anymore at this point.
		err = syscall.Kill(-c.pgid, syscall.SIGKILL) // note the minus sign
		if err != nil {
			return errors.WithStack(err)
		}
	case <-c.waitDone:
		// The process has already exited, nothing else to do here.
		// Note: This might leave other processes in the process group
		// running (which ignored the SIGTERM).
	}

	return nil
}

func (c *Cmd) prepareProcessGroupTermination() {
	// Set PGID so that we're able to terminate the process group on timeout
	if c.SysProcAttr == nil {
		c.SysProcAttr = &syscall.SysProcAttr{}
	}
	c.SysProcAttr.Setpgid = true
}

func (c *Cmd) getpgid() (int, error) {
	pgid, err := syscall.Getpgid(c.Process.Pid)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	return pgid, nil
}
