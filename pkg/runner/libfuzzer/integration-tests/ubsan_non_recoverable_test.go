package integration_tests

import (
	"runtime"
	"testing"

	"code-intelligence.com/cifuzz/pkg/report"
	"code-intelligence.com/cifuzz/pkg/runner/libfuzzer/integration-tests/testutils"
)

func TestIntegration_UBSANNonRecoverable(t *testing.T) {
	// We are using msvc on windows, which does not support UBSan yet.
	if testing.Short() || runtime.GOOS == "windows" {
		t.Skip()
	}

	testutils.TestWithAndWithoutMinijail(t, func(t *testing.T, disableMinijail bool) {
		test := testutils.NewLibfuzzerTest(t, "trigger_ubsan_non_recoverable", disableMinijail)

		_, _, reports := test.Run(t)

		testutils.CheckReports(t, reports, &testutils.CheckReportOptions{
			ErrorType:   report.ErrorType_RUNTIME_ERROR,
			Details:     "undefined behaviour",
			SourceFile:  "trigger_ubsan.cpp",
			NumFindings: 1,
		})

		// We don't check here that the seed corpus is non-empty because
		// the fuzz target triggers the undefined behavior immediately,
		// so that no interesting inputs can be tested and stored before
		// the crash is triggered.
	})
}
