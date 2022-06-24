package executil

import (
	"os"
	"os/exec"
	"strconv"

	"github.com/pkg/errors"
)

func (c *Cmd) TerminateProcessGroup() error {
	// Based on https://stackoverflow.com/a/44551450/2804197
	// Original author: https://stackoverflow.com/users/301049/rots
	kill := exec.Command("TASKKILL", "/T", "/F", "/PID", strconv.Itoa(c.Process.Pid))
	kill.Stderr = os.Stderr
	kill.Stdout = os.Stderr
	err := kill.Run()
	return errors.WithStack(err)
}

func (c *Cmd) prepareProcessGroupTermination() {
	// Nothing to prepare on Windows
	return
}

func (c *Cmd) getpgid() (int, error) {
	// We don't need a process group ID on Windows for process group
	// termination
	return 0, nil
}
