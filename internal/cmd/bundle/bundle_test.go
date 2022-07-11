package bundle

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code-intelligence.com/cifuzz/internal/build"
	"code-intelligence.com/cifuzz/pkg/artifact"
	"code-intelligence.com/cifuzz/util/fileutil"
)

type mockBuilder struct {
	*build.BuilderOptions
	seedCorpus string
}

func NewMockBuilder(projectDir, seedCorpus string) build.Builder {
	opts := &build.BuilderOptions{
		ProjectDir: projectDir,
		Engine:     "libfuzzer",
		Sanitizers: []string{"address"},
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
	}
	return &mockBuilder{
		BuilderOptions: opts,
		seedCorpus:     seedCorpus,
	}
}

func (m *mockBuilder) Opts() *build.BuilderOptions {
	return m.BuilderOptions
}

func (m *mockBuilder) BuildDir() string {
	return filepath.Join(m.Opts().ProjectDir, "build")
}

func (m *mockBuilder) Build(_ string) error {
	return nil
}

func (m *mockBuilder) Configure() error {
	return nil
}

func (m *mockBuilder) FindFuzzTestExecutable(fuzzTest string) (string, error) {
	return filepath.Join(m.BuildDir(), "pkg", fuzzTest), nil
}

func (m *mockBuilder) FindFuzzTestSeedCorpus(fuzzTest string) (string, error) {
	return m.seedCorpus, nil
}

// A library in a system library directory that is not certain to exist in the Docker image.
const uncommonSystemDepUnix = "/usr/lib/libBLAS.so"

// An external library in a non-system location.
var externalDep = generateExternalDepPath()

func generateExternalDepPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return filepath.Join(home, ".conan", "cache", "libfoo.so")
}

func (m *mockBuilder) GetRuntimeDeps(_ string) ([]string, error) {
	deps := []string{
		// A library in the project's build directory.
		filepath.Join(m.BuildDir(), "lib", "helper.so"),
		externalDep,
	}
	if runtime.GOOS != "windows" {
		deps = append(deps, uncommonSystemDepUnix)
	}
	return deps, nil
}

func TestAssembleArtifacts(t *testing.T) {
	seedCorpus, err := os.MkdirTemp("", "seed-corpus-*")
	require.NoError(t, err)
	defer fileutil.Cleanup(seedCorpus)
	err = fileutil.Touch(filepath.Join(seedCorpus, "seed"))
	require.NoError(t, err)

	// The project dir path has to be absolute, but doesn't have to exist.
	projectDir, err := filepath.Abs("project")
	require.NoError(t, err)

	builder := NewMockBuilder(projectDir, seedCorpus)

	fuzzers, manifest, systemDeps, err := assembleArtifacts("some_fuzz_test", builder)
	require.NoError(t, err)

	require.Equal(t, 1, len(fuzzers))
	assert.Equal(t, artifact.Fuzzer{
		Target:       "some_fuzz_test",
		Path:         filepath.Join("libfuzzer", "address", "some_fuzz_test", "bin", "pkg", "some_fuzz_test"),
		Engine:       "LIBFUZZER",
		Sanitizer:    "ADDRESS",
		BuildDir:     builder.BuildDir(),
		Seeds:        filepath.Join("libfuzzer", "address", "some_fuzz_test", "seeds"),
		LibraryPaths: filepath.Join("libfuzzer", "address", "some_fuzz_test", "external_libs"),
	}, *fuzzers[0])

	assert.Equal(t, map[string]string{
		filepath.Join("libfuzzer", "address", "some_fuzz_test", "bin", "pkg", "some_fuzz_test"): filepath.Join(builder.BuildDir(), "pkg", "some_fuzz_test"),
		filepath.Join("libfuzzer", "address", "some_fuzz_test", "bin", "lib", "helper.so"):      filepath.Join(builder.BuildDir(), "lib", "helper.so"),
		filepath.Join("libfuzzer", "address", "some_fuzz_test", "external_libs", "libfoo.so"):   externalDep,
		filepath.Join("libfuzzer", "address", "some_fuzz_test", "seeds"):                        seedCorpus,
		filepath.Join("libfuzzer", "address", "some_fuzz_test", "seeds", "seed"):                filepath.Join(seedCorpus, "seed"),
	}, manifest)

	if runtime.GOOS != "windows" {
		assert.Equal(t, []string{uncommonSystemDepUnix}, systemDeps)
	}
}
