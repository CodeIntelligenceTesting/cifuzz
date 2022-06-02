package executil

import (
	"bufio"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"code-intelligence.com/cifuzz/util/fileutil"
)

func TestCmd_StdoutTeePipe_ReadAsync(t *testing.T) {
	// Redirect stdout to a file
	outFile, err := fileutil.TempFile("outFile")
	require.NoError(t, err)
	defer fileutil.Cleanup(outFile.Name())

	os.Stdout = outFile
	cmd := Command("echo", "foo")
	pipe, err := cmd.StdoutTeePipe(os.Stdout)
	require.NoError(t, err)

	err = cmd.Start()
	require.NoError(t, err)

	doneCh := make(chan struct{})
	go func() {
		outFromPipe, err := ioutil.ReadAll(pipe)
		require.NoError(t, err)
		require.Equal(t, "foo", strings.TrimSpace(string(outFromPipe)))
		doneCh <- struct{}{}
	}()

	err = cmd.Wait()
	require.NoError(t, err)

	err = outFile.Close()
	require.NoError(t, err)

	outFromFile, err := ioutil.ReadFile(outFile.Name())
	require.NoError(t, err)
	require.Equal(t, "foo", strings.TrimSpace(string(outFromFile)))

	// Wait until the pipe was read
	<-doneCh

	err = pipe.Close()
	require.NoError(t, err)
}

func TestCmd_StdoutTeePipe_ReadAsyncMultiline(t *testing.T) {
	var err error
	// Don't spam stdout with the output of yes(1)
	os.Stdout, err = os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	require.NoError(t, err)

	cmd := Command("yes")
	pipe, err := cmd.StdoutTeePipe(os.Stdout)
	require.NoError(t, err)

	err = cmd.Start()
	require.NoError(t, err)

	doneCh := make(chan struct{})
	go func() {
		scanner := bufio.NewScanner(pipe)
		scanner.Scan()
		require.Equal(t, "y", scanner.Text())
		doneCh <- struct{}{}
	}()

	// Wait until the pipe was read
	<-doneCh

	err = pipe.Close()
	require.NoError(t, err)
}

func TestCmd_StdoutTeePipe_ReadSync(t *testing.T) {
	// Redirect stdout to a file
	outFile, err := fileutil.TempFile("outFile")
	require.NoError(t, err)
	defer fileutil.Cleanup(outFile.Name())

	os.Stdout = outFile
	cmd := Command("echo", "foo")
	pipe, err := cmd.StdoutTeePipe(os.Stdout)
	require.NoError(t, err)

	err = cmd.Start()
	require.NoError(t, err)

	err = cmd.Wait()
	require.NoError(t, err)

	outFromPipe, err := ioutil.ReadAll(pipe)
	require.NoError(t, err)
	require.Equal(t, "foo", strings.TrimSpace(string(outFromPipe)))

	err = pipe.Close()
	require.NoError(t, err)

	err = outFile.Close()
	require.NoError(t, err)

	outFromFile, err := ioutil.ReadFile(outFile.Name())
	require.NoError(t, err)
	require.Equal(t, "foo", strings.TrimSpace(string(outFromFile)))
}

func TestCmd_StdoutTeePipe_ReadSyncWithRun(t *testing.T) {
	// Redirect stdout to a file
	outFile, err := fileutil.TempFile("outFile")
	require.NoError(t, err)
	defer fileutil.Cleanup(outFile.Name())

	os.Stdout = outFile
	cmd := Command("echo", "foo")
	pipe, err := cmd.StdoutTeePipe(os.Stdout)
	require.NoError(t, err)

	err = cmd.Run()
	require.NoError(t, err)

	outFromPipe, err := ioutil.ReadAll(pipe)
	require.NoError(t, err)
	require.Equal(t, "foo", strings.TrimSpace(string(outFromPipe)))

	err = pipe.Close()
	require.NoError(t, err)

	err = outFile.Close()
	require.NoError(t, err)

	outFromFile, err := ioutil.ReadFile(outFile.Name())
	require.NoError(t, err)
	require.Equal(t, "foo", strings.TrimSpace(string(outFromFile)))
}

func TestCmd_StdoutTeePipe_NoRead(t *testing.T) {
	// Redirect stdout to a file
	outFile, err := fileutil.TempFile("outFile")
	require.NoError(t, err)
	defer fileutil.Cleanup(outFile.Name())

	os.Stdout = outFile
	cmd := Command("echo", "foo")
	pipe, err := cmd.StdoutTeePipe(os.Stdout)
	require.NoError(t, err)

	err = cmd.Start()
	require.NoError(t, err)

	err = cmd.Wait()
	require.NoError(t, err)

	err = outFile.Close()
	require.NoError(t, err)

	outFromFile, err := ioutil.ReadFile(outFile.Name())
	require.NoError(t, err)
	require.Equal(t, "foo", strings.TrimSpace(string(outFromFile)))

	err = pipe.Close()
	require.NoError(t, err)
}

func TestCmd_StdoutTeePipe_NoReadWithRun(t *testing.T) {
	// Redirect stdout to a file
	outFile, err := fileutil.TempFile("outFile")
	require.NoError(t, err)
	defer fileutil.Cleanup(outFile.Name())

	os.Stdout = outFile
	cmd := Command("echo", "foo")
	pipe, err := cmd.StdoutTeePipe(os.Stdout)
	require.NoError(t, err)

	err = cmd.Run()
	require.NoError(t, err)

	err = outFile.Close()
	require.NoError(t, err)

	outFromFile, err := ioutil.ReadFile(outFile.Name())
	require.NoError(t, err)
	require.Equal(t, "foo", strings.TrimSpace(string(outFromFile)))

	err = pipe.Close()
	require.NoError(t, err)
}
