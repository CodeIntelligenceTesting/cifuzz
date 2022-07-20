# cifuzz configuration
You can change the behavior of **cifuzz** both via command-line flags
and via settings stored in the `cifuzz.yaml` config file. Flags take
precedence over the respective config file setting.

## cifuzz.yaml settings

[build-system](#build-system) <br/>
[build-command](#build-command) <br/>
[seed-corpus-dirs](#seed-corpus-dirs) <br/>
[dict](#dict) <br/>
[engine-args](#engine-args) <br/>
[fuzz-test-args](#fuzz-test-args) <br/>
[timeout](#timeout) <br/>
[use-sandbox](#use-sandbox) <br/>
[print-json](#print-json) <br/>

<a id="build-system"></a>

### build-system

The build system used to build this project. If not set, cifuzz tries
to detect the build system automatically.
Valid values: "cmake", "unknown".

#### Example

```yaml
build-system: cmake
```

<a id="build-command"></a>

### build-command

If the build system type is "unknown", this command is used by
`cifuzz run` to build the fuzz test.

#### Example

```yaml
build-command: "make all"
```

<a id="seed-corpus-dirs"></a>

### seed-corpus-dirs

Directories containing sample inputs for the code under test.
See https://llvm.org/docs/LibFuzzer.html#corpus and
https://aflplus.plus/docs/fuzzing_in_depth/#a-collecting-inputs.

#### Example

```yaml
seed-corpus-dirs:
 - path/to/seed-corpus
```

<a id="dict"></a>

### dict

A file containing input language keywords or other interesting byte
sequences. See https://llvm.org/docs/LibFuzzer.html#dictionaries and
https://github.com/AFLplusplus/AFLplusplus/blob/stable/dictionaries/README.md.

#### Example
```yaml
dict: path/to/dictionary.dct
```

<a id="engine-args"></a>

### engine-args
Command-line arguments to pass to the fuzzing engine (libFuzzer or
AFL++). See https://llvm.org/docs/LibFuzzer.html#options and
https://www.mankier.com/8/afl-fuzz.

#### Example
```yaml
engine-args:
 - -rss_limit_mb=4096
```

<a id="fuzz-test-args"></a>

### fuzz-test-args
Command-line arguments to pass to the fuzz tests.

#### Example
```yaml
fuzz-test-args:
 - --config-file=path/to/config
```

<a id="timeout"></a>

### timeout

Maximum time in seconds to run the fuzz tests. The default is to run
indefinitely.

#### Example
```yaml
timeout: 300
```

<a id="use-sandbox"></a>

### use-sandbox

By default, fuzz tests are executed in a sandbox to prevent accidental
damage to the system. Set to false to run fuzz tests unsandboxed.
Only supported on Linux.

#### Example
```yaml
use-sandbox: false
```

<a id="print-json"></a>

### print-json

Set to true to print output of the `cifuzz run` command as JSON.

#### Example
```yaml
print-json: true
```
