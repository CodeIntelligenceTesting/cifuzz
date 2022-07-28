package install

import (
	"embed"
	"fmt"
	"io/fs"
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

type InstallationBundler struct {
	Version   string
	TargetDir string

	projectDir string
	mutex      *filemutex.FileMutex
	isLocked   bool
}

type Options struct {
	Version   string
	TargetDir string
}

func NewInstallationBundler(opts Options) (*InstallationBundler, error) {
	var err error

	// Validate options
	if opts.Version == "" {
		return nil, err
	}
	opts.TargetDir, err = validateTargetDir(opts.TargetDir)
	if err != nil {
		return nil, err
	}

	projectDir, err := FindProjectDir()
	if err != nil {
		return nil, err
	}

	i := &InstallationBundler{
		TargetDir:  opts.TargetDir,
		Version:    opts.Version,
		projectDir: projectDir,
	}

	i.mutex, err = filemutex.New(i.lockFile())
	if err != nil {
		return nil, errors.WithStack(err)
	}

	err = i.createDirectoryLayout()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	log.Printf("Building cifuzz in %v", opts.TargetDir)

	return i, nil
}

func (i *InstallationBundler) createDirectoryLayout() error {
	err := os.MkdirAll(i.binDir(), 0755)
	if err != nil {
		i.Cleanup()
		return errors.WithStack(err)
	}
	err = os.MkdirAll(i.libDir(), 0755)
	if err != nil {
		i.Cleanup()
		return errors.WithStack(err)
	}
	err = os.MkdirAll(i.shareDir(), 0755)
	if err != nil {
		i.Cleanup()
		return errors.WithStack(err)
	}

	return nil
}

func (i *InstallationBundler) binDir() string {
	return filepath.Join(i.TargetDir, "bin")
}

func (i *InstallationBundler) libDir() string {
	return filepath.Join(i.TargetDir, "lib")
}

func (i *InstallationBundler) shareDir() string {
	return filepath.Join(i.TargetDir, "share", "cifuzz")
}

func (i *InstallationBundler) lockFile() string {
	return filepath.Join(i.projectDir, ".installer-lock")
}

func (i *InstallationBundler) Cleanup() {
	fileutil.Cleanup(i.TargetDir)
	// Always remove the lock file, even if SKIP_CLEANUP is set, because
	// keeping it around is not useful for debugging purposes.
	_ = os.Remove(i.lockFile())
}

// Lock acquires a file lock to make sure that only one instance of the
// installer is executed at the same time. Note that this function does
// not provide thread-safety for using the same installer instance
// multiple times.
func (i *InstallationBundler) Lock() error {
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
func (i *InstallationBundler) Unlock() error {
	err := i.mutex.Unlock()
	if err != nil {
		return errors.WithStack(err)
	}
	i.isLocked = false
	return nil
}

func (i *InstallationBundler) BuildCIFuzzAndDeps() error {
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
		err = i.BuildMinijail()
		if err != nil {
			return err
		}

		err = i.BuildProcessWrapper()
		if err != nil {
			return err
		}
	}

	err = i.BuildCIFuzz()
	if err != nil {
		return err
	}

	err = i.CopyCMakeIntegration()
	if err != nil {
		return err
	}

	return nil
}

func (i *InstallationBundler) BuildMinijail() error {
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

	// Copy minijail binaries
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

func (i *InstallationBundler) BuildProcessWrapper() error {
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

	// Build process wrapper
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

func (i *InstallationBundler) BuildCIFuzz() error {
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

	// Build cifuzz
	ldFlags := fmt.Sprintf("-ldflags=-X code-intelligence.com/cifuzz/internal/cmd/root.version=%s", i.Version)
	cmd := exec.Command("go", "build", "-o", CIFuzzExecutablePath(i.binDir()), ldFlags, "cmd/cifuzz/main.go")
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

// CopyCMakeIntegration copies the CMake integration to shareDir.
// Directories are created as needed.
func (i *InstallationBundler) CopyCMakeIntegration() error {
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

	cmakeSrc := filepath.Join(i.projectDir, "tools", "cmake", "cifuzz")
	destDir := i.shareDir()
	opts := copy.Options{
		// Skip copying the replayer, which is a symlink on UNIX but checked out
		// by git as a file containing the relative path on Windows. It is
		// handled below.
		OnSymlink: func(string) copy.SymlinkAction {
			return copy.Skip
		},
	}
	err = copy.Copy(cmakeSrc, destDir, opts)
	if err != nil {
		return errors.WithStack(err)
	}

	// Copy the replayer, which is a symlink and thus may not have been copied
	// correctly on Windows.
	replayerSrc := filepath.Join(i.projectDir, "tools", "replayer", "src", "replayer.c")
	replayerDir := filepath.Join(destDir, "src")
	err = os.MkdirAll(replayerDir, 0755)
	if err != nil {
		return errors.WithStack(err)
	}
	err = copy.Copy(replayerSrc, filepath.Join(replayerDir, "replayer.c"))
	if err != nil {
		return errors.WithStack(err)
	}
	err = copy.Copy(replayerSrc, filepath.Join(replayerDir, "replayer.cpp"))
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func CIFuzzExecutablePath(binDir string) string {
	path := filepath.Join(binDir, "cifuzz")
	if runtime.GOOS == "windows" {
		path += ".exe"
	}
	return path
}

func PrintPathInstructions(binDir string) {
	if runtime.GOOS == "windows" {
		// TODO: On Windows, users generally don't expect having to fiddle with their PATH. We should update it for
		//       them, but that requires asking for admin access.
		_, _ = fmt.Fprintf(os.Stderr, `Please add the following directory to your PATH:
    %s
`, binDir)
	} else {
		_, _ = fmt.Fprintf(os.Stderr, `Please add the following to ~/.profile or ~/.bash_profile:
    export PATH="$PATH:%s"
`, binDir)
	}
}

// ExtractBundle extracts all installation files from bundle into targetDir and registers the CMake package
func ExtractBundle(targetDir string, bundle *embed.FS) error {
	// List of files which have to be made executable
	executableFiles := []string{
		"bin/cifuzz",
		"bin/minijail0",
		"lib/libminijailpreload.so",
		"lib/process_wrapper",
	}

	targetDir, err := validateTargetDir(targetDir)
	if err != nil {
		return err
	}

	bundleFs, err := fs.Sub(bundle, "bundle")
	if err != nil {
		return errors.WithStack(err)
	}

	// Extract files in bundle
	err = fs.WalkDir(bundleFs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			targetDir := filepath.Dir(filepath.Join(targetDir, path))
			err = os.MkdirAll(targetDir, 0755)
			if err != nil {
				return errors.WithStack(err)
			}

			content, err := fs.ReadFile(bundleFs, path)
			if err != nil {
				return errors.WithStack(err)
			}

			fileName := filepath.Join(targetDir, d.Name())
			err = os.WriteFile(fileName, content, 0644)
			if err != nil {
				return errors.WithStack(err)
			}

			// Make required files executable
			for _, executableFile := range executableFiles {
				if executableFile == path {
					err = os.Chmod(fileName, 0755)
					if err != nil {
						return errors.WithStack(err)
					}
				}
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	if runtime.GOOS != "windows" && os.Getuid() == 0 {
		// On non-Windows systems, CMake doesn't have the concept of a system
		// package registry. Instead, install the package into the well-known
		// prefix /usr/local using the following relative search path:
		// /(lib/|lib|share)/<name>*/(cmake|CMake)/
		// See:
		// https://cmake.org/cmake/help/latest/command/find_package.html#config-mode-search-procedure
		// https://gitlab.kitware.com/cmake/cmake/-/blob/5ed9232d781ccfa3a9fae709e12999c6649aca2f/Modules/Platform/UnixPaths.cmake#L30)
		cmakeSrc := filepath.Join(targetDir, "share", "cifuzz")
		cmakeDest := "/usr/local/share/cifuzz"
		err = copy.Copy(cmakeSrc, cmakeDest)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	// The CMake package registry entry has to point directly to the directory
	// containing the CIFuzzConfig.cmake file rather than any valid prefix for
	// the config mode search procedure.
	dirForRegistry := filepath.Join(targetDir, "share", "cifuzz", "cmake")
	return registerCMakePackage(dirForRegistry)
}

func FindProjectDir() (string, error) {
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

func validateTargetDir(installDir string) (string, error) {
	var err error

	if strings.HasPrefix(installDir, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", errors.WithStack(err)
		}
		installDir = home + strings.TrimPrefix(installDir, "~")
	}

	if installDir == "" {
		installDir, err = os.MkdirTemp("", "cifuzz-install-dir-")
		if err != nil {
			return "", errors.WithStack(err)
		}
	} else {
		installDir, err = filepath.Abs(installDir)
		if err != nil {
			return "", errors.WithStack(err)
		}

		exists, err := fileutil.Exists(installDir)
		if err != nil {
			return "", err
		}
		if exists {
			return "", errors.Errorf("Install directory '%s' already exists. Please remove it to continue.", installDir)
		}
	}

	return installDir, nil
}
