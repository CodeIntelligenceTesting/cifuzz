package integration_tests

import (
	"testing"

	"code-intelligence.com/cifuzz/pkg/report"
)

func TestIntegration_WithFileAccesses(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	BuildFuzzTarget(t, "trigger_asan_with_file_accesses")

	TestWithAndWithoutMinijail(t, func(t *testing.T, disableMinijail bool) {

		test := NewLibfuzzerTest(t, "trigger_asan_with_file_accesses", disableMinijail)
		_, reports := test.Run(t)

		CheckReports(t, reports, &CheckReportOptions{
			ErrorType:   report.ErrorType_CRASH,
			SourceFile:  "trigger_asan_with_file_accesses.c",
			Details:     "heap-buffer-overflow",
			NumFindings: 1,
		})
	})
}
