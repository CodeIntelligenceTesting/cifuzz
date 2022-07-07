package integration_tests

import (
	"testing"

	"code-intelligence.com/cifuzz/pkg/report"
	"code-intelligence.com/cifuzz/pkg/runner/libfuzzer/integration-tests/testutils"
)

func TestIntegration_OOM(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	testutils.BuildFuzzTarget(t, "trigger_oom")

	testutils.TestWithAndWithoutMinijail(t, func(t *testing.T, disableMinijail bool) {
		test := testutils.NewLibfuzzerTest(t, "trigger_oom", disableMinijail)
		test.EngineArgs = append(test.EngineArgs, "-malloc_limit_mb=1")

		_, reports := test.Run(t)

		testutils.CheckReports(t, reports, &testutils.CheckReportOptions{
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
