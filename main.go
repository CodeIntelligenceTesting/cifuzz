package main

import (
	"code-intelligence.com/cifuzz/cmd"

	"github.com/spf13/viper"
)

func init() {
	viper.SetEnvPrefix("CIFUZZ")
	viper.AutomaticEnv()
}

func main() {
	cmd.Execute()
}
