package log

import (
	"bytes"
	"io"
	"testing"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testOut io.ReadWriter

func TestMain(m *testing.M) {
	testOut = bytes.NewBuffer([]byte{})
	Output = testOut
	disableColor = true

	viper.Set("verbose", false)
	m.Run()
	viper.Set("verbose", false)
}

func TestDebug_NoVerbose(t *testing.T) {
	Debugf("Test")
	out, err := io.ReadAll(testOut)
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestDebug_Verbose(t *testing.T) {
	viper.Set("verbose", true)
	Debugf("Test")
	viper.Set("verbose", false)
	checkOutput(t, "Test\n")
}

func TestError_Verbose(t *testing.T) {
	viper.Set("verbose", true)
	Errorf(errors.New("test-error"), "Test")
	viper.Set("verbose", false)
	checkOutput(t, "Test\n", "test-error")
}

func TestError_NoVerbose(t *testing.T) {
	Errorf(errors.New("test-error"), "Test")
	out := checkOutput(t, "Test\n")
	require.NotContains(t, out, "test-error")
}

func TestSuccess(t *testing.T) {
	Success("Test")
	checkOutput(t, "Test\n")
}

func TestInfo(t *testing.T) {
	Info("Test")
	checkOutput(t, "Test\n")
}

func TestWarn(t *testing.T) {
	Warn("Test")
	checkOutput(t, "Test\n")
}

func checkOutput(t *testing.T, a ...string) string {
	out, err := io.ReadAll(testOut)
	require.NoError(t, err)
	for _, s := range a {
		require.Contains(t, string(out), s)
	}
	return string(out)
}
