package integration_tests

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/otiai10/copy"
	"github.com/stretchr/testify/require"

	"code-intelligence.com/cifuzz/pkg/report"
	"code-intelligence.com/cifuzz/pkg/runner/libfuzzer/integration-tests/testutils"
)

func TestIntegration_CrashingCorpusEntry(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	testutils.BuildFuzzTarget(t, "trigger_asan")

	testutils.TestWithAndWithoutMinijail(t, func(t *testing.T, disableMinijail bool) {
		test := testutils.NewLibfuzzerTest(t, "trigger_asan", disableMinijail)
		test.RunsLimit = 0
		test.SeedCorpusDir = makeTemporarySeedCorpusDir(t)

		_, _, reports := test.Run(t)

		testutils.CheckReports(t, reports, &testutils.CheckReportOptions{
			ErrorType:   report.ErrorType_CRASH,
			SourceFile:  "trigger_asan.c",
			Details:     "heap-buffer-overflow",
			NumFindings: 1,
		})
	})
}

func makeTemporarySeedCorpusDir(t *testing.T) string {
	testDataDir := testutils.GetTestDataDir(t)
	crashingInput := filepath.Join(testDataDir, "corpus", "crashing_input")

	tmpCorpusDir, err := ioutil.TempDir(baseTempDir, "custom_seed_corpus-")
	require.NoError(t, err)

	require.NoError(t, err)
	err = copy.Copy(crashingInput, filepath.Join(tmpCorpusDir, "crashing_input"), copy.Options{Sync: true})
	require.NoError(t, err)

	entries, err := os.ReadDir(tmpCorpusDir)
	require.NoError(t, err)
	require.NotEmpty(t, entries)

	return tmpCorpusDir
}
