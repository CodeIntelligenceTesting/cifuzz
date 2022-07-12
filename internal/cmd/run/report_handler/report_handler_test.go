package report_handler

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/gookit/color"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code-intelligence.com/cifuzz/internal/cmd/run/report_handler/metrics"
	"code-intelligence.com/cifuzz/pkg/log"
	"code-intelligence.com/cifuzz/pkg/report"
	"code-intelligence.com/cifuzz/util/fileutil"
	"code-intelligence.com/cifuzz/util/testutil"
)

var logOutput io.ReadWriter

func TestMain(m *testing.M) {
	// Disable color for this test to allow comparing strings without
	// having to add color to them
	color.Disable()

	logOutput = bytes.NewBuffer([]byte{})
	log.Output = logOutput

	testTempDir := testutil.ChdirToTempDir("report-handler-test-")
	defer fileutil.Cleanup(testTempDir)

	m.Run()
}

func TestReportHandler_EmptyCorpus(t *testing.T) {
	h, err := NewReportHandler(false, false)
	require.NoError(t, err)

	initStartedReport := &report.Report{
		Status:   report.RunStatus_INITIALIZING,
		NumSeeds: 0,
	}
	err = h.Handle(initStartedReport)
	require.NoError(t, err)
	checkOutput(t, logOutput, "Starting from an empty corpus")
	require.True(t, h.initFinished)
}

func TestReportHandler_NonEmptyCorpus(t *testing.T) {
	h, err := NewReportHandler(false, false)
	require.NoError(t, err)

	initStartedReport := &report.Report{
		Status:   report.RunStatus_INITIALIZING,
		NumSeeds: 1,
	}
	err = h.Handle(initStartedReport)
	require.NoError(t, err)
	checkOutput(t, logOutput, "Initializing fuzzer with")

	initFinishedReport := &report.Report{Status: report.RunStatus_RUNNING}
	err = h.Handle(initFinishedReport)
	require.NoError(t, err)
	checkOutput(t, logOutput, "Successfully initialized fuzzer")
}

func TestReportHandler_Metrics(t *testing.T) {
	h, err := NewReportHandler(false, false)
	require.NoError(t, err)

	printerOut := bytes.NewBuffer([]byte{})
	h.printer.(*metrics.LinePrinter).BasicTextPrinter.Writer = printerOut

	metricsReport := &report.Report{
		Status: report.RunStatus_RUNNING,
		Metric: &report.FuzzingMetric{
			Timestamp:           time.Now(),
			ExecutionsPerSecond: 1234,
			Features:            12,
		},
	}
	err = h.Handle(metricsReport)
	require.NoError(t, err)
	checkOutput(t, printerOut, metrics.MetricsToString(metricsReport.Metric))
}

func TestReportHandler_Finding(t *testing.T) {
	h, err := NewReportHandler(false, false)
	require.NoError(t, err)

	findingLogs := []string{"Oops", "The application crashed"}
	findingReport := &report.Report{
		Status: report.RunStatus_RUNNING,
		Finding: &report.Finding{
			Logs: findingLogs,
		},
	}
	err = h.Handle(findingReport)
	require.NoError(t, err)
	checkOutput(t, logOutput, append([]string{"Finding 1"}, findingLogs...)...)
}

func TestReportHandler_PrintJSON(t *testing.T) {
	h, err := NewReportHandler(true, false)
	require.NoError(t, err)

	jsonOut := bytes.NewBuffer([]byte{})
	h.jsonOutput = jsonOut

	findingLogs := []string{"Oops", "The program crashed"}
	findingReport := &report.Report{
		Status: report.RunStatus_RUNNING,
		Finding: &report.Finding{
			Logs: findingLogs,
		},
	}
	err = h.Handle(findingReport)
	require.NoError(t, err)
	checkOutput(t, jsonOut, findingLogs...)
}

func TestReportHandler_GenerateName(t *testing.T) {
	h, err := NewReportHandler(true, false)
	require.NoError(t, err)

	findingLogs := []string{"Oops", "The program crashed"}
	findingReport := &report.Report{
		Status: report.RunStatus_RUNNING,
		Finding: &report.Finding{
			Logs:      findingLogs,
			InputData: []byte("123"),
		},
	}
	err = h.Handle(findingReport)
	require.NoError(t, err)
	assert.Equal(t, "angry_ptolemy", findingReport.Finding.Name)
}

func TestReportHandler_NotOverrideName(t *testing.T) {
	h, err := NewReportHandler(true, false)
	require.NoError(t, err)

	findingLogs := []string{"Oops", "The program crashed"}
	findingReport := &report.Report{
		Status: report.RunStatus_RUNNING,
		Finding: &report.Finding{
			Logs: findingLogs,
			Name: "MyName",
		},
	}
	err = h.Handle(findingReport)
	require.NoError(t, err)
	assert.Equal(t, "MyName", findingReport.Finding.Name)
}
func checkOutput(t *testing.T, r io.Reader, s ...string) {
	output, err := io.ReadAll(r)
	require.NoError(t, err)
	for _, str := range s {
		require.Contains(t, string(output), str)
	}
}
