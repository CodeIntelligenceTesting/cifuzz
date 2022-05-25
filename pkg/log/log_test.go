package log

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

var origStderr *os.File

func TestMain(m *testing.M) {
	origStderr = os.Stderr

	viper.Set("verbose", false)
	m.Run()
	viper.Set("verbose", false)

	os.Stderr = origStderr
}

func redirectOutput(t *testing.T) (*os.File, *os.File) {
	t.Helper()

	rErr, wErr, _ := os.Pipe()
	os.Stderr = wErr
	return rErr, wErr
}

func restoreStderr(t *testing.T, rErr, wErr *os.File) string {
	t.Helper()

	wErr.Close()
	defer rErr.Close()

	stderr, _ := ioutil.ReadAll(rErr)

	return string(stderr)
}

func TestDebugF_NoVerbose(t *testing.T) {
	rErr, wErr := redirectOutput(t)

	Debugf("Test")

	stderr := restoreStderr(t, rErr, wErr)

	assert.Empty(t, stderr)
}

func TestDebugF_Verbose(t *testing.T) {
	rErr, wErr := redirectOutput(t)

	viper.Set("verbose", true)
	Debugf("Test")
	viper.Set("verbose", false)

	stderr := restoreStderr(t, rErr, wErr)

	assert.Contains(t, stderr, "Test")
}

func TestDebug(t *testing.T) {
	rErr, wErr := redirectOutput(t)

	viper.Set("verbose", true)
	Debug("Test")
	viper.Set("verbose", false)

	stderr := restoreStderr(t, rErr, wErr)
	assert.Contains(t, stderr, "Test\n")
}

func TestErrorF_Verbose(t *testing.T) {
	rErr, wErr := redirectOutput(t)

	viper.Set("verbose", true)
	Errorf(errors.New("test-error"), "Test")
	viper.Set("verbose", false)

	stderr := restoreStderr(t, rErr, wErr)

	assert.Contains(t, stderr, "Test")
	assert.Contains(t, stderr, "test-error")
}

func TestErrorF_NoVerbose(t *testing.T) {
	rErr, wErr := redirectOutput(t)

	Errorf(errors.New("test-error"), "Test")

	stderr := restoreStderr(t, rErr, wErr)

	assert.Contains(t, stderr, "Test")
	assert.NotContains(t, stderr, "test-error")
}

func TestError(t *testing.T) {
	rErr, wErr := redirectOutput(t)

	Error(errors.New("test-error"), "Test")

	stderr := restoreStderr(t, rErr, wErr)

	assert.Contains(t, stderr, "Test\n")
}

func TestSuccessF(t *testing.T) {
	rErr, wErr := redirectOutput(t)

	Successf("Test")

	stderr := restoreStderr(t, rErr, wErr)
	assert.Contains(t, stderr, "Test")
}

func TestSuccess(t *testing.T) {
	rErr, wErr := redirectOutput(t)

	Success("Test")

	stderr := restoreStderr(t, rErr, wErr)
	assert.Contains(t, stderr, "Test\n")
}

func TestInfoF(t *testing.T) {
	rErr, wErr := redirectOutput(t)

	Infof("Test")

	stderr := restoreStderr(t, rErr, wErr)
	assert.Contains(t, stderr, "Test\n")
}

func TestInfo(t *testing.T) {
	rErr, wErr := redirectOutput(t)

	Info("Test")

	stderr := restoreStderr(t, rErr, wErr)
	assert.Contains(t, stderr, "Test\n")
}

func TestWarnF(t *testing.T) {
	rErr, wErr := redirectOutput(t)

	Warnf("Test")

	stderr := restoreStderr(t, rErr, wErr)
	assert.Contains(t, stderr, "Test")
}

func TestWarn(t *testing.T) {
	rErr, wErr := redirectOutput(t)

	Warn("Test")

	stderr := restoreStderr(t, rErr, wErr)
	assert.Contains(t, stderr, "Test\n")
}
