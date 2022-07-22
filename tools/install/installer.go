package install

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/alexflint/go-filemutex"
	"github.com/otiai10/copy"
	"github.com/pkg/errors"

	"code-intelligence.com/cifuzz/util/envutil"
	"code-intelligence.com/cifuzz/util/fileutil"
)

type Installer struct {
	InstallDir string

	projectDir string
	mutex      *filemutex.FileMutex
	isLocked   bool
}

type Options struct {
	InstallDir string
}

func NewInstaller(opts *Options) (*Installer, error) {
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
		opts.InstallDir, err = os.MkdirTemp("", "cifuzz-install-dir-")
		if err != nil {
			return nil, errors.WithStack(err)
		}
	} else {
		opts.InstallDir, err = filepath.Abs(opts.InstallDir)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		exists, err := fileutil.Exists(opts.InstallDir)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		if exists {
			return nil, errors.Errorf("Install directory '%s' already exists. Please remove it to continue.", opts.InstallDir)
		}
	}
	log.Printf("Installing cifuzz to %v", opts.InstallDir)

	i := &Installer{
		projectDir: projectDir,
		InstallDir: opts.InstallDir,
	}

	i.mutex, err = filemutex.New(i.lockFile())
	if err != nil {
		return nil, errors.WithStack(err)
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

func (i *Installer) binDir() string {
	return filepath.Join(i.InstallDir, "bin")
}

func (i *Installer) libDir() string {
	return filepath.Join(i.InstallDir, "lib")
}

func (i *Installer) shareDir() string {
	return filepath.Join(i.InstallDir, "share", "cifuzz")
}

func (i *Installer) lockFile() string {
	return filepath.Join(i.projectDir, ".installer-lock")
}

func (i *Installer) CIFuzzExecutablePath() string {
	path := filepath.Join(i.binDir(), "cifuzz")
	if runtime.GOOS == "windows" {
		path += ".exe"
	}
	return path
}

func (i *Installer) Cleanup() {
	fileutil.Cleanup(i.InstallDir)
	// Always remove the lock file, even if SKIP_CLEANUP is set, because
	// keeping it around is not useful for debugging purposes.
	_ = os.Remove(i.lockFile())
}

// Lock acquires a file lock to make sure that only one instance of the
// installer is executed at the same time. Note that this function does
// not provide thread-safety for using the same installer instance
// multiple times.
func (i *Installer) Lock() error {
	if i.isLocked {
		return nil
	}
	err := i.mutex.Lock()
	if err != nil {
		return errors.WithStack(err)
	}
	i.isLocked = true
	return nil

}

// Unlock releases the file lock to allow other installer instances to
// run.
func (i *Installer) Unlock() error {
	err := i.mutex.Unlock()
	if err != nil {
		return errors.WithStack(err)
	}
	i.isLocked = false
	return nil
}

func (i *Installer) InstallCIFuzzAndDeps() error {
	var err error

	err = i.Lock()
	if err != nil {
		return err
	}
	defer func() {
		err = i.Unlock()
		if err != nil {
			log.Printf("error: %v", err)
		}
	}()

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

func (i *Installer) InstallMinijail() error {
	var err error

	err = i.Lock()
	if err != nil {
		return err
	}
	defer func() {
		err = i.Unlock()
		if err != nil {
			log.Printf("error: %v", err)
		}
	}()

	minijailDir := filepath.Join(i.projectDir, "third-party", "minijail")

	// Build minijail
	cmd := exec.Command("make", "CC_BINARY(minijail0)", "CC_LIBRARY(libminijailpreload.so)")
	cmd.Dir = minijailDir
	// The minijail Makefile changes the directory to $PWD, so we have
	// to set that.
	cmd.Env, err = envutil.Setenv(os.Environ(), "PWD", filepath.Join(i.projectDir, "third-party", "minijail"))
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
	src := filepath.Join(i.projectDir, "third-party", "minijail", "minijail0")
	dest := filepath.Join(i.binDir(), "minijail0")
	err = copy.Copy(src, dest)
	if err != nil {
		return errors.WithStack(err)
	}
	src = filepath.Join(i.projectDir, "third-party", "minijail", "libminijailpreload.so")
	dest = filepath.Join(i.libDir(), "libminijailpreload.so")
	err = copy.Copy(src, dest)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (i *Installer) InstallProcessWrapper() error {
	var err error
	err = i.Lock()
	if err != nil {
		return err
	}
	defer func() {
		err = i.Unlock()
		if err != nil {
			log.Printf("error: %v", err)
		}
	}()

	compiler := os.Getenv("CC")
	if compiler == "" {
		compiler = "clang"
	}
	dest := filepath.Join(i.libDir(), "process_wrapper")
	cmd := exec.Command(compiler, "-o", dest, "process_wrapper.c")
	cmd.Dir = filepath.Join(i.projectDir, "pkg", "minijail", "process_wrapper", "src")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	log.Printf("Command: %s", cmd.String())
	err = cmd.Run()
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (i *Installer) InstallCIFuzz() error {
	var err error
	err = i.Lock()
	if err != nil {
		return err
	}
	defer func() {
		err = i.Unlock()
		if err != nil {
			log.Printf("error: %v", err)
		}
	}()

	// Build and install cifuzz
	cmd := exec.Command("go", "build", "-o", i.CIFuzzExecutablePath(), "cmd/cifuzz/main.go")
	cmd.Dir = i.projectDir
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	log.Printf("Command: %s", cmd.String())
	err = cmd.Run()
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (i *Installer) InstallCMakeIntegration() error {
	var err error
	err = i.Lock()
	if err != nil {
		return err
	}
	defer func() {
		err = i.Unlock()
		if err != nil {
			log.Printf("error: %v", err)
		}
	}()

	if runtime.GOOS != "windows" && os.Getuid() == 0 {
		// On non-Windows systems, CMake doesn't have the concept of a system
		// package registry. Instead, install the package into the well-known
		// prefix /usr/local using the following relative search path:
		// /(lib/|lib|share)/<name>*/(cmake|CMake)/
		// See:
		// https://cmake.org/cmake/help/latest/command/find_package.html#config-mode-search-procedure
		// https://gitlab.kitware.com/cmake/cmake/-/blob/5ed9232d781ccfa3a9fae709e12999c6649aca2f/Modules/Platform/UnixPaths.cmake#L30)
		_, err := i.copyCMakeIntegration("/usr/local/share/cifuzz")
		if err != nil {
			return err
		}
	}
	dirForRegistry, err := i.copyCMakeIntegration(i.shareDir())
	if err != nil {
		return err
	}
	return registerCMakePackage(dirForRegistry)
}

func (i *Installer) PrintPathInstructions() {
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
func (i *Installer) copyCMakeIntegration(destDir string) (string, error) {
	cmakeSrc := filepath.Join(i.projectDir, "tools", "cmake", "cifuzz")
	opts := copy.Options{
		// Skip copying the replayer, which is a symlink on UNIX but checked out
		// by git as a file containing the relative path on Windows. It is
		// handled below.
		OnSymlink: func(string) copy.SymlinkAction {
			return copy.Skip
		},
	}
	err := copy.Copy(cmakeSrc, destDir, opts)
	if err != nil {
		return "", errors.WithStack(err)
	}

	// Copy the replayer, which is a symlink and thus may not have been copied
	// correctly on Windows.
	replayerSrc := filepath.Join(i.projectDir, "tools", "replayer", "src", "replayer.c")
	replayerDir := filepath.Join(destDir, "src")
	err = os.MkdirAll(replayerDir, 0755)
	if err != nil {
		return "", errors.WithStack(err)
	}
	err = copy.Copy(replayerSrc, filepath.Join(replayerDir, "replayer.c"))
	if err != nil {
		return "", errors.WithStack(err)
	}
	err = copy.Copy(replayerSrc, filepath.Join(replayerDir, "replayer.cpp"))
	if err != nil {
		return "", errors.WithStack(err)
	}

	// The CMake package registry entry has to point directly to the directory
	// containing the CIFuzzConfig.cmake file rather than any valid prefix for
	// the config mode search procedure.
	return filepath.Join(destDir, "cmake"), nil
}
