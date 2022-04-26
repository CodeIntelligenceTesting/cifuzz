package out

import (
	"fmt"
	"io"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/viper"
)

func print(target io.Writer, msgColor color.Attribute, icon, msg string, args ...interface{}) {
	color.Set(msgColor)
	_, _ = fmt.Fprintf(target, icon+msg+"\n", args...)
	defer color.Unset()
}

// Success highlights a message as successful
func Success(msg string, args ...interface{}) {
	print(os.Stdout, color.FgGreen, "✅ ", msg, args...)
}

// Warn highlights a message as a warning
func Warn(msg string, args ...interface{}) {
	print(os.Stderr, color.FgYellow, "⚠️ ", msg, args...)
}

// Error highlights a message as an error and shows the stack strace if the --verbose flag is active
func Error(err error, msg string, args ...interface{}) {
	print(os.Stderr, color.FgRed, "❌ ", msg, args...)
	Debug("%+v", err)
}

// Info outputs a regular user message without any highlighting
func Info(msg string, args ...interface{}) {
	print(os.Stdout, color.FgWhite, "", msg, args...)
}

// Debug outputs additional information when the --verbose flag is active
func Debug(msg string, args ...interface{}) {
	if viper.GetBool("verbose") {
		print(os.Stderr, color.FgWhite, "🔍 ", msg, args...)
	}
}
