package cmdutils

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func MarkFlagsRequired(cmd *cobra.Command, flags ...string) {
	for _, flag := range flags {
		err := cmd.MarkFlagRequired(flag)
		if err != nil {
			panic(err)
		}
	}
}

func ViperMustBindPFlag(key string, flag *pflag.Flag) {
	err := viper.BindPFlag(key, flag)
	if err != nil {
		panic(err)
	}
}
