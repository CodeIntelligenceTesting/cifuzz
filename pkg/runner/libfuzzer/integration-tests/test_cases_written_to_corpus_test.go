package integration_tests

import (
	"testing"

	"code-intelligence.com/cifuzz/pkg/runner/libfuzzer/integration-tests/testutils"
)

func TestIntegration_CasesWrittenToCorpus(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	testutils.BuildFuzzTarget(t, "new_paths_fuzzer")

	testutils.TestWithAndWithoutMinijail(t, func(t *testing.T, disableMinijail bool) {
		test := testutils.NewLibfuzzerTest(t, "new_paths_fuzzer", disableMinijail)

		_, reports := test.Run(t)

		testutils.CheckReports(t, reports, &testutils.CheckReportOptions{
			NumFindings: 0,
		})

		test.RequireSeedCorpusNotEmpty(t)
	})
}
