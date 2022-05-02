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

// SuccessF highlights a message as successful
func SuccessF(msg string, args ...interface{}) {
	print(os.Stdout, color.FgGreen, "✅ ", msg, args...)
}

func Success(msg string) {
	SuccessF(msg + "\n")
}

// WarnF highlights a message as a warning
func WarnF(msg string, args ...interface{}) {
	print(os.Stderr, color.FgYellow, "⚠️ ", msg, args...)
}

func Warn(msg string) {
	WarnF(msg + "\n")
}

// ErrorF highlights a message as an error and shows the stack strace if the --verbose flag is active
func ErrorF(err error, msg string, args ...interface{}) {
	print(os.Stderr, color.FgRed, "❌ ", msg, args...)
	DebugF("%+v", err)
}

func Error(err error, msg string) {
	ErrorF(err, msg+"\n")
}

// InfoF outputs a regular user message without any highlighting
func InfoF(msg string, args ...interface{}) {
	print(os.Stdout, color.FgWhite, "", msg, args...)
}

func Info(msg string) {
	InfoF(msg + "\n")
}

// DebugF outputs additional information when the --verbose flag is active
func DebugF(msg string, args ...interface{}) {
	if viper.GetBool("verbose") {
		print(os.Stderr, color.FgWhite, "🔍 ", msg, args...)
	}
}

func Debug(msg string) {
	DebugF(msg + "\n")
}
