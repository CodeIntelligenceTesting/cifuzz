package log

import (
	"fmt"
	"io"
	"sync"

	"github.com/pkg/errors"
)

var ActiveUpdatingPrinter updatingPrinter

type updatingPrinter interface {
	Clear()
}

var writeLock sync.Mutex

type ptermWriter struct {
	buf []byte
	out io.Writer
}

// NewPTermWriter returns a writer which ensures that the output written
// by it doesn't mess with the output of an active pterm.SpinnerPrinter.
func NewPTermWriter(out io.Writer) *ptermWriter {
	return &ptermWriter{out: out}
}

func (w *ptermWriter) Write(p []byte) (n int, err error) {
	// To avoid races, only one write is executed at a time
	writeLock.Lock()
	defer writeLock.Unlock()

	// To ensure that after the write, the spinner printer has a new
	// line for itself, we only write the output if it ends with a
	// newline. Else, we store it in a buffer which we write the next
	// time Write() is called with something that ends with a newline.
	lenOldBuf := len(w.buf)
	w.buf = append(w.buf, p...)
	if len(p) == 0 || p[len(p)-1] != '\n' {
		return len(p), nil
	}

	// Clear the updating printer output if any. We don't use
	// pterm.Fprint here, which also tries to clear spinner printer
	// output, because that only works when the spinner printer and this
	// function write to the same output stream, which is not always the
	// case.
	if ActiveUpdatingPrinter != nil {
		ActiveUpdatingPrinter.Clear()
	}

	// Write the buffer
	n, err = fmt.Fprint(w.out, string(w.buf))

	// Clear the buffer now that it was written
	w.buf = []byte{}

	// Return the number of bytes from p that were written. If
	// fmt.Fprint printed all bytes from the buffer, this is n minus the
	// length of the buffer before we appended p to it.
	return n - lenOldBuf, errors.WithStack(err)
}
