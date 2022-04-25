package out

import (
	"fmt"

	"github.com/fatih/color"
)

func print(msgColor color.Attribute, icon, msg string, args ...interface{}) {
	color.Set(msgColor)
	fmt.Printf(icon+msg+"\n", args...)
	color.Unset()
}

func Success(msg string, args ...interface{}) {
	print(color.FgGreen, "✔ ", msg, args...)
}

func Warn(msg string, args ...interface{}) {
	print(color.FgGreen, "! ", msg, args...)
}

func Error(err error, msg string, args ...interface{}) {
	print(color.FgRed, "✗ ", msg, args...)
}

func Info(msg string, args ...interface{}) {
	print(color.FgWhite, "", msg, args...)
}
