package minijail

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"code-intelligence.com/cifuzz/pkg/log"
	"code-intelligence.com/cifuzz/pkg/runfiles"
	"code-intelligence.com/cifuzz/util/envutil"
	"code-intelligence.com/cifuzz/util/fileutil"
	"code-intelligence.com/cifuzz/util/stringutil"
)

const (
	OutputDir = "/tmp/minijail-out"

	EnvPrefix = "CIFUZZ_MINIJAIL_"

	DebugEnvVarName    = EnvPrefix + "DEBUG"
	BindingsEnvVarName = EnvPrefix + "BINDINGS"

	BindingFlag = "bind"
	EnvFlag     = "env"

	// Mount flags as defined in golang.org/x/sys/unix. We're not using
	// that package because it's not available on macOS.
	MS_RDONLY      = 0x1
	MS_NOSUID      = 0x2
	MS_NODEV       = 0x4
	MS_STRICTATIME = 0x1000000
)

type WritableOption int

const (
	ReadOnly WritableOption = iota
	ReadWrite
)

type Binding struct {
	Source   string
	Target   string
	Writable WritableOption
}

func (b *Binding) String() string {
	if b.Target == "" {
		b.Target = b.Source
	}
	if b.Writable == ReadWrite {
		return fmt.Sprintf("%s,%s,1", b.Source, b.Target)
	}
	// Don't use a short form if the source or target contain a comma,
	// which would be interpreted as separators by minijail.
	if strings.ContainsRune(b.Source, ',') || strings.ContainsRune(b.Target, ',') {
		return fmt.Sprintf("%s,%s,0", b.Source, b.Target)
	}
	if b.Source != b.Target {
		return fmt.Sprintf("%s,%s", b.Source, b.Target)
	}
	return b.Source
}

func BindingFromString(s string) (*Binding, error) {
	tokens := strings.SplitN(s, ",", 3)
	switch len(tokens) {
	case 1:
		return &Binding{Source: tokens[0], Target: tokens[0], Writable: 0}, nil
	case 2:
		return &Binding{Source: tokens[0], Target: tokens[1], Writable: 0}, nil
	case 3:
		writable, err := strconv.Atoi(tokens[2])
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &Binding{Source: tokens[0], Target: tokens[1], Writable: WritableOption(writable)}, nil
	}
	return nil, errors.Errorf("Bad binding: %s", s)
}

// Deprecated: Use AddMinijailBindingToEnv instead, which doesn't use os.Setenv.
// TODO(adrian): AddMinijailBindingDeprecated will be removed once all adapters are
//               rewritten (CIFUZZ-1289).
func AddMinijailBindingDeprecated(path string, writable WritableOption) error {
	binding, err := getMinijailBinding(path, writable)
	if err != nil {
		return err
	}

	bindings := os.Getenv(BindingsEnvVarName)
	if bindings == "" {
		bindings = binding
	} else {
		bindings += ":" + binding
	}

	err = os.Setenv(BindingsEnvVarName, bindings)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func AddMinijailBindingToEnv(env []string, binding *Binding) ([]string, error) {
	bindings := envutil.Getenv(env, BindingsEnvVarName)
	if bindings == "" {
		bindings = binding.Source
	} else {
		bindings += ":" + binding.String()
	}

	env, err := envutil.Setenv(env, BindingsEnvVarName, bindings)
	if err != nil {
		return nil, err
	}

	return env, nil
}

func getMinijailBinding(path string, writable WritableOption) (string, error) {
	src, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", errors.WithStack(err)
	}
	writableStr := "0"
	if writable == ReadWrite {
		writableStr = "1"
	}
	return fmt.Sprintf("%s,%s,%s", src, path, writableStr), nil
}

var fixedMinijailArgs = []string{
	// Most of these args are the same as the ones clusterfuzz sets in
	// their minijail wrapper:
	// https://github.com/google/clusterfuzz/blob/4f8020c4c7ce73c1da0e68f04943af30bb5f0b32/src/clusterfuzz/_internal/system/minijail.py
	//
	// In contrast to clusterfuzz, we don't set "-T static", but do
	// preload libminijailpreload.so, to prevent circumvention of the
	// seccomp filters, as described in the minijail manual:
	// https://google.github.io/minijail/minijail0.1.html
	//
	"-U", "-m", // Quote from clusterfuzz:
	// root (uid 0 in namespace) -> USER.
	// The reason for this is that minijail does setresuid(0, 0, 0) before doing a
	// chroot, which means uid 0 needs access to the chroot dir (owned by USER).
	//
	// Note that we also run fuzzers as uid 0 (but with no capabilities in
	// permitted/effective/inherited sets which *should* mean there"s nothing
	// special about it). This is because the uid running the fuzzer also need
	// access to things owned by USER (fuzzer binaries, supporting files), and USER
	// can only be mapped once.
	"-M",      // Map current gid to root
	"-c", "0", // drop all capabilities.
	"-n", // no_new_privs
	"-v", // mount namespace
	"-p", // PID namespace
	"-l", // IPC namespace
	"-I", // Run jailed process as init.
	// Mount procfs read-only
	"-k", "proc,/proc,proc," + strconv.Itoa(MS_RDONLY),
	// Mount a tmpfs on /dev/shm to allow using shared memory.
	"-k", "tmpfs,/dev/shm,tmpfs," + strconv.Itoa(MS_NOSUID|MS_NODEV|MS_STRICTATIME) + ",mode=1777",
	// Added by us, to log to stderr
	"--logging=stderr",
}

var defaultBindings = []*Binding{
	// The second value specifies whether the binding is read-only (0)
	// or read-write (1).
	{Source: "/lib"},
	{Source: "/lib32"},
	{Source: "/lib64"},
	{Source: "/usr/lib"},
	{Source: "/usr/lib32"},
	// Added by us
	{Source: OutputDir, Writable: ReadWrite},
	// We allow access to /dev/null and /dev/urandom because AFL needs
	// access to them and some fuzz targets might as well (for example
	// our lighttpd example fuzz target).
	// They have to be mounted read-write, else minijail fails with
	// libminijail[1]: cannot bind-remount: [...] Operation not permitted
	{Source: "/dev/null", Writable: ReadWrite},
	{Source: "/dev/urandom", Writable: ReadWrite},
	// We allow access to /etc/passwd and /etc/group because some fuzz
	// targets (for example nginx) will fail if they can't obtain the
	// UID and GID for a specified user and group (specifying a UID
	// instead doesn't seem to be supported by nginx).
	{Source: "/etc/passwd"},
	{Source: "/etc/group"},
}

type Options struct {
	Args     []string
	Env      []string
	Bindings []*Binding
}

type minijail struct {
	*Options
	Args      []string
	chrootDir string
}

func NewMinijail(opts *Options) (*minijail, error) {
	// Evaluate symlinks in the executable path
	path, err := filepath.EvalSymlinks(opts.Args[0])
	if err != nil {
		return nil, errors.WithStack(err)
	}
	opts.Args[0] = path

	// --------------------------
	// --- Create directories ---
	// --------------------------
	// Create chroot directory
	chrootDir, err := ioutil.TempDir("", "minijail-chroot-")
	if err != nil {
		return nil, err
	}
	// TODO: Return a cleanup function
	//defer fileutil.Cleanup(chrootDir)

	// Create output directory
	err = os.MkdirAll(OutputDir, 0700)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// Create /tmp, /proc directories.
	for _, dir := range []string{"/proc", "/tmp"} {
		err = os.MkdirAll(filepath.Join(chrootDir, dir), 0o755)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	// Create /dev/shm which is required to allow using shared memory
	err = os.MkdirAll(filepath.Join(chrootDir, "/dev/shm"), 0o755)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// ----------------------------
	// --- Set up minijail args ---
	// ----------------------------
	minijailPath, err := runfiles.Finder.Minijail0Path()
	if err != nil {
		return nil, err
	}
	minijailArgs := append([]string{minijailPath}, fixedMinijailArgs...)

	if os.Getenv(DebugEnvVarName) != "" {
		log.Warn("Running minijail in debug mode, this is NOT SAFE FOR PRODUCTION!")
		// This causes minijail to not use preload hooking, which
		// sometimes results in better error messages, so it can be
		// useful for debugging but shouldn't be used in production,
		// because (quoting the Minijail manual [1]): "some jailing can
		// only be achieved from the process to which they will actually
		// apply [via preloading]".
		// [1] https://google.github.io/minijail/minijail0.1.html#implementation
		minijailArgs = append(minijailArgs, "-T", "static", "--ambient")
	} else {
		libminijailpreload, err := runfiles.Finder.LibMinijailPreloadPath()
		if err != nil {
			return nil, err
		}
		minijailArgs = append(minijailArgs, "--preload-library="+libminijailpreload)
	}

	// Change root filesystem to the chroot directory. See pivot_root(2).
	minijailArgs = append(minijailArgs, "-P", chrootDir)

	// -----------------------
	// --- Set up bindings ---
	// -----------------------
	bindings := append(opts.Bindings, defaultBindings...)

	// We expect the current working directory to be the artifacts
	// directory, which should be accessible to the fuzz target, so we
	// add a binding for it.
	// Some fuzz targets (e.g. the one for nginx) write to the working
	// directory, which is why we mount it read-write. We decided that
	// this is fine on CIFUZZ-1192.
	workdir, err := os.Getwd()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	bindings = append(bindings, &Binding{Source: workdir, Writable: ReadWrite})

	// Add binding for the executable
	bindings = append(bindings, &Binding{Source: path})

	// Add llvm to bindings
	llvmSymbolizerPath, err := runfiles.Finder.LLVMSymbolizerPath()
	if err != nil {
		return nil, err
	}
	llvmDir := filepath.Dir(filepath.Dir(llvmSymbolizerPath))
	bindings = append(bindings, &Binding{Source: llvmDir})

	// Add binding for process_wrapper. process_wrapper changes the
	// working directory and sets environment variables and then
	// executes the specified command.
	processWrapperPath, err := runfiles.Finder.ProcessWrapperPath()
	if err != nil {
		return nil, err
	}
	bindings = append(bindings, &Binding{Source: processWrapperPath})

	// Add additional bindings from the environment variable
	additionalBindingsEnv := os.Getenv(BindingsEnvVarName)
	for _, s := range strings.Split(additionalBindingsEnv, ":") {
		if s == "" {
			continue
		}
		binding, err := BindingFromString(s)
		if err != nil {
			return nil, err
		}
		bindings = append(bindings, binding)
	}

	// Create the bindings
	for _, binding := range bindings {
		if binding.Target == "" {
			binding.Target = binding.Source
		}
		// Skip if the source doesn't exist
		exists, err := fileutil.Exists(binding.Source)
		if err != nil {
			return nil, err
		}
		if !exists {
			continue
		}

		// Create the destination
		if fileutil.IsDir(binding.Source) {
			err = os.MkdirAll(filepath.Join(chrootDir, binding.Target), 0o755)
			if err != nil {
				return nil, errors.WithStack(err)
			}
		} else {
			err = os.MkdirAll(filepath.Join(chrootDir, filepath.Dir(binding.Target)), 0o755)
			if err != nil {
				return nil, errors.WithStack(err)
			}
			err = fileutil.Touch(filepath.Join(chrootDir, binding.Target))
			if err != nil {
				return nil, err
			}
		}

		minijailArgs = append(minijailArgs, "-b", binding.String())
	}

	// -----------------------------------
	// --- Set up process wrapper args ---
	// -----------------------------------
	// The process wrapper changes the working directory inside the
	// sandbox to the first argument
	processWrapperArgs := []string{processWrapperPath, workdir}

	// The process wrapper sets environment variables inside the sandbox
	// to the remaining arguments until the first "--".
	processWrapperArgs = append(processWrapperArgs, opts.Env...)

	// --------------------
	// --- Run minijail ---
	// --------------------
	args := stringutil.JoinSlices("--", minijailArgs, processWrapperArgs, opts.Args)

	// When CI_DEBUG_MINIJAIL_SLEEP_FOREVER is set, instead of executing
	// the actual command, we store it in the CMD environment variable
	// and start a shell to allow debugging issues interactively.
	if os.Getenv("CI_DEBUG_MINIJAIL_SLEEP_FOREVER") != "" {
		_ = os.MkdirAll(filepath.Join(chrootDir, "bin"), 0o755)
		minijailArgs = append(minijailArgs, "-b", "/bin")
		processWrapperArgs = append(processWrapperArgs, "CMD="+strings.Join(opts.Args, " "))
		args = stringutil.JoinSlices("--", minijailArgs, processWrapperArgs, []string{"/bin/sh"})
	}

	return &minijail{
		Options:   opts,
		chrootDir: chrootDir,
		Args:      args,
	}, nil
}

func (m *minijail) Cleanup() {
	fileutil.Cleanup(m.chrootDir)
}
