package bundle

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"

	"code-intelligence.com/cifuzz/internal/build/cmake"
	"code-intelligence.com/cifuzz/internal/completion"
	"code-intelligence.com/cifuzz/internal/config"
	"code-intelligence.com/cifuzz/pkg/artifact"
	"code-intelligence.com/cifuzz/pkg/log"
	"code-intelligence.com/cifuzz/pkg/vcs"
	"code-intelligence.com/cifuzz/util/fileutil"
)

// The (possibly empty) directory inside the fuzzing artifact archive that will be the fuzzers working directory.
const fuzzerWorkDirPath = "work_dir"

// Runtime dependencies of fuzz tests that live under these paths will not be included in the artifact archive and have
// to be provided by the Docker image instead.
var systemLibraryPaths = map[string][]string{
	"linux": {
		"/lib",
		"/usr/lib",
	},
	"darwin": {
		"/lib",
		"/usr/lib",
	},
}

// System library dependencies that are so common that we shouldn't emit a warning for them - they will be contained in
// any reasonable Docker image.
var wellKnownSystemLibraries = map[string][]*regexp.Regexp{
	"linux": {
		versionedLibraryRegexp("ld-linux-x86-64.so"),
		versionedLibraryRegexp("libc.so"),
		versionedLibraryRegexp("libgcc_s.so"),
		versionedLibraryRegexp("libm.so"),
		versionedLibraryRegexp("libstdc++.so"),
	},
}

func versionedLibraryRegexp(unversionedBasename string) *regexp.Regexp {
	return regexp.MustCompile(".*/" + regexp.QuoteMeta(unversionedBasename) + "[.0-9]*")
}

type bundleCmd struct {
	*cobra.Command

	config *config.Config
	opts   *bundleOpts
}

type bundleOpts struct {
	fuzzTests  []string
	outputPath string
}

func New(conf *config.Config) *cobra.Command {
	opts := &bundleOpts{}
	cmd := &cobra.Command{
		Use:               "bundle",
		Short:             "Bundles the fuzz test into an archive",
		Long:              "Bundles all runtime artifacts required by the given fuzz test into a self-contained archive that can be executed by a remote fuzzing worker",
		ValidArgsFunction: completion.ValidFuzzTests,
		Args:              cobra.ArbitraryArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if conf.BuildSystem != config.BuildSystemCMake {
				return errors.New("cifuzz bundle currently only supports CMake projects")
			}
			opts.fuzzTests = args
			return nil
		},
		RunE: func(c *cobra.Command, args []string) error {
			cmd := bundleCmd{
				Command: c,
				opts:    opts,
				config:  conf,
			}
			return cmd.run()
		},
	}

	cmd.Flags().StringVarP(&opts.outputPath, "output", "o", "", "Output path of the artifact (.tar.gz)")

	return cmd
}

func (c *bundleCmd) run() (err error) {
	if c.opts.outputPath == "" {
		if len(c.opts.fuzzTests) == 1 {
			c.opts.outputPath = c.opts.fuzzTests[0] + ".tar.gz"
		} else {
			c.opts.outputPath = "fuzz_tests.tar.gz"
		}
	}

	// TODO: Do not hardcode these values.
	sanitizers := []string{"address"}
	// UBSan is not supported by MSVC
	// TODO: Not needed anymore when sanitizers are configurable,
	//       then we do want to fail if the user explicitly asked for
	//       UBSan.
	if runtime.GOOS != "windows" {
		sanitizers = append(sanitizers, "undefined")
	}

	builder, err := cmake.NewBuilder(&cmake.BuilderOptions{
		ProjectDir: c.config.ProjectDir,
		// TODO: Do not hardcode this values.
		Engine:     "libfuzzer",
		Sanitizers: sanitizers,
		Stdout:     c.OutOrStdout(),
		Stderr:     c.ErrOrStderr(),
	})
	if err != nil {
		return err
	}

	err = builder.Configure()
	if err != nil {
		return err
	}

	if len(c.opts.fuzzTests) == 0 {
		c.opts.fuzzTests, err = builder.ListFuzzTests()
		if err != nil {
			return err
		}
	}

	err = builder.Build(c.opts.fuzzTests)
	if err != nil {
		return err
	}

	// Add all fuzz test artifacts to the archive.
	var fuzzers []*artifact.Fuzzer
	archiveManifest := make(map[string]string)
	deduplicatedSystemDeps := make(map[string]struct{})
	for _, fuzzTest := range c.opts.fuzzTests {
		fuzzTestFuzzers, fuzzTestArchiveManifest, systemDeps, err := assembleArtifacts(fuzzTest, builder)
		if err != nil {
			return err
		}
		fuzzers = append(fuzzers, fuzzTestFuzzers...)
		for _, systemDep := range systemDeps {
			deduplicatedSystemDeps[systemDep] = struct{}{}
		}
		// Produce an error when artifacts for different fuzzers conflict - this should never happen as
		// assembleArtifacts is expected to add a unique prefix for each fuzz test.
		for archivePath, absPath := range fuzzTestArchiveManifest {
			existingAbsPath, conflict := archiveManifest[archivePath]
			if conflict {
				return errors.Errorf("conflict for archive path %q: %q and %q", archivePath, existingAbsPath, absPath)
			}
			archiveManifest[archivePath] = absPath
		}
	}
	systemDeps := maps.Keys(deduplicatedSystemDeps)
	sort.Strings(systemDeps)

	tempDir, err := os.MkdirTemp("", "cifuzz-bundle-*")
	if err != nil {
		return err
	}
	defer fileutil.Cleanup(tempDir)

	// Create and add the top-level metadata file.
	metadata := &artifact.Metadata{
		Fuzzers: fuzzers,
		RunEnvironment: &artifact.RunEnvironment{
			// TODO(fmeum): Make configurable.
			Docker: "ubuntu",
		},
		CodeRevision: getCodeRevision(),
	}
	metadataYamlContent, err := metadata.ToYaml()
	if err != nil {
		return err
	}
	metadataYamlPath := filepath.Join(tempDir, artifact.MetadataFileName)
	err = os.WriteFile(metadataYamlPath, metadataYamlContent, 0644)
	if err != nil {
		return errors.Wrapf(err, "failed to write %s", artifact.MetadataFileName)
	}
	archiveManifest[artifact.MetadataFileName] = metadataYamlPath

	// The fuzzing artifact archive spec requires this directory even if it is empty.
	workDirPath := filepath.Join(tempDir, fuzzerWorkDirPath)
	err = os.Mkdir(workDirPath, 0755)
	if err != nil {
		return errors.WithStack(err)
	}
	archiveManifest[fuzzerWorkDirPath] = workDirPath

	archive, err := os.Create(c.opts.outputPath)
	if err != nil {
		return errors.Wrap(err, "failed to create fuzzing artifact archive")
	}
	archiveWriter := bufio.NewWriter(archive)
	defer archiveWriter.Flush()
	err = artifact.WriteArchive(archiveWriter, archiveManifest)
	if err != nil {
		return errors.Wrap(err, "failed to write fuzzing artifact archive")
	}

	log.Successf("Successfully created artifact: %s", c.opts.outputPath)
	if len(systemDeps) != 0 {
		log.Warnf(`The following system libraries are not part of the artifact and have to be provided by the Docker image %q:
  %s`, metadata.RunEnvironment.Docker, strings.Join(systemDeps, "\n  "))
	}
	return nil
}

func assembleArtifacts(fuzzTest string, builder Builder) (
	fuzzers []*artifact.Fuzzer,
	archiveManifest map[string]string,
	systemDeps []string,
	err error,
) {
	fuzzTestExecutableAbsPath, err := builder.FindFuzzTestExecutable(fuzzTest)
	if err != nil {
		err = errors.Wrapf(err, "failed to find fuzz test executable for %q", fuzzTest)
		return
	}

	archiveManifest = make(map[string]string)
	// Add all build artifacts under a subdirectory of the fuzz test base path so that these files don't clash with
	// seeds and dictionaries.
	buildArtifactsPrefix := filepath.Join(fuzzTestPrefix(fuzzTest, builder), "bin")

	// Add the fuzz test executable.
	fuzzTestExecutableRelPath, err := filepath.Rel(builder.BuildDir(), fuzzTestExecutableAbsPath)
	if err != nil {
		err = errors.WithStack(err)
		return
	}
	fuzzTestArchivePath := filepath.Join(buildArtifactsPrefix, fuzzTestExecutableRelPath)
	archiveManifest[fuzzTestArchivePath] = fuzzTestExecutableAbsPath

	// Add the runtime dependencies of the fuzz test executable.
	runtimeDeps, err := builder.GetRuntimeDeps(fuzzTest)
	if err != nil {
		return
	}
	externalLibrariesPrefix := ""
depsLoop:
	for _, dep := range runtimeDeps {
		var isBelowBuildDir bool
		isBelowBuildDir, err = fileutil.IsBelow(dep, builder.BuildDir())
		if err != nil {
			return
		}
		if isBelowBuildDir {
			var buildDirRelPath string
			buildDirRelPath, err = filepath.Rel(builder.BuildDir(), dep)
			if err != nil {
				err = errors.WithStack(err)
				return
			}
			archiveManifest[filepath.Join(buildArtifactsPrefix, buildDirRelPath)] = dep
			continue
		}

		// The runtime dependency is not built as part of the current project. It will be of one of the following types:
		// 1. A standard system library that is available in all reasonable Docker images.
		// 2. A more uncommon system library that may require additional packages to be installed (e.g. X11), but still
		//    lives in a standard system library directory (e.g. /usr/lib). Such dependencies are expected to be
		//    provided by the Docker image used as the run environment.
		// 3. Any other external dependency (e.g. a CMake target imported from another CMake project with a separate
		//    build directory). These are not expected to be part of the Docker image and thus added to the archive
		//    in a special directory that is added to the library search path at runtime.

		// 1. is handled by ignoring these runtime dependencies.
		for _, wellKnownSystemLibrary := range wellKnownSystemLibraries[runtime.GOOS] {
			if wellKnownSystemLibrary.MatchString(dep) {
				continue depsLoop
			}
		}

		// 2. is handled by returning a list of these libraries that is shown to the user as a warning about the
		// required contents of the Docker image specified as the run environment.
		for _, systemLibraryPath := range systemLibraryPaths[runtime.GOOS] {
			var isBelowLibPath bool
			isBelowLibPath, err = fileutil.IsBelow(dep, systemLibraryPath)
			if err != nil {
				return
			}
			if isBelowLibPath {
				systemDeps = append(systemDeps, dep)
				continue depsLoop
			}
		}

		// 3. is handled by staging the dependency in a special external library directory in the archive that is added
		// to the library search path in the run environment.
		// Note: Since all libraries are placed in a single directory, we have to ensure that basenames of external
		// libraries are unique. If they aren't, we report a conflict.
		externalLibrariesPrefix = filepath.Join(fuzzTestPrefix(fuzzTest, builder), "external_libs")
		archivePath := filepath.Join(externalLibrariesPrefix, filepath.Base(dep))
		if conflictingDep, hasConflict := archiveManifest[archivePath]; hasConflict {
			err = errors.Errorf(
				"fuzz test %q has conflicting runtime dependencies: %s and %s",
				fuzzTest,
				dep,
				conflictingDep,
			)
			return
		}
		archiveManifest[archivePath] = dep
	}

	// Add the default seed corpus directory if it exists.
	seedCorpus, err := builder.FindFuzzTestSeedCorpus(fuzzTest)
	if err != nil {
		return
	}
	var exists bool
	exists, err = fileutil.Exists(seedCorpus)
	if err != nil {
		return
	}
	archiveSeedsDir := ""
	if exists {
		archiveSeedsDir = filepath.Join(fuzzTestPrefix(fuzzTest, builder), "seeds")
		err = artifact.AddDirToManifest(archiveManifest, archiveSeedsDir, seedCorpus)
	}

	for _, sanitizer := range builder.Opts().Sanitizers {
		if sanitizer == "undefined" {
			// The artifact archive spec does not support UBSan as a standalone sanitizer.
			continue
		}
		fuzzers = append(fuzzers, &artifact.Fuzzer{
			Target:       fuzzTest,
			Path:         fuzzTestArchivePath,
			Engine:       "LIBFUZZER",
			Sanitizer:    strings.ToUpper(sanitizer),
			BuildDir:     builder.BuildDir(),
			Seeds:        archiveSeedsDir,
			LibraryPaths: externalLibrariesPrefix,
		})
	}

	return
}

// fuzzTestPrefix returns the path in the resulting artifact archive under which fuzz test specific files should be
// added.
func fuzzTestPrefix(fuzzTest string, builder Builder) string {
	sanitizerSegment := strings.Join(builder.Opts().Sanitizers, "+")
	if sanitizerSegment == "" {
		sanitizerSegment = "none"
	}
	return filepath.Join(builder.Opts().Engine, sanitizerSegment, fuzzTest)
}

func getCodeRevision() (codeRevision *artifact.CodeRevision) {
	gitCommit, err := vcs.GitCommit()
	if err != nil {
		log.Debugf("failed to get Git commit: %+v", err)
		return
	}

	gitBranch, err := vcs.GitBranch()
	if err != nil {
		log.Debugf("failed to get Git branch: %+v", err)
		return
	}

	if vcs.GitIsDirty() {
		log.Warnf("The Git repository has uncommitted changes. Archive metadata may be inaccurate.")
	}

	codeRevision = &artifact.CodeRevision{
		Git: &artifact.GitRevision{
			Commit: gitCommit,
			Branch: gitBranch,
		},
	}
	return
}

type Builder interface {
	Opts() *cmake.BuilderOptions
	BuildDir() string

	FindFuzzTestExecutable(fuzzTest string) (string, error)
	FindFuzzTestSeedCorpus(fuzzTest string) (string, error)
	GetRuntimeDeps(fuzzTest string) ([]string, error)
}
