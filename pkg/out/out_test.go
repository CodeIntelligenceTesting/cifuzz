package out

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func redirectStdout(t *testing.T) (*os.File, *os.File, *os.File) {
	t.Helper()

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w
	return r, w, oldStdout
}

func restoreStdout(t *testing.T, r, w, oldStdout *os.File) string {
	t.Helper()

	w.Close()
	out, _ := ioutil.ReadAll(r)
	os.Stdout = oldStdout
	return string(out)
}

func TestDebug_NoVerbose(t *testing.T) {
	r, w, s := redirectStdout(t)

	viper.Set("verbose", false)
	Debug("Test")

	out := restoreStdout(t, r, w, s)

	assert.Empty(t, out)
}

func TestDebug_Verbose(t *testing.T) {
	r, w, s := redirectStdout(t)

	viper.Set("verbose", true)
	Debug("Test")
	viper.Set("verbose", false)

	out := restoreStdout(t, r, w, s)

	assert.Contains(t, out, "Test")
}

func TestError_Verbose(t *testing.T) {
	r, w, s := redirectStdout(t)

	viper.Set("verbose", true)
	Error(errors.New("test-error"), "Test")
	viper.Set("verbose", false)

	out := restoreStdout(t, r, w, s)

	assert.Contains(t, out, "Test")
	assert.Contains(t, out, "test-error")
}

func TestError_NoVerbose(t *testing.T) {
	r, w, s := redirectStdout(t)

	viper.Set("verbose", false)
	Error(errors.New("test-error"), "Test")

	out := restoreStdout(t, r, w, s)

	assert.Contains(t, out, "Test")
	assert.NotContains(t, out, "test-error")
}

func TestSuccess(t *testing.T) {
	r, w, s := redirectStdout(t)

	viper.Set("verbose", false)
	Success("Test")

	out := restoreStdout(t, r, w, s)
	assert.Contains(t, out, "Test")
}

func TestInfo(t *testing.T) {
	r, w, s := redirectStdout(t)

	viper.Set("verbose", false)
	Info("Test")

	out := restoreStdout(t, r, w, s)
	assert.Contains(t, out, "Test")
}

func TestWarn(t *testing.T) {
	r, w, s := redirectStdout(t)

	viper.Set("verbose", false)
	Warn("Test")

	out := restoreStdout(t, r, w, s)
	assert.Contains(t, out, "Test")
}
