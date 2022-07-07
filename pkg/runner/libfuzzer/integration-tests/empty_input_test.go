package integration_tests

import (
	"runtime"
	"testing"

	"code-intelligence.com/cifuzz/pkg/report"
)

// Regression test: When crashing on an empty input the runner was not reporting a finding
// see also: https://code-intelligence.atlassian.net/browse/CLI-226
func TestIntegration_CrashOnEmptyInput(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	BuildFuzzTarget(t, "trigger_asan_on_empty_input")

	TestWithAndWithoutMinijail(t, func(t *testing.T, disableMinijail bool) {
		test := NewLibfuzzerTest(t, "trigger_asan_on_empty_input", disableMinijail)

		_, reports := test.Run(t)

		errMsg := "SEGV on unknown address"
		if runtime.GOOS == "windows" {
			errMsg = "access-violation on unknown address"
		}

		CheckReports(t, reports, &CheckReportOptions{
			ErrorType:           report.ErrorType_CRASH,
			SourceFile:          "trigger_asan_on_empty_input.c",
			Details:             errMsg,
			NumFindings:         1,
			AllowEmptyInputData: true,
		})
	})
}
