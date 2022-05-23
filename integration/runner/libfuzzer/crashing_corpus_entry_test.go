package libfuzzer

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"code-intelligence.com/cifuzz/integration/utils"
	"code-intelligence.com/cifuzz/pkg/report"
	"code-intelligence.com/cifuzz/util/fileutil"
)

func TestIntegration_CrashingCorpusEntry(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	utils.TestWithAndWithoutMinijail(t, func(t *testing.T, disableMinijail bool) {
		test := utils.NewLibfuzzerTest(t, "trigger_asan", disableMinijail)
		test.RunsLimit = 0
		test.SeedCorpusDir = makeTemporarySeedCorpusDir(t)

		_, _, reports := test.Run(t)

		utils.CheckReports(t, reports, &utils.CheckReportOptions{
			ErrorType:  report.ErrorType_CRASH,
			SourceFile: "trigger_asan.c",
			Details:    "heap-buffer-overflow",
		})
	})
}

func makeTemporarySeedCorpusDir(t *testing.T) string {
	rootDir := utils.GetProjectRoot(t)
	crashingInput := filepath.Join(rootDir, "testdata", "corpus", "crashing_input")
	tmpCorpusDir, err := ioutil.TempDir(baseTempDir, "custom_seed_corpus-")
	require.NoError(t, err)

	require.NoError(t, err)
	err = fileutil.CopyFile(crashingInput, filepath.Join(tmpCorpusDir, "crashing_input"), 0644)
	require.NoError(t, err)

	entries, err := os.ReadDir(tmpCorpusDir)
	require.NoError(t, err)
	require.NotEmpty(t, entries)

	return tmpCorpusDir
}
