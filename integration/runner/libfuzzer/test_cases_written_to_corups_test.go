package libfuzzer

import (
	"testing"

	"code-intelligence.com/cifuzz/integration/utils"
	"code-intelligence.com/cifuzz/pkg/report"
)

func TestIntegration_CasesWrittenToCorpus(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	utils.TestWithAndWithoutMinijail(t, func(t *testing.T, disableMinijail bool) {
		test := utils.NewLibfuzzerTest(t, "do_stuff_fuzzer", disableMinijail)

		_, _, reports := test.Run(t)

		utils.CheckReports(t, reports, &utils.CheckReportOptions{
			ErrorType:  report.ErrorType_CRASH,
			SourceFile: "do_stuff_fuzzer.cpp",
			Details:    "heap-buffer-overflow",
		})

		test.RequireSeedCorpusNotEmpty(t)
	})
}
