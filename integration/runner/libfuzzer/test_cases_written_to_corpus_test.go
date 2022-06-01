package libfuzzer

import (
	"testing"

	"code-intelligence.com/cifuzz/integration/utils"
)

func TestIntegration_CasesWrittenToCorpus(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	utils.TestWithAndWithoutMinijail(t, func(t *testing.T, disableMinijail bool) {
		test := utils.NewLibfuzzerTest(t, "new_paths_fuzzer", disableMinijail)

		_, _, reports := test.Run(t)

		utils.CheckReports(t, reports, &utils.CheckReportOptions{
			NumFindings: 0,
		})

		test.RequireSeedCorpusNotEmpty(t)
	})
}
