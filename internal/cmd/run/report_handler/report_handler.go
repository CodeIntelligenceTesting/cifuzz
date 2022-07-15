package report_handler

import (
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/gookit/color"
	"github.com/hokaccha/go-prettyjson"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"golang.org/x/term"

	"code-intelligence.com/cifuzz/internal/cmd/run/report_handler/metrics"
	"code-intelligence.com/cifuzz/internal/names"
	"code-intelligence.com/cifuzz/pkg/log"
	"code-intelligence.com/cifuzz/pkg/report"
	"code-intelligence.com/cifuzz/util/stringutil"
)

type ReportHandler struct {
	printJSON            bool
	verbose              bool
	usingUpdatingPrinter bool

	printer      metrics.Printer
	startedAt    time.Time
	initStarted  bool
	initFinished bool

	lastMetrics  *report.FuzzingMetric
	firstMetrics *report.FuzzingMetric

	numFindings    uint
	numSeedsAtInit uint
	seedCorpusDir  string

	jsonOutput io.Writer
}

func NewReportHandler(seedCorpusDir string, printJSON, verbose bool) (*ReportHandler, error) {
	var err error
	h := &ReportHandler{
		seedCorpusDir: seedCorpusDir,
		printJSON:     printJSON,
		verbose:       verbose,
		startedAt:     time.Now(),
		jsonOutput:    os.Stdout,
	}

	// When --json was used, we don't want anything but JSON output on
	// stdout, so we make the printer use stderr.
	var printerOutput *os.File
	if printJSON {
		printerOutput = os.Stderr
	} else {
		printerOutput = os.Stdout
	}

	// Use an updating printer if the output stream is a TTY
	if term.IsTerminal(int(printerOutput.Fd())) {
		h.printer, err = metrics.NewUpdatingPrinter(printerOutput)
		if err != nil {
			return nil, err
		}
		h.usingUpdatingPrinter = true
	} else {
		h.printer = metrics.NewLinePrinter(printerOutput)
	}

	return h, nil
}

func (h *ReportHandler) Handle(r *report.Report) error {
	var err error

	if r.Finding != nil {
		// Count the number of findings for the final metrics
		h.numFindings += 1

		if r.Finding.Name == "" {
			// create a name based on a hash of the crashing input
			h := sha1.New()
			h.Write(r.Finding.InputData)
			r.Finding.Name = names.GetDeterministicName(h.Sum(nil))
		}

		if err := r.Finding.Save(); err != nil {
			return err
		}
	}

	// Print report as JSON if the --json flag was specified
	if h.printJSON {
		var jsonString string
		// Print with color if the output stream is a TTY
		if file, ok := h.jsonOutput.(*os.File); !ok || !term.IsTerminal(int(file.Fd())) {
			bytes, err := prettyjson.Marshal(r)
			if err != nil {
				return errors.WithStack(err)
			}
			jsonString = string(bytes)
		} else {
			jsonString, err = stringutil.ToJsonString(r)
			if err != nil {
				return err
			}
		}
		if h.usingUpdatingPrinter {
			// Clear the updating printer
			h.printer.(*metrics.UpdatingPrinter).Clear()
		}
		_, _ = fmt.Fprintln(h.jsonOutput, jsonString)
		return nil
	}

	if r.Status == report.RunStatus_INITIALIZING && !h.initStarted {
		h.initStarted = true
		h.numSeedsAtInit = r.NumSeeds
		if r.NumSeeds == 0 {
			log.Info("Starting from an empty corpus")
			h.initFinished = true
		} else {
			log.Info("Initializing fuzzer with ", pterm.FgLightCyan.Sprintf("%d", r.NumSeeds), " seed inputs")
		}
	}

	if r.Status == report.RunStatus_RUNNING && !h.initFinished {
		log.Info("Successfully initialized fuzzer with seed inputs")
		h.initFinished = true
	}

	if r.Finding != nil && !h.verbose {
		log.Print("\n")
		log.Printf("=========================== Finding %d ===========================", h.numFindings)
		log.Print(strings.Join(r.Finding.Logs, "\n"))

		if r.Finding.InputFile != "" {
			destPath := filepath.Join(h.seedCorpusDir, r.Finding.Name)

			copyCmd := fmt.Sprintf("mkdir -p %s && cp", h.seedCorpusDir)
			if runtime.GOOS == "windows" {
				copyCmd = fmt.Sprintf("if not exist %s mkdir %s && copy", destPath, destPath)
			}

			log.Print("\n")
			log.Print("You can add this crashing input to the seed corpus with:")
			log.Infof("  %s %s %s", copyCmd, r.Finding.InputFile, destPath)
			log.Print("After adding, the input will be applied every time you run the regression tests / replayer binary (for example during your CI/CD pipeline).")
			log.Print("For more information you can take a look at https://github.com/CodeIntelligenceTesting/cifuzz#regression-testing")
		}

		log.Print("=================================================================")
	}

	if r.Metric != nil {
		h.lastMetrics = r.Metric
		if h.firstMetrics == nil {
			h.firstMetrics = r.Metric
		}
		h.printer.PrintMetrics(r.Metric)
	}

	return nil
}

func (h *ReportHandler) PrintFinalMetrics(numSeeds uint) error {
	// We don't want to print colors to stderr unless it's a TTY
	if !term.IsTerminal(int(os.Stderr.Fd())) {
		color.Disable()
	}

	if h.usingUpdatingPrinter {
		// Stop the updating printer
		updatingPrinter := h.printer.(*metrics.UpdatingPrinter)
		err := updatingPrinter.Stop()
		if err != nil {
			return errors.WithStack(err)
		}
	} else {
		// Stopping the updating printer leaves an empty line, which
		// we actually want before the final metrics (because it looks
		// better), so in case we did not use an updating printer,
		// print an empty line anyway.
		log.Print("\n")
	}

	duration := time.Since(h.startedAt)
	totalSeeds := numSeeds
	newSeeds := totalSeeds - h.numSeedsAtInit

	var averageExecsStr string

	if h.firstMetrics == nil {
		averageExecsStr = metrics.NumberString("n/a")
	} else {
		var averageExecs uint64
		metricsDuration := h.lastMetrics.Timestamp.Sub(h.firstMetrics.Timestamp)
		if metricsDuration.Milliseconds() == 0 {
			// The first and last metrics are either the same or were
			// printed too fast one after the other to calculate a
			// meaningful average, so we just use the exec/s from the
			// current metrics as the average.
			averageExecs = uint64(h.lastMetrics.ExecutionsPerSecond)
		} else {
			// We use milliseconds here to calculate a more accurate average
			execs := h.lastMetrics.TotalExecutions - h.firstMetrics.TotalExecutions
			averageExecs = uint64(float64(execs) / (float64(metricsDuration.Milliseconds()) / 1000))
		}
		averageExecsStr = metrics.NumberString("%d", averageExecs)
	}

	// Round towards the next larger second to avoid that very short
	// runs show "Ran for 0s".
	durationStr := (duration.Truncate(time.Second) + time.Second).String()

	lines := []string{
		metrics.DescString("Execution time:\t") + metrics.NumberString(durationStr),
		metrics.DescString("Average exec/s:\t") + averageExecsStr,
		metrics.DescString("Findings:\t") + metrics.NumberString("%d", h.numFindings),
		metrics.DescString("New seeds:\t") + metrics.NumberString("%d", newSeeds) +
			metrics.DescString(" (total: %s)", metrics.NumberString("%d", totalSeeds)),
	}

	w := tabwriter.NewWriter(log.NewPTermWriter(os.Stderr), 0, 0, 1, ' ', 0)
	for _, line := range lines {
		_, err := fmt.Fprintln(w, line)
		if err != nil {
			return errors.WithStack(err)
		}
	}
	err := w.Flush()
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
