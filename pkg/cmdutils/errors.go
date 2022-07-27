package cmdutils

import (
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/pkg/errors"
)

var ErrSilent = WrapSilentError(errors.New("SilentError"))

// SilentError indicates that the error message should not be printed
// when the error is handled.
type SilentError struct {
	err error
}

func (e SilentError) Error() string {
	return e.err.Error()
}

func (e SilentError) Unwrap() error {
	return e.err
}

// WrapSilentError wraps an existing error into a SilentError to avoid
// having the error message printed when the error is handled.
func WrapSilentError(err error) error {
	return &SilentError{err}
}

// IncorrectUsageError indicates that the command wasn't used correctly,
// for example because required arguments are missing.
// When an IncorrectUsageError is handled, the usage message should be
// printed.
type IncorrectUsageError struct {
	err error
}

func (e IncorrectUsageError) Error() string {
	return e.err.Error()
}

func (e IncorrectUsageError) Unwrap() error {
	return e.err
}

// WrapIncorrectUsageError wraps an existing error into a
// IncorrectUsageError to have the usage message printed when the error
// is handled.
func WrapIncorrectUsageError(err error) error {
	return &IncorrectUsageError{err}
}

func NewSignalError(signal syscall.Signal) *SignalError {
	return &SignalError{signal}
}

type SignalError struct {
	Signal syscall.Signal
}

func (e SignalError) Error() string {
	return fmt.Sprintf("terminated by signal %d (%s)", int(e.Signal), e.Signal.String())
}

// CouldBeSandboxError indicates that the error might have been caused
// by the sandbox restricting access. When a CouldBeSandboxError is
// handled, a message should be printed which suggests to disable the
// sandbox if its not needed.
type CouldBeSandboxError struct {
	err error
}

func (e CouldBeSandboxError) Error() string {
	return e.err.Error()
}

func (e CouldBeSandboxError) Unwrap() error {
	return e.err
}

// WrapCouldBeSandboxError wraps an existing error into a
// CouldBeSandboxError to hint on disabling the sandbox when the error
// is handled.
func WrapCouldBeSandboxError(err error) error {
	return &CouldBeSandboxError{err}
}

// ExecError includes information about the exec.Cmd which failed in the
// error message.
type ExecError struct {
	err error
	cmd *exec.Cmd
}

func (e *ExecError) msg() string {
	var exitErr *exec.ExitError
	if errors.As(e.err, &exitErr) {
		stderr := string(exitErr.Stderr)
		if stderr != "" && !strings.HasSuffix(stderr, "\n") {
			stderr += "\n"
		}
		return fmt.Sprintf("%s%s", stderr, filepath.Base(e.cmd.Args[0]))
	}
	return ""
}

func (e *ExecError) Error() string {
	return fmt.Sprintf("%s: %s\n", e.msg(), e.err.Error())
}

func (e *ExecError) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			_, _ = fmt.Fprintf(s, "%s: %+v\n", e.msg(), e.err)
			return
		}
		fallthrough
	case 's', 'q':
		_, _ = io.WriteString(s, e.Error())
	}
}

func (e *ExecError) Unwrap() error {
	return e.err
}

// WrapExecError wraps an existing error into an ExecError to include
// information about the exec.Cmd which failed in the error message.
func WrapExecError(err error, cmd *exec.Cmd) error {
	return &ExecError{err, cmd}
}
