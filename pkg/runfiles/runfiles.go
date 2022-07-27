package runfiles

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pkg/errors"
)

type RunfilesFinder interface {
	CIFuzzIncludePath() (string, error)
	ClangPath() (string, error)
	JazzerAgentDeployJarPath() (string, error)
	JazzerDriverPath() (string, error)
	LibMinijailPreloadPath() (string, error)
	LLVMCovPath() (string, error)
	LLVMProfDataPath() (string, error)
	LLVMSymbolizerPath() (string, error)
	Minijail0Path() (string, error)
	ProcessWrapperPath() (string, error)
	ReplayerSourcePath() (string, error)
}

var Finder RunfilesFinder

func init() {
	// Set the default runfiles finder.
	//
	// If the environment variable CIFUZZ_INSTALL_ROOT is set, we use
	// that as the installation directory, else we assume that the
	// current executable lives in $INSTALL_DIR/bin, so we go up one
	// directory from there and use that as the installation directory.
	installDir, found := os.LookupEnv("CIFUZZ_INSTALL_ROOT")
	if !found || installDir == "" {
		executablePath, err := os.Executable()
		if err != nil {
			panic(errors.WithStack(err))
		}

		installDir, err = filepath.Abs(filepath.Join(filepath.Dir(executablePath), ".."))
		if err != nil {
			panic(errors.WithStack(err))
		}
	}

	Finder = RunfilesFinderImpl{InstallDir: installDir}
}

// FindSystemJavaHome returns the absolute path to the base directory of the
// default system JDK/JRE. It first looks up JAVA_HOME and then falls back to
// using the java binary in the PATH.
func FindSystemJavaHome() (string, error) {
	javaHome := os.Getenv("JAVA_HOME")
	if javaHome != "" {
		return javaHome, nil
	}

	if runtime.GOOS == "darwin" {
		// On some macOS installations, an executable 'java_home' exists
		// which prints the JAVA_HOME of the default installation to stdout
		var outbuf bytes.Buffer
		cmd := exec.Command("/usr/libexec/java_home")
		cmd.Stdout = &outbuf
		err := cmd.Run()
		if err == nil {
			return strings.TrimSpace(outbuf.String()), nil
		}
	}

	javaSymlink, err := exec.LookPath("java")
	if err != nil {
		return "", errors.WithStack(err)
	}
	// The java binary in the PATH, e.g. at /usr/bin/java, is typically a
	// symlink pointing to the actual java binary in the bin subdirectory of the
	// JAVA_HOME.
	javaBinary, err := filepath.EvalSymlinks(javaSymlink)
	if err != nil {
		return "", errors.WithStack(err)
	}
	absoluteJavaBinary, err := filepath.Abs(javaBinary)
	if err != nil {
		return "", errors.WithStack(err)
	}
	return filepath.Dir(filepath.Dir(absoluteJavaBinary)), nil
}

func JazzerDriverPath() (string, error) {
	return Finder.JazzerDriverPath()
}
