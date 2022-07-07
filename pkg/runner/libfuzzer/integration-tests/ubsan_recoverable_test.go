package integration_tests

import (
	"runtime"
	"testing"

	"code-intelligence.com/cifuzz/pkg/report"
)

func TestIntegration_UBSANRecoverable(t *testing.T) {
	// We are using msvc on windows, which does not support UBSan yet.
	if testing.Short() || runtime.GOOS == "windows" {
		t.Skip()
	}

	BuildFuzzTarget(t, "trigger_ubsan")

	TestWithAndWithoutMinijail(t, func(t *testing.T, disableMinijail bool) {
		test := NewLibfuzzerTest(t, "trigger_ubsan", disableMinijail)

		_, reports := test.Run(t)

		CheckReports(t, reports, &CheckReportOptions{
			ErrorType:           report.ErrorType_RUNTIME_ERROR,
			Details:             "undefined behaviour",
			SourceFile:          "trigger_ubsan.cpp",
			AllowEmptyInputData: true,
			NumFindings:         1,
		})

		// We don't check here that the seed corpus is non-empty because
		// the fuzz target triggers the undefined behavior immediately,
		// so that no interesting inputs can be tested and stored before
		// the crash is triggered.
	})
}
