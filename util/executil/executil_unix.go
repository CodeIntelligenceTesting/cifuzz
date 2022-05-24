//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package executil

import (
	"syscall"
	"time"

	"code-intelligence.com/cifuzz/pkg/dialog"
	"github.com/pkg/errors"
)

func (c *Cmd) TerminateProcessGroup(pgid int) {
	dialog.Infof("Sending SIGTERM to process group %d", pgid)
	// We ignore errors here because the process group might not exist
	// anymore at this point.
	_ = syscall.Kill(-pgid, syscall.SIGTERM) // note the minus sign

	// Give the process group a few seconds to exit
	select {
	case <-time.After(processGroupTerminationGracePeriod):
		// The process group didn't exit within the grace period, so we
		// send it a SIGKILL now
		dialog.Infof("Sending SIGKILL to process group %d", pgid)
		// We ignore errors here because the process group might not exist
		// anymore at this point.
		_ = syscall.Kill(-pgid, syscall.SIGKILL) // note the minus sign
	case <-c.waitDone:
		// The process has already exited, nothing else to do here.
		// Note: This might leave other processes in the process group
		// running (which ignored the SIGTERM).
	}
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
