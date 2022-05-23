package libfuzzer

import (
	"testing"
	"time"

	"code-intelligence.com/cifuzz/integration/utils"
	"github.com/stretchr/testify/require"
)

func TestIntegration_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	utils.TestWithAndWithoutMinijail(t, func(t *testing.T, disableMinijail bool) {
		test := utils.NewLibfuzzerTest(t, "do_nothing_fuzzer", disableMinijail)
		test.Timeout = time.Second
		// Don't limit the number of runs, to ensure that the test stops
		// because of the timeout and not because the runs limit was
		// reached.
		test.RunsLimit = -1

		_, stderr, _ := test.Run(t)
		require.Contains(t, stderr, "DONE")
	})
}
