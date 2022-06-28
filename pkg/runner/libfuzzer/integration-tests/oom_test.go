package integration_tests

import (
	"testing"

	"code-intelligence.com/cifuzz/pkg/report"
)

func TestIntegration_OOM(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Parallel()

	BuildFuzzTarget(t, "trigger_oom")

	TestWithAndWithoutMinijail(t, func(t *testing.T, disableMinijail bool) {
		test := NewLibfuzzerTest(t, "trigger_oom", disableMinijail)
		test.EngineArgs = append(test.EngineArgs, "-malloc_limit_mb=1")

		_, reports := test.Run(t)

		CheckReports(t, reports, &CheckReportOptions{
			ErrorType:   report.ErrorType_CRASH,
			SourceFile:  "trigger_oom.cpp",
			Details:     "out-of-memory",
			NumFindings: 1,
		})

		// We don't check here that the seed corpus is non-empty because
		// the trigger_oom fuzz target triggers the OOM immediately, so
		// that no interesting inputs can be tested and stored before
		// the crash is triggered.
	})
}
