package integration_tests

import (
	"runtime"
	"testing"

	"code-intelligence.com/cifuzz/pkg/report"
)

func TestIntegration_InputTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Parallel()

	buildDir := BuildFuzzTarget(t, "trigger_timeout")

	TestWithAndWithoutMinijail(t, func(t *testing.T, disableMinijail bool) {
		test := NewLibfuzzerTest(t, buildDir, "trigger_timeout", disableMinijail)
		// The input timeout should be reported on the first input
		test.RunsLimit = 1
		test.EngineArgs = append(test.EngineArgs, "-timeout=1")

		_, reports := test.Run(t)

		options := &CheckReportOptions{
			ErrorType:   report.ErrorType_CRASH,
			Details:     "timeout",
			NumFindings: 1,
		}
		if runtime.GOOS == "linux" {
			options.SourceFile = "trigger_timeout.cpp"
		}

		CheckReports(t, reports, options)
	})
}
