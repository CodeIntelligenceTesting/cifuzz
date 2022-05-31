package metrics

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/pterm/pterm"
	"golang.org/x/term"

	"code-intelligence.com/cifuzz/pkg/report"
)

func NewLinePrinter(output io.Writer) *LinePrinter {
	p := pterm.DefaultBasicText.WithWriter(output)
	return &LinePrinter{
		BasicTextPrinter: p,
		startedAt:        time.Now(),
	}
}

type LinePrinter struct {
	*pterm.BasicTextPrinter
	startedAt time.Time
}

func (p *LinePrinter) PrintMetrics(metrics *report.FuzzingMetric) {
	s := fmt.Sprint(
		MetricsToString(metrics),
		DelimString(" ("),
		pterm.LightYellow(time.Since(p.startedAt).Round(time.Second).String()),
		DelimString(")"),
	)
	if len(s) == 0 || s[len(s)-1] != '\n' {
		s += "\n"
	}
	// Print without color if the output is not a TTY
	if file, ok := p.BasicTextPrinter.Writer.(*os.File); !ok || !term.IsTerminal(int(file.Fd())) {
		s = pterm.RemoveColorFromString(s)
	}
	p.Print(s)
}
