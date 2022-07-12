package integration_tests

import (
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestIntegration_WithEnvs_NoStatsPrinted(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Parallel()

	buildDir := BuildFuzzTarget(t, "trigger_asan")

	TestWithAndWithoutMinijail(t, func(t *testing.T, disableMinijail bool) {
		test := NewLibfuzzerTest(t, buildDir, "trigger_asan", disableMinijail)
		test.FuzzerEnv = []string{"ASAN_OPTIONS=print_stats=0"}
		test.Timeout = time.Second

		output, _ := test.Run(t)
		require.NotContains(t, output, "Stats:")
	})
}

func TestIntegration_WithEnvs_StatsPrinted(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Parallel()

	buildDir := BuildFuzzTarget(t, "trigger_asan")

	TestWithAndWithoutMinijail(t, func(t *testing.T, disableMinijail bool) {
		test := NewLibfuzzerTest(t, buildDir, "trigger_asan", disableMinijail)
		test.FuzzerEnv = []string{"ASAN_OPTIONS=print_stats=1:atexit=1"}
		test.Timeout = time.Second

		output, _ := test.Run(t)
		require.Contains(t, output, "Stats:")
	})
}

func TestIntegration_WithEnvs_SpacesInEnvFlag(t *testing.T) {
	if testing.Short() || runtime.GOOS == "windows" {
		t.Skip()
	}
	t.Parallel()

	buildDir := BuildFuzzTarget(t, "trigger_asan")

	// Test that environment variables with spaces don't pass
	// arbitrary minijail flags. A stray "foo" argument would cause
	// minijail to fail.
	// We use an executable which immediately passes to not waste
	// resources.
	test := NewLibfuzzerTest(t, buildDir, "trigger_asan", true)
	test.FuzzTarget = "true"
	test.FuzzerEnv = []string{"FOO=bar foo"}
	test.Timeout = time.Second

	test.Run(t)
}
