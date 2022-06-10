package install

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/otiai10/copy"
	"github.com/pkg/errors"

	"code-intelligence.com/cifuzz/util/envutil"
	"code-intelligence.com/cifuzz/util/fileutil"
)

type installer struct {
	InstallDir string

	projectDir string
}

type Options struct {
	InstallDir string
}

func NewInstaller(opts *Options) (*installer, error) {
	if opts == nil {
		opts = &Options{}
	}

	projectDir, err := findProjectDir()
	if err != nil {
		return nil, err
	}

	if strings.HasPrefix(opts.InstallDir, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, errors.WithStack(err)
		}
		opts.InstallDir = home + strings.TrimPrefix(opts.InstallDir, "~")
	}

	if opts.InstallDir == "" {
		opts.InstallDir, err = ioutil.TempDir("", "cifuzz-install-dir-")
		if err != nil {
			return nil, errors.WithStack(err)
		}
	} else {
		exists, err := fileutil.Exists(opts.InstallDir)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		if exists {
			return nil, errors.Errorf("Install directory '%s' already exists. Please remove it to continue.", opts.InstallDir)
		}
	}

	i := &installer{
		projectDir: projectDir,
		InstallDir: opts.InstallDir,
	}

	// Create the directory layout
	err = os.MkdirAll(i.binDir(), 0755)
	if err != nil {
		fileutil.Cleanup(opts.InstallDir)
		return nil, errors.WithStack(err)
	}
	err = os.MkdirAll(i.libDir(), 0755)
	if err != nil {
		fileutil.Cleanup(opts.InstallDir)
		return nil, errors.WithStack(err)
	}
	err = os.MkdirAll(i.shareDir(), 0755)
	if err != nil {
		fileutil.Cleanup(opts.InstallDir)
		return nil, errors.WithStack(err)
	}

	return i, nil
}

func (i *installer) binDir() string {
	return filepath.Join(i.InstallDir, "bin")
}

func (i *installer) libDir() string {
	return filepath.Join(i.InstallDir, "lib")
}

func (i *installer) shareDir() string {
	return filepath.Join(i.InstallDir, "share")
}

func (i *installer) Cleanup() {
	fileutil.Cleanup(i.InstallDir)
}

func (i *installer) InstallCIFuzzAndDeps() error {
	err := i.InstallJazzer()
	if err != nil {
		return err
	}

	if runtime.GOOS == "linux" {
		err = i.InstallMinijail()
		if err != nil {
			return err
		}

		err = i.InstallProcessWrapper()
		if err != nil {
			return err
		}
	}

	err = i.InstallCMakeIntegration()
	if err != nil {
		return err
	}

	err = i.InstallCIFuzz()
	if err != nil {
		return err
	}

	return nil
}

func (i *installer) InstallJazzer() error {
	// Build Jazzer
	cmd := exec.Command("bazel", "build", "//agent:jazzer_agent_deploy", "//driver:jazzer_driver")
	cmd.Dir = filepath.Join(i.projectDir, "third-party/jazzer")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	log.Printf("Command: %s", cmd.String())
	err := cmd.Run()
	if err != nil {
		return errors.WithStack(err)
	}

	// Install Jazzer binaries
	src := filepath.Join(i.projectDir, "third-party/jazzer/bazel-bin/agent/jazzer_agent_deploy.jar")
	dest := filepath.Join(i.libDir(), "jazzer_agent_deploy.jar")
	err = fileutil.CopyFile(src, dest, 0600)
	if err != nil {
		return err
	}
	src = filepath.Join(i.projectDir, "third-party/jazzer/bazel-bin/driver/jazzer_driver")
	dest = filepath.Join(i.binDir(), "jazzer_driver")
	err = fileutil.CopyFile(src, dest, 0700)
	if err != nil {
		return err
	}
	return nil
}

func (i *installer) InstallMinijail() error {
	var err error

	// Build minijail
	cmd := exec.Command("make", "CC_BINARY(minijail0)", "CC_LIBRARY(libminijailpreload.so)")
	cmd.Dir = filepath.Join(i.projectDir, "third-party/minijail")
	// The minijail Makefile changes the directory to $PWD, so we have
	// to set that.
	cmd.Env, err = envutil.Setenv(os.Environ(), "PWD", filepath.Join(i.projectDir, "third-party/minijail"))
	if err != nil {
		return err
	}
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	log.Printf("Command: %s", cmd.String())
	err = cmd.Run()
	if err != nil {
		return errors.WithStack(err)
	}

	// Install minijail binaries
	src := filepath.Join(i.projectDir, "third-party/minijail/minijail0")
	dest := filepath.Join(i.binDir(), "minijail0")
	err = fileutil.CopyFile(src, dest, 0700)
	if err != nil {
		return err
	}
	src = filepath.Join(i.projectDir, "third-party/minijail/libminijailpreload.so")
	dest = filepath.Join(i.libDir(), "libminijailpreload.so")
	err = fileutil.CopyFile(src, dest, 0600)
	if err != nil {
		return err
	}

	return nil
}

func (i *installer) InstallProcessWrapper() error {
	compiler := os.Getenv("CC")
	if compiler == "" {
		compiler = "clang"
	}
	dest := filepath.Join(i.libDir(), "process_wrapper")
	cmd := exec.Command(compiler, "-o", dest, "process_wrapper.c")
	cmd.Dir = filepath.Join(i.projectDir, "pkg/minijail/process_wrapper/src")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	log.Printf("Command: %s", cmd.String())
	err := cmd.Run()
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (i *installer) InstallCIFuzz() error {
	// Build and install cifuzz
	dest := filepath.Join(i.binDir(), "cifuzz")
	cmd := exec.Command("go", "build", "-o", dest, "cmd/cifuzz/main.go")
	cmd.Dir = i.projectDir
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	log.Printf("Command: %s", cmd.String())
	err := cmd.Run()
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (i *installer) InstallCMakeIntegration() error {
	if runtime.GOOS != "windows" && os.Getuid() == 0 {
		// On non-Windows systems, CMake doesn't have the concept of a system
		// package registry. Instead, install the package into a well-known
		// path
		// See:
		// https://cmake.org/cmake/help/latest/command/find_package.html#config-mode-search-procedure
		// https://gitlab.kitware.com/cmake/cmake/-/blob/5ed9232d781ccfa3a9fae709e12999c6649aca2f/Modules/Platform/UnixPaths.cmake#L30)
		_, err := i.copyCMakeIntegration("/usr/local")
		if err != nil {
			return err
		}
	}
	dirForRegistry, err := i.copyCMakeIntegration(filepath.Join(i.shareDir(), "cmake"))
	if err != nil {
		return err
	}
	return registerCMakePackage(dirForRegistry)
}

func (i *installer) PrintPathInstructions() {
	if runtime.GOOS == "windows" {
		// TODO: On Windows, users generally don't expect having to fiddle with their PATH. We should update it for
		//       them, but that requires asking for admin access.
		fmt.Fprintf(os.Stderr, `
Please add the following directory to your PATH:
    %s
`, i.binDir())
	} else {
		fmt.Fprintf(os.Stderr, `
Please add the following to ~/.profile or ~/.bash_profile:
    export PATH="$PATH:%s"
`, i.binDir())
	}
}

func findProjectDir() (string, error) {
	// Find the project root directory
	projectDir, err := os.Getwd()
	if err != nil {
		return "", errors.WithStack(err)
	}
	exists, err := fileutil.Exists(filepath.Join(projectDir, "go.mod"))
	if err != nil {
		return "", errors.WithStack(err)
	}
	for !exists {
		if filepath.Dir(projectDir) == projectDir {
			return "", errors.Errorf("Couldn't find project root directory")
		}
		projectDir = filepath.Dir(projectDir)
		exists, err = fileutil.Exists(filepath.Join(projectDir, "go.mod"))
		if err != nil {
			return "", errors.WithStack(err)
		}
	}
	return projectDir, nil
}

// copyCMakeIntegration copies the CMake integration to destDir and returns the
// path that should be registered with the CMake package registry, if needed on
// the platform.
// Directories are created as needed.
func (i *installer) copyCMakeIntegration(destDir string) (string, error) {
	cmakeSrc := filepath.Join(i.projectDir, "tools", "cmake", "CIFuzz")
	cmakeDst := filepath.Join(destDir, "CIFuzz")
	opts := copy.Options{
		// Skip copying the replayer, which is a symlink on UNIX but a file
		// containing the relative path on Windows. It is handled below.
		OnSymlink: func(string) copy.SymlinkAction {
			return copy.Skip
		},
	}
	err := copy.Copy(cmakeSrc, cmakeDst, opts)
	if err != nil {
		return "", errors.WithStack(err)
	}

	// Copy the replayer, which is a symlink and thus may not have been copied
	// correctly on Windows.
	replayerSrc := filepath.Join(i.projectDir, "tools", "replayer", "src", "replayer.c")
	replayerDir := filepath.Join(cmakeDst, "src")
	err = os.MkdirAll(replayerDir, 0755)
	if err != nil {
		return "", errors.WithStack(err)
	}
	replayerDst := filepath.Join(replayerDir, "replayer.c")
	err = copy.Copy(replayerSrc, replayerDst)
	if err != nil {
		return "", errors.WithStack(err)
	}

	// The CMake package registry entry has to point directly to the directory
	// containing the CIFuzzConfig.cmake file rather than any valid prefix for
	// the config mode search procedure.
	return filepath.Join(cmakeDst, "share", "CIFuzz"), nil
}
