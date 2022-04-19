package main

import (
	"github.com/CodeIntelligenceTesting/cifuzz/cmd"
	"github.com/spf13/viper"
)

func init() {
	viper.SetEnvPrefix("CIFUZZ")
	viper.AutomaticEnv()
}

func main() {
	cmd.Execute()
}
