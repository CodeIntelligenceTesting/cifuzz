package log

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/mattn/go-colorable"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

func init() {
	// Make the color package print color control sequences to stderr
	// instead of stdout
	color.Output = colorable.NewColorableStderr()

	// The color package disables color if stdout is not a terminal, but
	// we print to stderr, so we disable color if stderr is not a
	// terminal.
	color.NoColor = !term.IsTerminal(int(os.Stderr.Fd()))
}

func logf(msgColor color.Attribute, icon, format string, a ...any) {
	log(msgColor, icon, fmt.Sprintf(format, a...))
}

func log(msgColor color.Attribute, icon string, a ...any) {
	color.Set(msgColor)
	defer color.Unset()

	s := icon + fmt.Sprint(a...)
	if len(s) == 0 || s[len(s)-1] != '\n' {
		s += "\n"
	}
	_, _ = fmt.Fprint(os.Stderr, s)
}

// Successf highlights a message as successful
func Successf(format string, a ...any) {
	logf(color.FgGreen, "✅ ", format, a...)
}

func Success(a ...any) {
	log(color.FgGreen, "✅ ", a...)
}

// Warnf highlights a message as a warning
func Warnf(format string, a ...any) {
	logf(color.FgYellow, "⚠️ ", format, a...)
}

func Warn(a ...any) {
	log(color.FgYellow, "⚠️ ", a...)
}

// Errorf highlights a message as an error and shows the stack strace if the --verbose flag is active
func Errorf(err error, format string, a ...any) {
	logf(color.FgRed, "❌ ", format, a...)
	Debugf("%+v", err)
}

func Error(err error, a ...any) {
	log(color.FgRed, "❌ ", a...)
	Debugf("%+v", err)
}

// Infof outputs a regular user message without any highlighting
func Infof(format string, a ...any) {
	logf(color.FgWhite, "", format, a...)
}

func Info(a ...any) {
	log(color.FgWhite, "", a...)
}

// Debugf outputs additional information when the --verbose flag is active
func Debugf(format string, a ...any) {
	if viper.GetBool("verbose") {
		logf(color.FgWhite, "🔍 ", format, a...)
	}
}

func Debug(a ...any) {
	if viper.GetBool("verbose") {
		log(color.FgWhite, "🔍 ", a...)
	}
}
