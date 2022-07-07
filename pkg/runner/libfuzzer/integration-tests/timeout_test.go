package integration_tests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"code-intelligence.com/cifuzz/pkg/runner/libfuzzer/integration-tests/testutils"
)

func TestIntegration_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	testutils.BuildFuzzTarget(t, "do_nothing_fuzzer")

	testutils.TestWithAndWithoutMinijail(t, func(t *testing.T, disableMinijail bool) {
		test := testutils.NewLibfuzzerTest(t, "do_nothing_fuzzer", disableMinijail)
		test.Timeout = time.Second
		// Don't limit the number of runs, to ensure that the test stops
		// because of the timeout and not because the runs limit was
		// reached.
		test.RunsLimit = -1

		output, _ := test.Run(t)
		require.Contains(t, output, "DONE")
	})
}
