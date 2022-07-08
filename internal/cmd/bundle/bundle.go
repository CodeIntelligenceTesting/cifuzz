package bundle

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"code-intelligence.com/cifuzz/internal/build/cmake"
	"code-intelligence.com/cifuzz/internal/completion"
	"code-intelligence.com/cifuzz/internal/config"
	"code-intelligence.com/cifuzz/pkg/artifact"
	"code-intelligence.com/cifuzz/pkg/dialog"
	"code-intelligence.com/cifuzz/pkg/log"
	"code-intelligence.com/cifuzz/pkg/vcs"
	"code-intelligence.com/cifuzz/util/fileutil"
)

// The (possibly empty) directory inside the fuzzing artifact archive that will be the fuzzers working directory.
const fuzzerWorkDirPath = "work_dir"

// System library dependencies that are so common that we shouldn't emit a warning for them.
var wellKnownSystemLibraries = []*regexp.Regexp{
	versionedLibraryRegexp("ld-linux-x86-64.so"),
	versionedLibraryRegexp("libc.so"),
	versionedLibraryRegexp("libgcc_s.so"),
	versionedLibraryRegexp("libm.so"),
	versionedLibraryRegexp("libstdc++.so"),
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
	fuzzTest   string
	outputPath string
}

func New(conf *config.Config) *cobra.Command {
	opts := &bundleOpts{}
	cmd := &cobra.Command{
		Use:               "bundle",
		Short:             "Bundles the fuzz test into an archive",
		Long:              "Bundles all runtime artifacts required by the given fuzz test into a self-contained archive that can be executed by a remote fuzzing worker",
		ValidArgsFunction: completion.ValidFuzzTests,
		Args:              cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if conf.BuildSystem != config.BuildSystemCMake {
				return errors.New("cifuzz bundle currently only supports CMake projects")
			}
			opts.fuzzTest = args[0]
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
		c.opts.outputPath, err = dialog.InputFilename(
			c.InOrStdin(),
			"Please enter the filename for the artifact (.tar.gz)",
			c.opts.fuzzTest+".tar.gz",
		)
		if err != nil {
			return err
		}
	}

	builder, err := cmake.BuildWithCMake(c.config, c.OutOrStdout(), c.ErrOrStderr(), c.opts.fuzzTest)
	if err != nil {
		return errors.Wrapf(err, "failed to build %q", c.opts.fuzzTest)
	}
	fuzzers, archiveManifest, systemDeps, err := assembleArtifacts(c.opts.fuzzTest, builder)
	if err != nil {
		return err
	}

	tempDir, err := os.MkdirTemp("", "cifuzz-bundle-*")
	if err != nil {
		return err
	}
	defer fileutil.Cleanup(tempDir)

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

	// The fuzzing artifact archive spec requires this directory
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

func assembleArtifacts(fuzzTest string, builder *cmake.Builder) (
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
	archivePathPrefix := filepath.Join(builder.Engine, strings.Join(builder.Sanitizers, "+"), fuzzTest)

	fuzzTestExecutableRelPath, err := filepath.Rel(builder.BuildDir, fuzzTestExecutableAbsPath)
	if err != nil {
		err = errors.WithStack(err)
		return
	}
	fuzzTestArchivePath := filepath.Join(archivePathPrefix, fuzzTestExecutableRelPath)
	archiveManifest[fuzzTestArchivePath] = fuzzTestExecutableAbsPath

	runtimeDeps, err := builder.GetRuntimeDeps(fuzzTest)
depsLoop:
	for _, dep := range runtimeDeps {
		var isUnderBuildDir bool
		isUnderBuildDir, err = fileutil.IsUnder(dep, builder.BuildDir)
		if err != nil {
			return
		}
		if isUnderBuildDir {
			var buildDirRelPath string
			buildDirRelPath, err = filepath.Rel(builder.BuildDir, dep)
			if err != nil {
				err = errors.WithStack(err)
				return
			}
			archiveManifest[filepath.Join(archivePathPrefix, buildDirRelPath)] = dep
		} else {
			for _, wellKnownSystemLibrary := range wellKnownSystemLibraries {
				if wellKnownSystemLibrary.MatchString(dep) {
					continue depsLoop
				}
			}
			systemDeps = append(systemDeps, dep)
		}
	}

	for _, sanitizer := range builder.Sanitizers {
		if sanitizer == "undefined" {
			// The artifact archive spec does not support UBSan as a standalone sanitizer.
			continue
		}
		fuzzers = append(fuzzers, &artifact.Fuzzer{
			Target:    fuzzTest,
			Path:      fuzzTestArchivePath,
			Engine:    "LIBFUZZER",
			Sanitizer: strings.ToUpper(sanitizer),
			BuildDir:  builder.BuildDir,
		})
	}

	return
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
