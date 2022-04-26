package out

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

	m.Run()

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

func TestDebug_NoVerbose(t *testing.T) {
	rOut, wOut, rErr, wErr := redirectOutput(t)

	viper.Set("verbose", false)
	Debug("Test")

	stdout, stderr := restoreStdout(t, rOut, wOut, rErr, wErr)

	assert.Empty(t, stderr)
	assert.Empty(t, stdout)
}

func TestDebug_Verbose(t *testing.T) {
	rOut, wOut, rErr, wErr := redirectOutput(t)

	viper.Set("verbose", true)
	Debug("Test")
	viper.Set("verbose", false)

	stdout, stderr := restoreStdout(t, rOut, wOut, rErr, wErr)

	assert.Contains(t, stderr, "Test")
	assert.Empty(t, stdout, "Test")
}

func TestError_Verbose(t *testing.T) {
	rOut, wOut, rErr, wErr := redirectOutput(t)

	viper.Set("verbose", true)
	Error(errors.New("test-error"), "Test")
	viper.Set("verbose", false)

	stdout, stderr := restoreStdout(t, rOut, wOut, rErr, wErr)

	assert.Contains(t, stderr, "Test")
	assert.Contains(t, stderr, "test-error")
	assert.Empty(t, stdout)
}

func TestError_NoVerbose(t *testing.T) {
	rOut, wOut, rErr, wErr := redirectOutput(t)

	viper.Set("verbose", false)
	Error(errors.New("test-error"), "Test")

	stdout, stderr := restoreStdout(t, rOut, wOut, rErr, wErr)

	assert.Contains(t, stderr, "Test")
	assert.NotContains(t, stderr, "test-error")
	assert.Empty(t, stdout)
}

func TestSuccess(t *testing.T) {
	rOut, wOut, rErr, wErr := redirectOutput(t)

	viper.Set("verbose", false)
	Success("Test")

	stdout, stderr := restoreStdout(t, rOut, wOut, rErr, wErr)
	assert.Contains(t, stdout, "Test")
	assert.Empty(t, stderr)
}

func TestInfo(t *testing.T) {
	rOut, wOut, rErr, wErr := redirectOutput(t)

	viper.Set("verbose", false)
	Info("Test")

	stdout, stderr := restoreStdout(t, rOut, wOut, rErr, wErr)
	assert.Contains(t, stdout, "Test")
	assert.Empty(t, stderr)
}

func TestWarn(t *testing.T) {
	rOut, wOut, rErr, wErr := redirectOutput(t)

	viper.Set("verbose", false)
	Warn("Test")

	stdout, stderr := restoreStdout(t, rOut, wOut, rErr, wErr)
	assert.Contains(t, stderr, "Test")
	assert.Empty(t, stdout)
}
