package libfuzzer

import (
	"testing"

	"code-intelligence.com/cifuzz/integration/utils"
	"code-intelligence.com/cifuzz/pkg/report"
)

func TestIntegration_WithFileAccesses(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	utils.TestWithAndWithoutMinijail(t, func(t *testing.T, disableMinijail bool) {

		test := utils.NewLibfuzzerTest(t, "trigger_asan_with_file_accesses", disableMinijail)
		// change to the build directory of the fuzz targets to make sure
		// that the needed example.conf is found
		test.ExecutionDir = utils.GetFuzzTargetBuildDir(t)
		_, _, reports := test.Run(t)

		utils.CheckReports(t, reports, &utils.CheckReportOptions{
			ErrorType:  report.ErrorType_CRASH,
			SourceFile: "trigger_asan_with_file_accesses.c",
			Details:    "heap-buffer-overflow",
		})
	})
}
