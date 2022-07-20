package minijail

import (
	"bytes"
	"io"
	"regexp"
)

var ignoredPattern = regexp.MustCompile(`^libminijail[\d+]: child process \d+ exited with status \d+`)

type OutputFilter struct {
	nextWriter io.Writer
	buf        *bytes.Buffer
}

func NewOutputFilter(nextWriter io.Writer) *OutputFilter {
	return &OutputFilter{nextWriter: nextWriter, buf: bytes.NewBuffer([]byte{})}
}

func (w *OutputFilter) Write(p []byte) (n int, err error) {
	// To be able to match lines, we only print up to the last newline
	// and store everything not printed in the buffer
	index := bytes.LastIndexByte(p, '\n')
	if index == -1 {
		w.buf.Write(p)
		return len(p), nil
	}

	toPrint, toStore := p[:index+1], p[index+1:]

	// Prepend the bytes stored in the buffer to the bytes we're about
	// to print
	toPrint = append(w.buf.Bytes(), toPrint...)

	w.buf.Reset()
	w.buf.Write(toStore)

	if ignoredPattern.Match(toPrint) {
		return len(p), nil
	}

	written, err := w.nextWriter.Write(toPrint)
	numBytesNotWritten := len(toPrint) - written
	return len(p) - numBytesNotWritten, err
}
