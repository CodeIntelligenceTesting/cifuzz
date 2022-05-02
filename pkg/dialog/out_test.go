package dialog

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

var orgStdout *os.File
var orgStderr *os.File

func TestMain(m *testing.M) {
	orgStdout = os.Stdout
	orgStderr = os.Stderr

	viper.Set("verbose", false)
	m.Run()
	viper.Set("verbose", false)

	os.Stderr = orgStderr
	os.Stdout = orgStdout

}

func redirectOutput(t *testing.T) (*os.File, *os.File, *os.File, *os.File) {
	t.Helper()

	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr
	return rOut, wOut, rErr, wErr
}

func restoreStdout(t *testing.T, rOut, wOut, rErr, wErr *os.File) (string, string) {
	t.Helper()

	wOut.Close()
	wErr.Close()
	defer rOut.Close()
	defer rOut.Close()

	stdout, _ := ioutil.ReadAll(rOut)
	stderr, _ := ioutil.ReadAll(rErr)

	return string(stdout), string(stderr)
}

func TestDebugF_NoVerbose(t *testing.T) {
	rOut, wOut, rErr, wErr := redirectOutput(t)

	DebugF("Test")

	stdout, stderr := restoreStdout(t, rOut, wOut, rErr, wErr)

	assert.Empty(t, stderr)
	assert.Empty(t, stdout)
}

func TestDebugF_Verbose(t *testing.T) {
	rOut, wOut, rErr, wErr := redirectOutput(t)

	viper.Set("verbose", true)
	DebugF("Test")
	viper.Set("verbose", false)

	stdout, stderr := restoreStdout(t, rOut, wOut, rErr, wErr)

	assert.Contains(t, stderr, "Test")
	assert.Empty(t, stdout)
}

func TestDebug(t *testing.T) {
	rOut, wOut, rErr, wErr := redirectOutput(t)

	viper.Set("verbose", true)
	Debug("Test")
	viper.Set("verbose", false)

	_, stderr := restoreStdout(t, rOut, wOut, rErr, wErr)
	assert.Contains(t, stderr, "Test\n")
}

func TestErrorF_Verbose(t *testing.T) {
	rOut, wOut, rErr, wErr := redirectOutput(t)

	viper.Set("verbose", true)
	ErrorF(errors.New("test-error"), "Test")
	viper.Set("verbose", false)

	stdout, stderr := restoreStdout(t, rOut, wOut, rErr, wErr)

	assert.Contains(t, stderr, "Test")
	assert.Contains(t, stderr, "test-error")
	assert.Empty(t, stdout)
}

func TestErrorF_NoVerbose(t *testing.T) {
	rOut, wOut, rErr, wErr := redirectOutput(t)

	ErrorF(errors.New("test-error"), "Test")

	stdout, stderr := restoreStdout(t, rOut, wOut, rErr, wErr)

	assert.Contains(t, stderr, "Test")
	assert.NotContains(t, stderr, "test-error")
	assert.Empty(t, stdout)
}

func TestError(t *testing.T) {
	rOut, wOut, rErr, wErr := redirectOutput(t)

	Error(errors.New("test-error"), "Test")

	stdout, stderr := restoreStdout(t, rOut, wOut, rErr, wErr)

	assert.Contains(t, stderr, "Test\n")
	assert.Empty(t, stdout)
}

func TestSuccessF(t *testing.T) {
	rOut, wOut, rErr, wErr := redirectOutput(t)

	SuccessF("Test")

	stdout, stderr := restoreStdout(t, rOut, wOut, rErr, wErr)
	assert.Contains(t, stdout, "Test")
	assert.Empty(t, stderr)
}

func TestSuccess(t *testing.T) {
	rOut, wOut, rErr, wErr := redirectOutput(t)

	Success("Test")

	stdout, stderr := restoreStdout(t, rOut, wOut, rErr, wErr)
	assert.Contains(t, stdout, "Test\n")
	assert.Empty(t, stderr)
}

func TestInfoF(t *testing.T) {
	rOut, wOut, rErr, wErr := redirectOutput(t)

	InfoF("Test")

	stdout, stderr := restoreStdout(t, rOut, wOut, rErr, wErr)
	assert.Contains(t, stdout, "Test")
	assert.Empty(t, stderr)
}

func TestInfo(t *testing.T) {
	rOut, wOut, rErr, wErr := redirectOutput(t)

	Info("Test")

	stdout, stderr := restoreStdout(t, rOut, wOut, rErr, wErr)
	assert.Contains(t, stdout, "Test\n")
	assert.Empty(t, stderr)
}

func TestWarnF(t *testing.T) {
	rOut, wOut, rErr, wErr := redirectOutput(t)

	WarnF("Test")

	stdout, stderr := restoreStdout(t, rOut, wOut, rErr, wErr)
	assert.Contains(t, stderr, "Test")
	assert.Empty(t, stdout)
}

func TestWarn(t *testing.T) {
	rOut, wOut, rErr, wErr := redirectOutput(t)

	Warn("Test")

	stdout, stderr := restoreStdout(t, rOut, wOut, rErr, wErr)
	assert.Contains(t, stderr, "Test\n")
	assert.Empty(t, stdout)
}
