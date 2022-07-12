package cmake

import (
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"code-intelligence.com/cifuzz/util/fileutil"
)

var baseTempDir string

func TestMain(m *testing.M) {
	var err error
	baseTempDir, err = os.MkdirTemp("", "cmake-test-")
	if err != nil {
		log.Fatalf("Failed to create temp dir for tests: %+v", err)
	}
	defer fileutil.Cleanup(baseTempDir)

	m.Run()
}

func TestNewBuilder(t *testing.T) {
	projectDir, err := os.MkdirTemp(baseTempDir, "project-dir-")
	require.NoError(t, err)

	// Create a builder with engine "engine1"
	builder1, err := NewBuilder(&BuilderOptions{
		ProjectDir: projectDir,
		Engine:     "engine1",
		Sanitizers: []string{"sanitizer1", "sanitizer2"},
		Stdout:     os.Stderr,
		Stderr:     os.Stderr,
	})
	require.NoError(t, err)
	require.DirExists(t, builder1.BuildDir())

	// Create a builder with engine "engine2"
	builder2, err := NewBuilder(&BuilderOptions{
		ProjectDir: projectDir,
		Engine:     "engine2",
		Sanitizers: []string{"sanitizer1", "sanitizer2"},
		Stdout:     os.Stderr,
		Stderr:     os.Stderr,
	})
	require.NoError(t, err)
	require.DirExists(t, builder2.BuildDir())

	// Check that the two builders have different build directories
	// (because they use different engines)
	require.NotEqual(t, builder1.BuildDir(), builder2.BuildDir())

	// Create another builder with "engine1"
	builder3, err := NewBuilder(&BuilderOptions{
		ProjectDir: projectDir,
		Engine:     "engine1",
		Sanitizers: []string{"sanitizer1", "sanitizer2"},
		Stdout:     os.Stderr,
		Stderr:     os.Stderr,
	})
	require.NoError(t, err)
	require.DirExists(t, builder3.BuildDir())

	// Check that builder1 and builder3 have the same build directory
	// (because they use the same engine and sanitizers)
	require.Equal(t, builder1.BuildDir(), builder3.BuildDir())
}
