package libfuzzer

import (
	"testing"

	"code-intelligence.com/cifuzz/integration/utils"
	"code-intelligence.com/cifuzz/pkg/report"
)

func TestIntegration_OOM(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	utils.TestWithAndWithoutMinijail(t, func(t *testing.T, disableMinijail bool) {
		test := utils.NewLibfuzzerTest(t, "trigger_oom", disableMinijail)
		test.EngineArgs = append(test.EngineArgs, "-malloc_limit_mb=1")

		_, _, reports := test.Run(t)

		utils.CheckReports(t, reports, &utils.CheckReportOptions{
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
