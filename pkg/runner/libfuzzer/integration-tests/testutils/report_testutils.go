package testutils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"code-intelligence.com/cifuzz/pkg/report"
)

type CheckReportOptions struct {
	ErrorType           report.ErrorType
	SourceFile          string
	Details             string
	AllowEmptyInputData bool
	NumFindings         int
}

// CheckReports offers an easy way to check a set of reports against some options
func CheckReports(t *testing.T, reports []*report.Report, options *CheckReportOptions) {
	numFindings := 0
	for _, report := range reports {
		require.NotNil(t, report)

		finding := report.GetFinding()
		// finding can be null for status == initializing
		if finding == nil {
			continue
		}
		numFindings++

		require.Equal(t, options.ErrorType, finding.Type)

		logs := strings.Join(finding.Logs, "\n")

		if options.SourceFile != "" {
			// Check that the report contains the fuzz target source file, which
			// means that address sanitizer was able to use llvm-symbolizer to
			// convert the memory addresses to source code locations.
			require.Contains(t, logs, options.SourceFile)
		}

		if options.Details != "" {
			require.Contains(t, finding.GetDetails(), options.Details)
		}

		if !options.AllowEmptyInputData {
			require.NotEmpty(t, finding.InputData, "InputData is empty")
		}
	}
	require.Equal(t, options.NumFindings, numFindings, "total amount of findings not as expected")
}
