package envutil

import (
	"os"
	"strings"

	"github.com/pkg/errors"

	"code-intelligence.com/cifuzz/util/sliceutil"
	"code-intelligence.com/cifuzz/util/stringutil"
)

const sep = string(os.PathListSeparator)

// AppendToPathList appends a string to another string containing a list
// of paths, separated by os.PathListSeparator (like the PATH and
// LD_LIBRARY_PATH environment variables). It doesn't add duplicates and
// removes any empty strings from the list.
func AppendToPathList(list string, value ...string) string {
	if len(value) == 0 {
		return list
	}

	values := strings.Split(list, sep)

	for _, newVal := range value {
		if !sliceutil.Contains(values, newVal) {
			values = append(values, newVal)
		}
	}

	return stringutil.JoinNonEmpty(values, sep)
}

// Like os.LookupEnv but uses the specified environment instead of the
// current process environment.
func LookupEnv(env []string, key string) (string, bool) {
	envMap := ToMap(env)
	val, ok := envMap[key]
	return val, ok
}

// Like os.Getenv but uses the specified environment instead of the
// current process environment.
func Getenv(env []string, key string) string {
	envMap := ToMap(env)
	return envMap[key]
}

// Like os.Setenv but uses the specified environment instead of the
// current process environment.
func Setenv(env []string, key, value string) ([]string, error) {
	if strings.ContainsAny(key, "="+"\x00") {
		return nil, errors.Errorf("invalid key: %q", key)
	}

	if strings.ContainsRune(value, '\x00') {
		return nil, errors.Errorf("invalid value: %q", value)
	}

	kv := key + "=" + value

	// Check if the key is already set
	prefix := key + "="
	for i, e := range env {
		if strings.HasPrefix(e, prefix) {
			// Replace the value
			env[i] = kv
			return env, nil
		}
	}

	// The key is not set yet, append it
	env = append(env, kv)
	return env, nil
}

// ToMap converts the specified strings representing an environment in
// the form "key=value" to a map.
func ToMap(env []string) map[string]string {
	res := make(map[string]string)
	for _, e := range env {
		s := strings.SplitN(e, "=", 2)
		if len(s) != 2 {
			continue
		}
		key, val := s[0], s[1]
		res[key] = val
	}
	return res
}
