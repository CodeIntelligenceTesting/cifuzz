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

func log(msgColor color.Attribute, icon, msg string, args ...interface{}) {
	color.Set(msgColor)
	defer color.Unset()

	s := fmt.Sprintf(icon+msg, args...)
	if len(s) == 0 || s[len(s)-1] != '\n' {
		s += "\n"
	}
	_, _ = fmt.Fprint(os.Stderr, s)
}

// Successf highlights a message as successful
func Successf(msg string, args ...interface{}) {
	log(color.FgGreen, "✅ ", msg, args...)
}

func Success(msg string) {
	Successf(msg + "\n")
}

// Warnf highlights a message as a warning
func Warnf(msg string, args ...interface{}) {
	log(color.FgYellow, "⚠️ ", msg, args...)
}

func Warn(msg string) {
	Warnf(msg + "\n")
}

// Errorf highlights a message as an error and shows the stack strace if the --verbose flag is active
func Errorf(err error, msg string, args ...interface{}) {
	log(color.FgRed, "❌ ", msg, args...)
	Debugf("%+v", err)
}

func Error(err error, msg string) {
	Errorf(err, msg+"\n")
}

// Infof outputs a regular user message without any highlighting
func Infof(msg string, args ...interface{}) {
	log(color.FgWhite, "", msg, args...)
}

func Info(msg string) {
	Infof(msg + "\n")
}

// Debugf outputs additional information when the --verbose flag is active
func Debugf(msg string, args ...interface{}) {
	if viper.GetBool("verbose") {
		log(color.FgWhite, "🔍 ", msg, args...)
	}
}

func Debug(msg string) {
	Debugf(msg + "\n")
}
