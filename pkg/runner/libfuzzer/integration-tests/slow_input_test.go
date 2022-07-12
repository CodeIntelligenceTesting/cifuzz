package integration_tests

import (
	"testing"

	"code-intelligence.com/cifuzz/pkg/report"
)

func TestIntegration_SlowInput(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Parallel()

	buildDir := BuildFuzzTarget(t, "trigger_slow_input")

	TestWithAndWithoutMinijail(t, func(t *testing.T, disableMinijail bool) {
		test := NewLibfuzzerTest(t, buildDir, "trigger_slow_input", disableMinijail)
		// The input timeout should be reported on the first input
		test.RunsLimit = 1
		test.EngineArgs = append(test.EngineArgs, "-report_slow_units=1")

		_, reports := test.Run(t)

		CheckReports(t, reports, &CheckReportOptions{
			ErrorType:   report.ErrorType_WARNING,
			Details:     "Slow input detected",
			NumFindings: 1,
		})
	})
}
