package integration_tests

import (
	"testing"

	"code-intelligence.com/cifuzz/pkg/report"
	"code-intelligence.com/cifuzz/pkg/runner/libfuzzer/integration-tests/testutils"
)

func TestIntegration_WithFileAccesses(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	testutils.BuildFuzzTarget(t, "trigger_asan_with_file_accesses")

	testutils.TestWithAndWithoutMinijail(t, func(t *testing.T, disableMinijail bool) {

		test := testutils.NewLibfuzzerTest(t, "trigger_asan_with_file_accesses", disableMinijail)
		// change to the build directory of the fuzz targets to make sure
		// that the needed example.conf is found
		test.ExecutionDir = testutils.GetFuzzTargetBuildDir(t)
		_, _, reports := test.Run(t)

		testutils.CheckReports(t, reports, &testutils.CheckReportOptions{
			ErrorType:   report.ErrorType_CRASH,
			SourceFile:  "trigger_asan_with_file_accesses.c",
			Details:     "heap-buffer-overflow",
			NumFindings: 1,
		})
	})
}
