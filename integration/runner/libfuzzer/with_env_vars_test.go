package libfuzzer

import (
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"code-intelligence.com/cifuzz/integration/utils"
)

func TestIntegration_WithEnvs_NoStatsPrinted(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	utils.TestWithAndWithoutMinijail(t, func(t *testing.T, disableMinijail bool) {
		test := utils.NewLibfuzzerTest(t, "trigger_asan", disableMinijail)
		test.FuzzerEnv = []string{"ASAN_OPTIONS=print_stats=0"}
		test.Timeout = time.Second

		_, stderr, _ := test.Run(t)
		require.NotContains(t, stderr, "Stats:")
	})
}

func TestIntegration_WithEnvs_StatsPrinted(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	utils.TestWithAndWithoutMinijail(t, func(t *testing.T, disableMinijail bool) {
		test := utils.NewLibfuzzerTest(t, "trigger_asan", disableMinijail)
		test.FuzzerEnv = []string{"ASAN_OPTIONS=print_stats=1:atexit=1"}
		test.Timeout = time.Second

		_, stderr, _ := test.Run(t)
		require.Contains(t, stderr, "Stats:")
	})
}

func TestIntegration_WithEnvs_SpacesInEnvFlag(t *testing.T) {
	if testing.Short() || runtime.GOOS == "windows" {
		t.Skip()
	}

	// Test that environment variables with spaces don't pass
	// arbitrary minijail flags. A stray "foo" argument would cause
	// minijail to fail.
	// We use an executable which immediately passes to not waste
	// resources.
	test := utils.NewLibfuzzerTest(t, "trigger_asan", true)
	test.FuzzTarget = "true"
	test.FuzzerEnv = []string{"FOO=bar foo"}
	test.Timeout = time.Second

	test.Run(t)
}
