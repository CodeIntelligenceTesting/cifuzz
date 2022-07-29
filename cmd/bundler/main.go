package main

import (
	"github.com/spf13/pflag"
	"os"
	"path/filepath"

	"code-intelligence.com/cifuzz/pkg/install"
	"code-intelligence.com/cifuzz/pkg/log"
)

func main() {
	version := pflag.String("version", "v0.0.1", "the target version of cifuzz")
	pflag.Parse()

	projectDir, err := install.FindProjectDir()
	if err != nil {
		log.Error(err, err.Error())
		os.Exit(1)
	}
	targetDir := filepath.Join(projectDir, "installer-bundle", "bundle")

	opts := install.Options{
		Version:   *version,
		TargetDir: targetDir,
	}
	bundler, err := install.NewInstallationBundler(opts)
	if err != nil {
		log.Error(err, err.Error())
		os.Exit(1)
	}

	err = bundler.BuildCIFuzzAndDeps()
	if err != nil {
		log.Error(err, err.Error())
		os.Exit(1)
	}
}
