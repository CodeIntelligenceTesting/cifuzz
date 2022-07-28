//go:build linux

package process_wrapper

import (
	"bytes"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code-intelligence.com/cifuzz/pkg/install"
	"code-intelligence.com/cifuzz/pkg/runfiles"
)

func TestMain(m *testing.M) {
	bundler, err := install.NewInstallationBundler(install.Options{Version: "dev"})
	if err != nil {
		log.Fatalf("%+v", err)
	}
	err = bundler.BuildProcessWrapper()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	runfiles.Finder = runfiles.RunfilesFinderImpl{InstallDir: bundler.TargetDir}

	res := m.Run()
	bundler.Cleanup()
	os.Exit(res)
}

func TestProcessWrapper_ChangesDirectory(t *testing.T) {
	processWrapperPath, err := runfiles.Finder.ProcessWrapperPath()
	require.NoError(t, err)
	tempDir, err := os.MkdirTemp("", "process_wrapper")
	require.NoError(t, err)

	var errbuf bytes.Buffer
	var outbuf bytes.Buffer
	cmd := exec.Command(processWrapperPath, tempDir, "--", "/usr/bin/pwd")
	cmd.Stderr = &errbuf
	cmd.Stdout = &outbuf
	err = cmd.Run()
	assert.NoError(t, err)
	assert.Empty(t, errbuf.String())
	assert.Equal(t, tempDir, strings.TrimSpace(outbuf.String()))
}

func TestProcessWrapper_SetsArgvAndEnvp(t *testing.T) {
	processWrapperPath, err := runfiles.Finder.ProcessWrapperPath()
	require.NoError(t, err)
	tempDir, err := os.MkdirTemp("", "process_wrapper")
	require.NoError(t, err)

	var errbuf bytes.Buffer
	var outbuf bytes.Buffer
	cmd := exec.Command(
		processWrapperPath,
		tempDir,
		"WRAPPER_ONLY_VAR=wrapper",
		"WRAPPER_AND_ARGUMENT_VAR=wrapper",
		"--",
		"/usr/bin/env",
		"ARGUMENT_ONLY_VAR=argument",
		"WRAPPER_AND_ARGUMENT_VAR=argument",
	)
	cmd.Stderr = &errbuf
	cmd.Stdout = &outbuf
	err = cmd.Run()
	assert.NoError(t, err)
	assert.Empty(t, errbuf.String())
	assert.Equal(
		t,
		"WRAPPER_ONLY_VAR=wrapper\nWRAPPER_AND_ARGUMENT_VAR=argument\nARGUMENT_ONLY_VAR=argument",
		strings.TrimSpace(outbuf.String()),
	)
}
