package report

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code-intelligence.com/cifuzz/util/fileutil"
	"code-intelligence.com/cifuzz/util/testutil"
)

func TestMain(m *testing.M) {
	testTempDir := testutil.ChdirToTempDir("finding-test-")
	defer fileutil.Cleanup(testTempDir)

	m.Run()
}

func TestFinding_Save(t *testing.T) {
	finding := &Finding{
		Name: "test-name",
		Logs: []string{
			"Oops",
			"The application crashed",
		},
	}

	err := finding.Save()
	require.NoError(t, err)

	findingDir := filepath.Join(nameFindingDir, finding.Name)
	jsonPath := filepath.Join(findingDir, nameJsonFile)
	assert.DirExists(t, findingDir)
	assert.FileExists(t, jsonPath)
	assert.NoFileExists(t, filepath.Join(findingDir, nameCrashingInput))

	bytes, err := os.ReadFile(jsonPath)
	jsonString := string(bytes)
	require.NoError(t, err)
	assert.Contains(t, jsonString, finding.Name)
	assert.Contains(t, jsonString, finding.Logs[0])
	assert.Contains(t, jsonString, finding.Logs[1])

}

func TestFinding_MoveInputFile(t *testing.T) {
	// create an input file
	testfile := "crash_123_test"
	err := os.WriteFile(testfile, []byte("TEST"), 0644)
	require.NoError(t, err)

	finding := &Finding{
		Name:      "test-name",
		InputFile: testfile,
		Logs: []string{
			"Oops",
			"The application crashed",
			fmt.Sprintf("some surrounding text, %s more text", testfile),
		},
	}

	err = finding.Save()
	require.NoError(t, err)

	newFilename := filepath.Join(nameFindingDir, finding.Name, nameCrashingInput)
	assert.FileExists(t, newFilename)
	// check if the log was updated
	assert.Contains(t, finding.Logs[2], newFilename)
}
