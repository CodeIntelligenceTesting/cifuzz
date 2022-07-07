package integration_tests

import (
	"testing"
)

func TestIntegration_CasesWrittenToCorpus(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	BuildFuzzTarget(t, "new_paths_fuzzer")

	TestWithAndWithoutMinijail(t, func(t *testing.T, disableMinijail bool) {
		test := NewLibfuzzerTest(t, "new_paths_fuzzer", disableMinijail)

		_, reports := test.Run(t)

		CheckReports(t, reports, &CheckReportOptions{
			NumFindings: 0,
		})

		test.RequireSeedCorpusNotEmpty(t)
	})
}
