package dialog

import (
	"fmt"
	"io"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/viper"
)

func print(target io.Writer, msgColor color.Attribute, icon, msg string, args ...interface{}) {
	color.Set(msgColor)
	_, _ = fmt.Fprintf(target, icon+msg, args...)
	defer color.Unset()
}

// Successf highlights a message as successful
func Successf(msg string, args ...interface{}) {
	print(os.Stdout, color.FgGreen, "✅ ", msg, args...)
}

func Success(msg string) {
	Successf(msg + "\n")
}

// Warnf highlights a message as a warning
func Warnf(msg string, args ...interface{}) {
	print(os.Stderr, color.FgYellow, "⚠️ ", msg, args...)
}

func Warn(msg string) {
	Warnf(msg + "\n")
}

// Errorf highlights a message as an error and shows the stack strace if the --verbose flag is active
func Errorf(err error, msg string, args ...interface{}) {
	print(os.Stderr, color.FgRed, "❌ ", msg, args...)
	Debugf("%+v", err)
}

func Error(err error, msg string) {
	Errorf(err, msg+"\n")
}

// Infof outputs a regular user message without any highlighting
func Infof(msg string, args ...interface{}) {
	print(os.Stdout, color.FgWhite, "", msg, args...)
}

func Info(msg string) {
	Infof(msg + "\n")
}

// Debugf outputs additional information when the --verbose flag is active
func Debugf(msg string, args ...interface{}) {
	if viper.GetBool("verbose") {
		print(os.Stderr, color.FgWhite, "🔍 ", msg, args...)
	}
}

func Debug(msg string) {
	Debugf(msg + "\n")
}
