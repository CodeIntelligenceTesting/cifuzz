package metrics

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gookit/color"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"

	"code-intelligence.com/cifuzz/pkg/log"
	"code-intelligence.com/cifuzz/pkg/report"
)

func NewUpdatingPrinter(output io.Writer) (*UpdatingPrinter, error) {
	// pterm.SpinnerPrinter doesn't support specifying the output, it
	// always uses color.output, so we have to set that. Note that this
	// affects the default output of all methods from the pterm and
	// color packages.
	color.SetOutput(output)

	var err error
	p := &UpdatingPrinter{
		SpinnerPrinter: pterm.DefaultSpinner.WithShowTimer(false),
		startedAt:      time.Now(),
		output:         output,
	}
	log.ActiveUpdatingPrinter = p

	p.SpinnerPrinter, err = p.SpinnerPrinter.Start()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	p.ticker = time.NewTicker(time.Second)
	go func() {
		for range p.ticker.C {
			p.Update()
		}
	}()

	return p, nil
}

type UpdatingPrinter struct {
	*pterm.SpinnerPrinter
	ticker    *time.Ticker
	startedAt time.Time
	output    io.Writer

	lastMetrics *report.FuzzingMetric
}

func (p *UpdatingPrinter) Update() {
	if p.lastMetrics != nil {
		p.lastMetrics.SecondsSinceLastFeature += 1
	}
	p.printMetrics(p.lastMetrics)
}

func (p *UpdatingPrinter) PrintMetrics(metrics *report.FuzzingMetric) {
	p.lastMetrics = metrics
	p.ticker.Reset(time.Second)
	p.printMetrics(metrics)
}

func (p *UpdatingPrinter) printMetrics(metrics *report.FuzzingMetric) {
	s := fmt.Sprint(
		MetricsToString(metrics),
		DelimString(" ("),
		pterm.LightYellow(time.Since(p.startedAt).Round(time.Second).String()),
		DelimString(")"),
	)
	p.UpdateText(s)
}

func (p *UpdatingPrinter) Clear() {
	pterm.Fprinto(p.output, strings.Repeat(" ", pterm.GetTerminalWidth()), "\r")
}
