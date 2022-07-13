package log

import (
	"fmt"
	"io"
	"os"

	"github.com/pterm/pterm"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

var disableColor bool

var Output io.Writer

func init() {
	Output = os.Stderr
	// Disable color if stderr is not a terminal. We don't use
	// color.Disable() here because that would disable color for all
	// pterm and color methods, but we might want to use color in output
	// printed to stdout (if stdout is a terminal).
	disableColor = !term.IsTerminal(int(os.Stderr.Fd()))
}

func log(style pterm.Style, icon string, a ...any) {
	s := icon + fmt.Sprint(a...)
	if len(s) == 0 || s[len(s)-1] != '\n' {
		s += "\n"
	}

	if disableColor {
		s = pterm.RemoveColorFromString(s)
	} else {
		s = style.Sprint(s)
	}

	// Clear the updating printer output if any. We don't use
	// pterm.Fprint here, which also tries to clear spinner printer
	// output, because that only works when the spinner printer and this
	// function write to the same output stream, which is not always the
	// case, because we let the spinner printer write to stdout unless
	// --json was used. The advantage of that is that it allows piping
	// stderr to a log file while still seeing the output that's mostly
	// relevant during execution. But if that continues to add complexity
	// to our code, we might want to reassess the cost/benefit.
	if ActiveUpdatingPrinter != nil {
		ActiveUpdatingPrinter.Clear()
	}
	_, _ = fmt.Fprint(Output, s)
}

// Successf highlights a message as successful
func Successf(format string, a ...any) {
	Success(fmt.Sprintf(format, a...))
}

func Success(a ...any) {
	log(pterm.Style{pterm.FgGreen}, "✅ ", a...)
}

// Warnf highlights a message as a warning
func Warnf(format string, a ...any) {
	Warn(fmt.Sprintf(format, a...))
}

func Warn(a ...any) {
	log(pterm.Style{pterm.Bold, pterm.FgYellow}, "⚠️ ", a...)
}

// Errorf highlights a message as an error and shows the stack strace if the --verbose flag is active
func Errorf(err error, format string, a ...any) {
	Error(err, fmt.Sprintf(format, a...))
}

func Error(err error, a ...any) {
	// If no message is provided, print the message of the error
	if len(a) == 0 {
		a = []any{err.Error()}
	}
	log(pterm.Style{pterm.Bold, pterm.FgRed}, "❌ ", a...)
	Debugf("%+v", err)
}

// Infof outputs a regular user message without any highlighting
func Infof(format string, a ...any) {
	Info(fmt.Sprintf(format, a...))
}

func Info(a ...any) {
	log(pterm.Style{pterm.Fuzzy}, "", a...)
}

// Debugf outputs additional information when the --verbose flag is active
func Debugf(format string, a ...any) {
	Debug(fmt.Sprintf(format, a...))
}

func Debug(a ...any) {
	if viper.GetBool("verbose") {
		log(pterm.Style{pterm.Fuzzy}, "🔍 ", a...)
	}
}

// Printf writes without any colors
func Printf(format string, a ...any) {
	Print(fmt.Sprintf(format, a...))
}

func Print(a ...any) {
	log(pterm.Style{pterm.FgDefault}, "", a...)
}
