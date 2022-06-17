package integration_tests

import (
	"runtime"
	"testing"

	"code-intelligence.com/cifuzz/pkg/report"
	"code-intelligence.com/cifuzz/pkg/runner/libfuzzer/integration-tests/testutils"
)

func TestIntegration_InputTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	testutils.TestWithAndWithoutMinijail(t, func(t *testing.T, disableMinijail bool) {
		test := testutils.NewLibfuzzerTest(t, "trigger_timeout", disableMinijail)
		// The input timeout should be reported on the first input
		test.RunsLimit = 1
		test.EngineArgs = append(test.EngineArgs, "-timeout=1")

		_, _, reports := test.Run(t)

		options := &testutils.CheckReportOptions{
			ErrorType:   report.ErrorType_CRASH,
			Details:     "timeout",
			NumFindings: 1,
		}
		if runtime.GOOS == "linux" {
			options.SourceFile = "trigger_timeout.cpp"
		}

		testutils.CheckReports(t, reports, options)
	})
}
