package integration_tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/otiai10/copy"
	"github.com/stretchr/testify/require"

	"code-intelligence.com/cifuzz/pkg/report"
)

func TestIntegration_CrashingCorpusEntry(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Parallel()

	BuildFuzzTarget(t, "trigger_asan")

	TestWithAndWithoutMinijail(t, func(t *testing.T, disableMinijail bool) {
		test := NewLibfuzzerTest(t, "trigger_asan", disableMinijail)
		test.RunsLimit = 0
		test.SeedCorpusDir = makeTemporarySeedCorpusDir(t)

		_, reports := test.Run(t)

		CheckReports(t, reports, &CheckReportOptions{
			ErrorType:   report.ErrorType_CRASH,
			SourceFile:  "trigger_asan.c",
			Details:     "heap-buffer-overflow",
			NumFindings: 1,
		})
	})
}

func makeTemporarySeedCorpusDir(t *testing.T) string {
	testDataDir := TestDataDir(t)
	crashingInput := filepath.Join(testDataDir, "corpus", "crashing_input")

	tmpCorpusDir, err := os.MkdirTemp(baseTempDir, "custom_seed_corpus-")
	require.NoError(t, err)

	require.NoError(t, err)
	err = copy.Copy(crashingInput, filepath.Join(tmpCorpusDir, "crashing_input"))
	require.NoError(t, err)

	entries, err := os.ReadDir(tmpCorpusDir)
	require.NoError(t, err)
	require.NotEmpty(t, entries)

	return tmpCorpusDir
}
