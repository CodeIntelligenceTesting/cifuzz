package out

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/viper"
)

func print(msgColor color.Attribute, icon, msg string, args ...interface{}) {
	color.Set(msgColor)
	fmt.Printf(icon+msg+"\n", args...)
	defer color.Unset()
}

// Success highlights a message as successful
func Success(msg string, args ...interface{}) {
	print(color.FgGreen, "✅ ", msg, args...)
}

// Warn highlights a message as a warning
func Warn(msg string, args ...interface{}) {
	print(color.FgYellow, "⚠️ ", msg, args...)
}

// Error highlights a message as an error and shows the stack strace if the --verbose flag is active
func Error(err error, msg string, args ...interface{}) {
	print(color.FgRed, "❌ ", msg, args...)
	Debug("%+v", err)
}

// Info outputs a regular user message without any highlighting
func Info(msg string, args ...interface{}) {
	print(color.FgWhite, "", msg, args...)
}

// Debug outputs additional information when the --verbose flag is active
func Debug(msg string, args ...interface{}) {
	if viper.GetBool("verbose") {
		print(color.FgWhite, "🔍 ", msg, args...)
	}
}
