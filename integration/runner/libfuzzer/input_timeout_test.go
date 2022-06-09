package libfuzzer

import (
	"runtime"
	"testing"

	"code-intelligence.com/cifuzz/integration/utils"
	"code-intelligence.com/cifuzz/pkg/report"
)

func TestIntegration_InputTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	utils.TestWithAndWithoutMinijail(t, func(t *testing.T, disableMinijail bool) {
		test := utils.NewLibfuzzerTest(t, "trigger_timeout", disableMinijail)
		// The input timeout should be reported on the first input
		test.RunsLimit = 1
		test.EngineArgs = append(test.EngineArgs, "-timeout=1")

		_, _, reports := test.Run(t)

		options := &utils.CheckReportOptions{
			ErrorType:   report.ErrorType_CRASH,
			Details:     "timeout",
			NumFindings: 1,
		}
		if runtime.GOOS == "linux" {
			options.SourceFile = "trigger_timeout.cpp"
		}

		utils.CheckReports(t, reports, options)
	})
}