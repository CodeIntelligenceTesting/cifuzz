package metrics

import (
	"fmt"
	"strconv"
	"time"

	"github.com/pterm/pterm"

	"code-intelligence.com/cifuzz/pkg/report"
)

type Printer interface {
	PrintMetrics(metrics *report.FuzzingMetric)
}

func DescString(format string, a ...any) string {
	return pterm.FgWhite.Sprintf(format, a...)
}

func NumberString(format string, a ...any) string {
	return pterm.FgLightCyan.Sprintf(format, a...)
}

func DelimString(format string, a ...any) string {
	return pterm.FgLightWhite.Sprintf(format, a...)
}

func MetricsToString(metrics *report.FuzzingMetric) string {
	var paths, lastNewPath, executionsPerSecond string
	if metrics == nil {
		paths = "0"
		lastNewPath = "none yet"
		executionsPerSecond = "n/a"
	} else {
		paths = strconv.FormatInt(int64(metrics.Features), 10)
		lastNewPath = (time.Second * time.Duration(metrics.SecondsSinceLastFeature)).String()
		executionsPerSecond = strconv.FormatInt(int64(metrics.ExecutionsPerSecond), 10)
	}

	return fmt.Sprint(DescString("paths: "),
		NumberString("%s", paths),
		DelimString(" - "),
		DescString("last new path: "),
		NumberString("%s", lastNewPath),
		DelimString(" - "),
		DescString("exec/s: "),
		NumberString("%s", executionsPerSecond),
	)
}
