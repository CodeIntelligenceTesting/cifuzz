package log

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/viper"
)

func log(msgColor color.Attribute, icon, msg string, args ...interface{}) {
	color.Set(msgColor)
	s := fmt.Sprintf(icon+msg, args...)
	if len(s) == 0 || s[len(s)-1] != '\n' {
		s += "\n"
	}
	_, _ = fmt.Fprint(os.Stderr, s)
	defer color.Unset()
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
