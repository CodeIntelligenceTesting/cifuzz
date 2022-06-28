package integration_tests

import (
	"testing"

	"code-intelligence.com/cifuzz/pkg/report"
)

func TestIntegration_ASAN(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Parallel()

	BuildFuzzTarget(t, "trigger_asan")

	TestWithAndWithoutMinijail(t, func(t *testing.T, disableMinijail bool) {
		test := NewLibfuzzerTest(t, "trigger_asan", disableMinijail)

		_, reports := test.Run(t)

		CheckReports(t, reports, &CheckReportOptions{
			ErrorType:   report.ErrorType_CRASH,
			SourceFile:  "trigger_asan.c",
			Details:     "heap-buffer-overflow",
			NumFindings: 1,
		})
	})
}
