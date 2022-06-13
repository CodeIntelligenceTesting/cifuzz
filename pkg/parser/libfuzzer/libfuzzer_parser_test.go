package libfuzzer_output_parser

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code-intelligence.com/cifuzz/pkg/report"
	"code-intelligence.com/cifuzz/util/fileutil"
)

const maxBufferedReports = 10

func TestMain(m *testing.M) {
	viper.Set("verbose", true)
	flag.Parse()

	res := m.Run()
	os.Exit(res)
}

func TestLibFuzzerAdapter_ReportsParsing(t *testing.T) {
	testInputFile, err := ioutil.TempFile("", "testSlowInput-")
	require.NoError(t, err, "failed to create temp slow input file")
	testInput := []byte{'t', 'e', 's', 't'}
	_, err = testInputFile.Write(testInput)
	require.NoError(t, err)
	defer fileutil.Cleanup(testInputFile.Name())

	tests := []struct {
		name     string
		logs     string
		expected []*report.Report
	}{
		{
			name:     "empty logs",
			logs:     "",
			expected: []*report.Report{},
		},
		{
			name: "multiple coverage logs",
			logs: `
INFO: A corpus is not provided, starting from an empty corpus
#4749	NEW    cov: 6 ft: 4 corp: 3/8b exec/s: 10 rss: 47Mb L: 4/4 MS: 5 ChangeBit-InsertByte
#4805	REDUCE cov: 6 ft: 4 corp: 3/7b exec/s: 12 rss: 47Mb L: 3/3 MS: 1 EraseBytes-
#22045	REDUCE cov: 7 ft: 5 corp: 4/11b exec/s: 123 rss: 81Mb L: 4/4 MS 5 CopyPart-ChangeByte-
some invalid logs`,
			expected: []*report.Report{
				{Status: report.RunStatus_INITIALIZING},
				{
					Status: report.RunStatus_RUNNING,
					Metric: &report.FuzzingMetric{
						ExecutionsPerSecond:     10,
						Features:                4,
						Edges:                   6,
						CorpusSize:              3,
						TotalExecutions:         4749,
						SecondsSinceLastFeature: 0,
						SecondsSinceLastEdge:    0,
					},
				},
				{
					Status: report.RunStatus_RUNNING,
					Metric: &report.FuzzingMetric{
						ExecutionsPerSecond:     12,
						Features:                4,
						Edges:                   6,
						CorpusSize:              3,
						TotalExecutions:         4805,
						SecondsSinceLastFeature: 0,
						SecondsSinceLastEdge:    0,
					},
				},
				{
					Status: report.RunStatus_RUNNING,
					Metric: &report.FuzzingMetric{
						ExecutionsPerSecond: 123,
						Features:            5,
						Edges:               7,
						CorpusSize:          4,
						TotalExecutions:     22045,
					},
				},
			},
		},
		{
			name: "Progress and crash report",
			logs: `
INFO: A corpus is not provided, starting from an empty corpus
#4805	REDUCE cov: 6 ft: 4 corp: 3/7b exec/s: 10 rss: 47Mb L: 3/3 MS: 1 EraseBytes-
==8141==ERROR: AddressSanitizer: global-buffer-overflow on address 0x00
error info 1
error info 2`,
			expected: []*report.Report{
				{Status: report.RunStatus_INITIALIZING},
				{
					Status: report.RunStatus_RUNNING,
					Metric: &report.FuzzingMetric{
						ExecutionsPerSecond:     10,
						Features:                4,
						Edges:                   6,
						CorpusSize:              3,
						TotalExecutions:         4805,
						SecondsSinceLastFeature: 0,
						SecondsSinceLastEdge:    0,
					},
				},
				{
					Status: report.RunStatus_RUNNING,
					Finding: &report.Finding{
						Type:    report.ErrorType_CRASH,
						Details: "global-buffer-overflow on address 0x00",
						Logs: []string{
							"==8141==ERROR: AddressSanitizer: global-buffer-overflow on address 0x00",
							"error info 1",
							"error info 2",
						},
					},
				},
			},
		},
		{
			name: "UBSAN recoverable",
			logs: `
INFO: Seed: 2610909839
INFO: Loaded 1 modules   (3 inline 8-bit counters): 3 [0x629040, 0x629043),
INFO: Loaded 1 PC tables (3 PCs): 3 [0x629048,0x629078),
INFO: -max_len is not provided; libFuzzer will not generate inputs larger than 4096 bytes
INFO: A corpus is not provided, starting from an empty corpus
fuzz_targets/manual.cpp:6:5: runtime error: signed integer overflow: 2147483647 + 1 cannot be represented in type 'int'
SUMMARY: UndefinedBehaviorSanitizer: undefined-behavior fuzz_targets/manual.cpp:6:5 in
#2	INITED cov: 2 ft: 2 corp: 1/1b lim: 4 exec/s: 0 rss: 28Mb
#4194304	pulse  cov: 2 ft: 2 corp: 1/1b lim: 4096 exec/s: 1398101 rss: 359Mb
#8388608	pulse  cov: 2 ft: 2 corp: 1/1b lim: 4096 exec/s: 1398101 rss: 590Mb
==397442== libFuzzer: run interrupted; exiting`,
			expected: []*report.Report{
				{Status: report.RunStatus_INITIALIZING},
				{
					Status: report.RunStatus_RUNNING,
					Metric: &report.FuzzingMetric{
						ExecutionsPerSecond:     0,
						Features:                2,
						Edges:                   2,
						CorpusSize:              1,
						TotalExecutions:         2,
						SecondsSinceLastFeature: 0,
						SecondsSinceLastEdge:    0,
					},
				},
				{
					Status: report.RunStatus_RUNNING,
					Metric: &report.FuzzingMetric{
						ExecutionsPerSecond:     1398101,
						Features:                2,
						Edges:                   2,
						CorpusSize:              1,
						TotalExecutions:         4194304,
						SecondsSinceLastFeature: 0,
						SecondsSinceLastEdge:    0,
					},
				},
				{
					Status: report.RunStatus_RUNNING,
					Finding: &report.Finding{
						Type:    report.ErrorType_RUNTIME_ERROR,
						Details: "undefined behaviour: signed integer overflow",
						Logs: []string{
							"fuzz_targets/manual.cpp:6:5: runtime error: signed integer overflow: 2147483647 + 1 cannot be represented in type 'int'",
							"SUMMARY: UndefinedBehaviorSanitizer: undefined-behavior fuzz_targets/manual.cpp:6:5 in",
						},
					},
				},
				{
					Status: report.RunStatus_RUNNING,
					Metric: &report.FuzzingMetric{
						ExecutionsPerSecond:     1398101,
						Features:                2,
						Edges:                   2,
						CorpusSize:              1,
						TotalExecutions:         8388608,
						SecondsSinceLastFeature: 0,
						SecondsSinceLastEdge:    0,
					},
				},
			},
		},
		{
			name: "long operations warning",
			logs: "INFO: A corpus is not provided, starting from an empty corpus\n" +
				"#128 pulse cov: 4 ft: 2 corp: 27/1688b exec/s: 1 rss: 905Mb\n" +
				"#256 pulse cov: 4 ft: 2 corp: 41/2726b exec/s: 1 rss: 941Mb\n" +
				"Slowest unit: 26 s: \n" +
				fmt.Sprintf("artifact_prefix='./'; Test unit written to %s\n", testInputFile.Name()) +
				"Base64: c2VhcmNoLXNlcnZpY2UKL3NlYXJjaD9xdWVyeT3/LQUmY2hhbm5lbElkPWtrZXVfZGVfREUmY3VzdG9tZXJJZD0zJmlkc09ubHk9VCZzZXNzaW9uSWQ9LgoK\n" +
				"#512 pulse cov: 4 ft: 4 corp: 59/4451b exec/s: 1 rss: 981Mb",
			expected: []*report.Report{
				{Status: report.RunStatus_INITIALIZING},
				{
					Status: report.RunStatus_RUNNING,
					Metric: &report.FuzzingMetric{
						ExecutionsPerSecond:     1,
						Features:                2,
						Edges:                   4,
						CorpusSize:              27,
						TotalExecutions:         128,
						SecondsSinceLastFeature: 0,
						SecondsSinceLastEdge:    0,
					},
				},
				{
					Status: report.RunStatus_RUNNING,
					Metric: &report.FuzzingMetric{
						ExecutionsPerSecond:     1,
						Features:                2,
						Edges:                   4,
						CorpusSize:              41,
						TotalExecutions:         256,
						SecondsSinceLastFeature: 0,
					},
				},
				{
					Status: report.RunStatus_RUNNING,
					Metric: &report.FuzzingMetric{
						ExecutionsPerSecond:     1,
						Features:                4,
						Edges:                   4,
						CorpusSize:              59,
						TotalExecutions:         512,
						SecondsSinceLastFeature: 0,
						SecondsSinceLastEdge:    0,
					},
				},
				{
					Status: report.RunStatus_RUNNING,
					Finding: &report.Finding{
						Type:      report.ErrorType_WARNING,
						Details:   "Slow input detected. Processing time: 26 s",
						InputData: testInput,
						Logs: []string{
							"Slow input: 26 seconds for processing",
							fmt.Sprintf("artifact_prefix='./'; Test unit written to %s", testInputFile.Name()),
							"Base64: c2VhcmNoLXNlcnZpY2UKL3NlYXJjaD9xdWVyeT3/LQUmY2hhbm5lbElkPWtrZXVfZGVfREUmY3VzdG9tZXJJZD0zJmlkc09ubHk9VCZzZXNzaW9uSWQ9LgoK",
						},
					},
				},
			},
		},
		{
			name: "non-aborting ASan",
			logs: `
INFO: A corpus is not provided, starting from an empty corpus
=================================================================
==16==ERROR: AddressSanitizer: stack-buffer-overflow on address 0x7fffb9492184 at pc 0x0000004969aa bp 0x7fffb9492150 sp 0x7fffb9491918
[...]
  Right alloca redzone:    cb
  Shadow gap:              cc
3280532619
AddressSanitizer:DEADLYSIGNAL
=================================================================
==16==ERROR: AddressSanitizer: SEGV on unknown address 0x000000000000 (pc 0x000000000000 bp 0x7fffb9492290 sp 0x7fffb9492158 T0)
[...]
==16==ABORTING
MS: 0 ; base unit: 0000000000000000000000000000000000000000
0x8b,0xf4,0x88,0xc3,0x68,0x51,0xdf,0x2,
\x8b\xf4\x88\xc3hQ\xdf\x02
artifact_prefix='./'; Test unit written to ` + testInputFile.Name() + `
Base64: i/SIw2hR3wI=`,
			expected: []*report.Report{
				{Status: report.RunStatus_INITIALIZING},
				{
					Status: report.RunStatus_RUNNING,
					Finding: &report.Finding{
						Type:    report.ErrorType_CRASH,
						Details: "stack-buffer-overflow on address 0x7fffb9492184 at pc 0x0000004969aa bp 0x7fffb9492150 sp 0x7fffb9491918",
						Logs: []string{
							"==16==ERROR: AddressSanitizer: stack-buffer-overflow on address 0x7fffb9492184 at pc 0x0000004969aa bp 0x7fffb9492150 sp 0x7fffb9491918",
							"[...]",
							"  Right alloca redzone:    cb",
							"  Shadow gap:              cc",
							"3280532619",
							"AddressSanitizer:DEADLYSIGNAL",
							"=================================================================",
						},
					},
				},
				{
					Status: report.RunStatus_RUNNING,
					Finding: &report.Finding{
						Type:      report.ErrorType_CRASH,
						InputData: testInput,
						Details:   "SEGV on unknown address 0x000000000000 (pc 0x000000000000 bp 0x7fffb9492290 sp 0x7fffb9492158 T0)",
						Logs: []string{
							"==16==ERROR: AddressSanitizer: SEGV on unknown address 0x000000000000 (pc 0x000000000000 bp 0x7fffb9492290 sp 0x7fffb9492158 T0)",
							"[...]",
							"==16==ABORTING",
							"MS: 0 ; base unit: 0000000000000000000000000000000000000000",
							"0x8b,0xf4,0x88,0xc3,0x68,0x51,0xdf,0x2,",
							"\\x8b\\xf4\\x88\\xc3hQ\\xdf\\x02",
							"artifact_prefix='./'; Test unit written to " + testInputFile.Name(),
							"Base64: i/SIw2hR3wI=",
						},
					},
				},
			},
		},
		{
			name: "MSan Bugs",
			logs: `
INFO: A corpus is not provided, starting from an empty corpus
#4805	REDUCE cov: 6 ft: 4 corp: 3/7b exec/s: 10 rss: 47Mb L: 3/3 MS: 1 EraseBytes-
==2248837==WARNING: MemorySanitizer: use-of-uninitialized-value
error info 1
error info 2`,
			expected: []*report.Report{
				{Status: report.RunStatus_INITIALIZING},
				{
					Status: report.RunStatus_RUNNING,
					Metric: &report.FuzzingMetric{
						ExecutionsPerSecond:     10,
						Features:                4,
						Edges:                   6,
						CorpusSize:              3,
						TotalExecutions:         4805,
						SecondsSinceLastFeature: 0,
						SecondsSinceLastEdge:    0,
					},
				},
				{
					Status: report.RunStatus_RUNNING,
					Finding: &report.Finding{
						Type:    report.ErrorType_CRASH,
						Details: "use-of-uninitialized-value",
						Logs: []string{
							"==2248837==WARNING: MemorySanitizer: use-of-uninitialized-value",
							"error info 1",
							"error info 2",
						},
					},
				},
			},
		},
		{
			name: "java libfuzzer driver crash",
			logs: `
INFO: A corpus is not provided, starting from an empty corpus
== Java Exception: java.lang.ArrayIndexOutOfBoundsException: Index 22 out of bounds for length 8
	at com.example.parser.Parser.parseBytes(Parser.java:11)
	at fuzz_targets.FuzzParser.fuzzerTestOneInput(FuzzParser.java:23)

== libFuzzer crashing input ==

MS: 0 ; base unit: 0000000000000000000000000000000000000000
0x64,0x65,0x61,0x64,0x62,0x65,0x65,0x66,
deadbeef
` + fmt.Sprintf("artifact_prefix='./'; Test unit written to %s", testInputFile.Name()) + `
Base64: ZGVhZGJlZWY=`,
			expected: []*report.Report{
				{Status: report.RunStatus_INITIALIZING},
				{
					Status: report.RunStatus_RUNNING,
					Finding: &report.Finding{
						Type:      report.ErrorType_WARNING,
						Details:   "java.lang.ArrayIndexOutOfBoundsException: Index 22 out of bounds for length 8",
						InputData: testInput,
						Logs: []string{
							"== Java Exception: java.lang.ArrayIndexOutOfBoundsException: Index 22 out of bounds for length 8",
							"\tat com.example.parser.Parser.parseBytes(Parser.java:11)",
							"\tat fuzz_targets.FuzzParser.fuzzerTestOneInput(FuzzParser.java:23)",
							"",
							"== libFuzzer crashing input ==",
							"",
							"MS: 0 ; base unit: 0000000000000000000000000000000000000000",
							"0x64,0x65,0x61,0x64,0x62,0x65,0x65,0x66,",
							"deadbeef",
							fmt.Sprintf("artifact_prefix='./'; Test unit written to %s", testInputFile.Name()),
							"Base64: ZGVhZGJlZWY=",
						},
					},
				},
			},
		},
		{
			name: "jazzer FuzzerSecurityIssue",
			logs: `
INFO: A corpus is not provided, starting from an empty corpus
== Java Exception: com.code_intelligence.jazzer.api.FuzzerSecurityIssueHigh: Output contains </script
at com.example.JsonSanitizerXSSFuzzer.fuzzerTestOneInput(JsonSanitizerXSSFuzzer.java:44)
Caused by: com.code_intelligence.jazzer.api.FuzzerSecurityIssueHigh: Output contains </script
... 1 more
== libFuzzer crashing input ==
QQ<script-
` + fmt.Sprintf("artifact_prefix='./'; Test unit written to %s", testInputFile.Name()) + `
Base64: UVFcb1w8L1xzY3JpcHQt`,
			expected: []*report.Report{
				{Status: report.RunStatus_INITIALIZING},
				{
					Status: report.RunStatus_RUNNING,
					Finding: &report.Finding{
						Type:      report.ErrorType_CRASH,
						Details:   "Security Issue: Output contains </script",
						InputData: testInput,
						Logs: []string{
							"== Java Exception: com.code_intelligence.jazzer.api.FuzzerSecurityIssueHigh: Output contains </script",
							"at com.example.JsonSanitizerXSSFuzzer.fuzzerTestOneInput(JsonSanitizerXSSFuzzer.java:44)",
							"Caused by: com.code_intelligence.jazzer.api.FuzzerSecurityIssueHigh: Output contains </script",
							"... 1 more",
							"== libFuzzer crashing input ==",
							"QQ<script-",
							fmt.Sprintf("artifact_prefix='./'; Test unit written to %s", testInputFile.Name()),
							"Base64: UVFcb1w8L1xzY3JpcHQt",
						},
					},
				},
			},
		},
		{
			name: "java assertion error",
			logs: `
INFO: A corpus is not provided, starting from an empty corpus
== Java Assertion Error
== libFuzzer crashing input ==
MS: 0 ; base unit: 0000000000000000000000000000000000000000

deadbeef
` + fmt.Sprintf("artifact_prefix='./'; Test unit written to %s", testInputFile.Name()) + `
Base64: ZGVhZGJlZWY=`,
			expected: []*report.Report{
				{Status: report.RunStatus_INITIALIZING},
				{
					Status: report.RunStatus_RUNNING,
					Finding: &report.Finding{
						Type:      report.ErrorType_WARNING,
						Details:   "Java Assertion Error",
						InputData: testInput,
						Logs: []string{
							"== Java Assertion Error",
							"== libFuzzer crashing input ==",
							"MS: 0 ; base unit: 0000000000000000000000000000000000000000",
							"",
							"deadbeef",
							fmt.Sprintf("artifact_prefix='./'; Test unit written to %s", testInputFile.Name()),
							"Base64: ZGVhZGJlZWY=",
						},
					},
				},
			},
		},
		{
			name: "segfault at the end",
			logs: `
INFO: A corpus is not provided, starting from an empty corpus
==16== ERROR: libFuzzer: deadly signal
    #0 0x4be181 in __sanitizer_print_stack_trace /llvmbuild/llvm-project-llvmorg-10.0.0/compiler-rt/lib/asan/asan_stack.cpp:86:3

SUMMARY: libFuzzer: deadly signal
0xa,0x23,0xa,0x21,0xa,0x3,0x66,0x6f,0x6f,0x12,0x1a,0x1a,0x18,0x62,0x5e,0x0,0x0,0x64,0x65,0x61,0x64,0x62,0x65,0x65,0x66,0x31,0x32,0x33,0x34,0x35,0x36,0x37,0x38,0x39,0x57,0xc7,0x9e,
\x0a#\x0a!\x0a\x03foo\x12\x1a\x1a\x18b^\x00\x00deadbeef123456789W\xc7\x9e
` + fmt.Sprintf("artifact_prefix='./'; Test unit written to %s", testInputFile.Name()) + `
Base64: CiMKIQoDZm9vEhoaGGJeAABkZWFkYmVlZjEyMzQ1Njc4OVfHng==`,
			expected: []*report.Report{
				{Status: report.RunStatus_INITIALIZING},
				{
					Status: report.RunStatus_RUNNING,
					Finding: &report.Finding{
						Type:      report.ErrorType_CRASH,
						Details:   "deadly signal",
						InputData: testInput,
						Logs: []string{
							"==16== ERROR: libFuzzer: deadly signal",
							"    #0 0x4be181 in __sanitizer_print_stack_trace /llvmbuild/llvm-project-llvmorg-10.0.0/compiler-rt/lib/asan/asan_stack.cpp:86:3",
							"",
							"SUMMARY: libFuzzer: deadly signal",
							"0xa,0x23,0xa,0x21,0xa,0x3,0x66,0x6f,0x6f,0x12,0x1a,0x1a,0x18,0x62,0x5e,0x0,0x0,0x64,0x65,0x61,0x64,0x62,0x65,0x65,0x66,0x31,0x32,0x33,0x34,0x35,0x36,0x37,0x38,0x39,0x57,0xc7,0x9e,",
							"\\x0a#\\x0a!\\x0a\\x03foo\\x12\\x1a\\x1a\\x18b^\\x00\\x00deadbeef123456789W\\xc7\\x9e",
							fmt.Sprintf("artifact_prefix='./'; Test unit written to %s", testInputFile.Name()),
							"Base64: CiMKIQoDZm9vEhoaGGJeAABkZWFkYmVlZjEyMzQ1Njc4OVfHng==",
						},
					},
				},
			},
		},
		{
			name: "metric line in the middle of a report",
			logs: `
INFO: A corpus is not provided, starting from an empty corpus
==16== ERROR: libFuzzer: deadly signal
#38	INITED cov: 15 ft: 39 corp: 8/147b exec/s: 38 rss: 44Mb
    #0 0x4a0021 in __sanitizer_print_stack_trace /llvmbuild/llvm-project-llvmorg-10.0.0/compiler-rt/lib/asan/asan_stack.cpp:86:3

SUMMARY: libFuzzer: deadly signal
0x27,0x72,0x72,0x72,0x72,0x62,0x61,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x66,0x6f,0x6f,0x72,0x0,0x72,0x0,0x0,0x1,0x72,0x72,0x0,0x0,0x0,0x0,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,
'rrrrbarrrrrrrrrfoor\x00r\x00\x00\x01rr\x00\x00\x00\x00rrrrrrrrrrrrrrrrrrrrrrrrr
` + fmt.Sprintf("artifact_prefix='./'; Test unit written to %s", testInputFile.Name()) + `
Base64: J3JycnJiYXJycnJycnJycmZvb3IAcgAAAXJyAAAAAHJycnJycnJycnJycnJycnJycnJycnJycnI=`,
			expected: []*report.Report{
				{Status: report.RunStatus_INITIALIZING},
				{
					Status: report.RunStatus_RUNNING,
					Metric: &report.FuzzingMetric{
						ExecutionsPerSecond:     38,
						Features:                39,
						Edges:                   15,
						CorpusSize:              8,
						TotalExecutions:         38,
						SecondsSinceLastFeature: 0,
						SecondsSinceLastEdge:    0,
					},
				},
				{
					Status: report.RunStatus_RUNNING,
					Finding: &report.Finding{
						Type:      report.ErrorType_CRASH,
						Details:   "deadly signal",
						InputData: testInput,
						Logs: []string{
							"==16== ERROR: libFuzzer: deadly signal",
							"    #0 0x4a0021 in __sanitizer_print_stack_trace /llvmbuild/llvm-project-llvmorg-10.0.0/compiler-rt/lib/asan/asan_stack.cpp:86:3",
							"",
							"SUMMARY: libFuzzer: deadly signal",
							"0x27,0x72,0x72,0x72,0x72,0x62,0x61,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x66,0x6f,0x6f,0x72,0x0,0x72,0x0,0x0,0x1,0x72,0x72,0x0,0x0,0x0,0x0,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,0x72,",
							"'rrrrbarrrrrrrrrfoor\\x00r\\x00\\x00\\x01rr\\x00\\x00\\x00\\x00rrrrrrrrrrrrrrrrrrrrrrrrr",
							fmt.Sprintf("artifact_prefix='./'; Test unit written to %s", testInputFile.Name()),
							"Base64: J3JycnJiYXJycnJycnJycmZvb3IAcgAAAXJyAAAAAHJycnJycnJycnJycnJycnJycnJycnJycnI=",
						},
					},
				},
			},
		},
		{
			name: "missing end of report",
			logs: `
INFO: A corpus is not provided, starting from an empty corpus
==16==ERROR: AddressSanitizer: stack-buffer-overflow on address 0x7fffb9492184 at pc 0x0000004969aa bp 0x7fffb9492150 sp 0x7fffb9491918
[...end of report not detected...]
3280532619
==16==ERROR: AddressSanitizer: SEGV on unknown address 0x000000000000 (pc 0x000000000000 bp 0x7fffb9492290 sp 0x7fffb9492158 T0)
[...]
==16==ABORTING
MS: 0 ; base unit: 0000000000000000000000000000000000000000
0x8b,0xf4,0x88,0xc3,0x68,0x51,0xdf,0x2,
\x8b\xf4\x88\xc3hQ\xdf\x02
artifact_prefix='./'; Test unit written to ` + testInputFile.Name() + `
Base64: i/SIw2hR3wI=`,
			expected: []*report.Report{
				{Status: report.RunStatus_INITIALIZING},
				{
					Status: report.RunStatus_RUNNING,
					Finding: &report.Finding{
						Type:    report.ErrorType_CRASH,
						Details: "stack-buffer-overflow on address 0x7fffb9492184 at pc 0x0000004969aa bp 0x7fffb9492150 sp 0x7fffb9491918",
						Logs: []string{
							"==16==ERROR: AddressSanitizer: stack-buffer-overflow on address 0x7fffb9492184 at pc 0x0000004969aa bp 0x7fffb9492150 sp 0x7fffb9491918",
							"[...end of report not detected...]",
							"3280532619",
						},
					},
				},
				{
					Status: report.RunStatus_RUNNING,
					Finding: &report.Finding{
						Type:      report.ErrorType_CRASH,
						InputData: testInput,
						Details:   "SEGV on unknown address 0x000000000000 (pc 0x000000000000 bp 0x7fffb9492290 sp 0x7fffb9492158 T0)",
						Logs: []string{
							"==16==ERROR: AddressSanitizer: SEGV on unknown address 0x000000000000 (pc 0x000000000000 bp 0x7fffb9492290 sp 0x7fffb9492158 T0)",
							"[...]",
							"==16==ABORTING",
							"MS: 0 ; base unit: 0000000000000000000000000000000000000000",
							"0x8b,0xf4,0x88,0xc3,0x68,0x51,0xdf,0x2,",
							"\\x8b\\xf4\\x88\\xc3hQ\\xdf\\x02",
							"artifact_prefix='./'; Test unit written to " + testInputFile.Name(),
							"Base64: i/SIw2hR3wI=",
						},
					},
				},
			},
		},
		{
			name: "timeout-error",
			logs: `INFO: Running with entropic power schedule (0xFF, 100).
INFO: Seed: 3221175179
INFO: Loaded 1 modules   (8 inline 8-bit counters): 8 [0x5acf93, 0x5acf9b), 
INFO: Loaded 1 PC tables (8 PCs): 8 [0x5acfa0,0x5ad020), 
INFO: -max_len is not provided; libFuzzer will not generate inputs larger than 4096 bytes
INFO: A corpus is not provided, starting from an empty corpus
#2	INITED cov: 3 ft: 3 corp: 1/1b exec/s: 0 rss: 30Mb
ALARM: working on the last Unit for 1 seconds
       and the timeout value is 1 (use -timeout=N to change)
MS: 3 ChangeBit-InsertByte-ChangeBinInt-; base unit: adc83b19e793491b1c6ea0fd8b46cd9f32e592fc
0x2,0x2a,
\x02*
` + fmt.Sprintf("artifact_prefix='./'; Test unit written to %s", testInputFile.Name()) + `
Base64: Aio=
==3336601== ERROR: libFuzzer: timeout after 1 seconds
error info 1
error info 2
SUMMARY: libFuzzer: timeout`,
			expected: []*report.Report{
				{Status: report.RunStatus_INITIALIZING},
				{
					Status: report.RunStatus_RUNNING,
					Metric: &report.FuzzingMetric{
						ExecutionsPerSecond:     0,
						Features:                3,
						Edges:                   3,
						CorpusSize:              1,
						TotalExecutions:         2,
						SecondsSinceLastFeature: 0,
						SecondsSinceLastEdge:    0,
					},
				},
				{
					Status: report.RunStatus_RUNNING,
					Finding: &report.Finding{
						Type:      report.ErrorType_CRASH,
						Details:   "timeout after 1 seconds",
						InputData: testInput,
						Logs: []string{
							"ALARM: working on the last Unit for 1 seconds",
							"       and the timeout value is 1 (use -timeout=N to change)",
							"MS: 3 ChangeBit-InsertByte-ChangeBinInt-; base unit: adc83b19e793491b1c6ea0fd8b46cd9f32e592fc",
							"0x2,0x2a,",
							"\\x02*",
							fmt.Sprintf("artifact_prefix='./'; Test unit written to %s", testInputFile.Name()),
							"Base64: Aio=",
							"==3336601== ERROR: libFuzzer: timeout after 1 seconds",
							"error info 1",
							"error info 2",
							"SUMMARY: libFuzzer: timeout",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, w := io.Pipe()

			reporter := NewLibfuzzerOutputParser(&Options{SupportJazzer: true})
			reportsCh := make(chan *report.Report, maxBufferedReports)
			reporterErrCh := make(chan error)

			go func() {
				// pipe in logs line by line
				for _, line := range strings.Split(tt.logs, "\n") {
					_, err := io.WriteString(w, line+"\n")
					require.NoError(t, err)
				}
				require.NoError(t, r.Close())
			}()

			// setup consumer
			go func() {
				reporterErrCh <- reporter.Parse(context.Background(), r, reportsCh)
			}()

			doneCh := make(chan struct{})
			go func() {
				defer func() { doneCh <- struct{}{} }()
				i := 0
				for report := range reportsCh {
					removeTimestamps(report)
					if report.GetFinding() != nil {
						report.Finding.MoreDetails = nil
					}
					require.Equal(t, tt.expected[i], report)
					i += 1
				}
				require.Equal(t, len(tt.expected), i)
			}()

			err := <-reporterErrCh
			require.NoError(t, err)
			<-doneCh
		})
	}
}

func TestBufferOverflowCrashLogs(t *testing.T) {
	// This causes the following error message to be printed during tests:
	//
	//   Error reading crash file: open ./crash-142497307d30883730eb651b077805ff926dcbd8: no such file or directory
	//
	// We tried to replace it with an existing temporary file, but that
	// caused that the resulting report doesn't include the
	// "artifact_prefix='./'; Test unit written to" line, causing
	// assertCorrectCrashesParsing to fail.
	expectedCrashFile, err := ioutil.TempFile("", "crash-")
	require.NoError(t, err)
	defer fileutil.Cleanup(expectedCrashFile.Name())
	testInput := []byte("test")
	_, err = expectedCrashFile.Write(testInput)
	require.NoError(t, err)
	assertCorrectCrashesParsing(t,
		"global-buffer-overflow on address 0x00",
		expectedCrashFile.Name(),
		testInput,
		[]string{
			"==8141==ERROR: AddressSanitizer: global-buffer-overflow on address 0x00",
			"error info 1",
			"artifact_prefix='./'; Test unit written to " + expectedCrashFile.Name(),
			"Base64: Aio=",
		})
}

func TestOOMCrashLogs(t *testing.T) {
	// This also causes an error message to be printed in the tests,
	// like TestBufferOverflowCrashLogs does.
	expectedCrashFile, err := ioutil.TempFile("", "oom-")
	require.NoError(t, err)
	defer fileutil.Cleanup(expectedCrashFile.Name())
	testInput := []byte("test")
	_, err = expectedCrashFile.Write(testInput)
	require.NoError(t, err)
	assertCorrectCrashesParsing(t,
		"out-of-memory (used: 251Mb; limit: 250Mb)",
		expectedCrashFile.Name(),
		testInput,
		[]string{
			"==18== ERROR: libFuzzer: out-of-memory (used: 251Mb; limit: 250Mb)",
			"error info 1",
			"artifact_prefix='./'; Test unit written to " + expectedCrashFile.Name(),
			"Base64: Aio=",
		})
}

func assertCorrectCrashesParsing(t *testing.T, errorDetails, crashFile string, crashingInput []byte, logs []string) {
	expectedReports := []*report.Report{
		{
			Status: report.RunStatus_RUNNING,
			Finding: &report.Finding{
				Type:      report.ErrorType_CRASH,
				InputData: crashingInput,
				Details:   errorDetails,
				Logs:      logs,
			},
		},
	}

	r, w := io.Pipe()

	reporter := NewLibfuzzerOutputParser(nil)
	reporter.initFinished = true
	reportsCh := make(chan *report.Report, maxBufferedReports)
	reporterErrCh := make(chan error)

	go func() {
		for _, logLine := range logs {
			_, err := io.WriteString(w, logLine+"\n")
			require.NoError(t, err)
		}
		require.NoError(t, r.Close())
	}()

	go func() {
		reporterErrCh <- reporter.Parse(context.Background(), r, reportsCh)
	}()

	doneCh := make(chan struct{})
	go func() {
		i := 0
		for report := range reportsCh {
			assert.Equal(t, expectedReports[i], report)
			i += 1
		}
		assert.Equal(t, len(expectedReports), i)
		doneCh <- struct{}{}
	}()

	err := <-reporterErrCh
	require.NoError(t, err)

	for _, logLine := range logs {
		f, ok := parseAsTestInputFilePath(logLine)
		if ok {
			require.Equal(t, crashFile, f)
			break
		}
	}

	<-doneCh
}

func removeTimestamps(r *report.Report) {
	if r.Metric != nil {
		r.Metric.Timestamp = time.Time{}
	}
}
