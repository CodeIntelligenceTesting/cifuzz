package main

import (
	"github.com/spf13/viper"

	"code-intelligence.com/cifuzz/internal/cmd/root"
)

func init() {
	viper.SetEnvPrefix("CIFUZZ")
	viper.AutomaticEnv()
}

func main() {
	root.Execute()
}
