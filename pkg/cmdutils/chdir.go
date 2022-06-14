package cmdutils

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

func Chdir() error {
	workdir := viper.GetString("directory")
	if workdir == "" {
		// The --directory flag was not set, nothing to do here
		return nil
	}
	err := os.Chdir(workdir)
	return errors.WithStack(err)
}
