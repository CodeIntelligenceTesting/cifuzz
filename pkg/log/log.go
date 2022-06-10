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

func log(msgColor pterm.Color, icon string, a ...any) {
	s := icon + fmt.Sprint(a...)
	if len(s) == 0 || s[len(s)-1] != '\n' {
		s += "\n"
	}

	if disableColor {
		s = pterm.RemoveColorFromString(s)
	} else {
		s = pterm.Style{msgColor}.Sprint(s)
	}

	_, _ = fmt.Fprint(Output, s)
}

// Successf highlights a message as successful
func Successf(format string, a ...any) {
	Success(fmt.Sprintf(format, a...))
}

func Success(a ...any) {
	log(pterm.FgGreen, "✅ ", a...)
}

// Warnf highlights a message as a warning
func Warnf(format string, a ...any) {
	Warn(fmt.Sprintf(format, a...))
}

func Warn(a ...any) {
	log(pterm.FgYellow, "⚠️ ", a...)
}

// Errorf highlights a message as an error and shows the stack strace if the --verbose flag is active
func Errorf(err error, format string, a ...any) {
	Error(err, fmt.Sprintf(format, a...))
}

func Error(err error, a ...any) {
	log(pterm.FgRed, "❌ ", a...)
	Debugf("%+v", err)
}

// Infof outputs a regular user message without any highlighting
func Infof(format string, a ...any) {
	Info(fmt.Sprintf(format, a...))
}

func Info(a ...any) {
	log(pterm.FgWhite, "", a...)
}

// Debugf outputs additional information when the --verbose flag is active
func Debugf(format string, a ...any) {
	Debug(fmt.Sprintf(format, a...))
}

func Debug(a ...any) {
	if viper.GetBool("verbose") {
		log(pterm.FgWhite, "🔍 ", a...)
	}
}

// Printf writes without any colors
func Printf(format string, a ...any) {
	Print(fmt.Sprintf(format, a...))
}

func Print(a ...any) {
	log(pterm.FgDefault, "", a...)
}
