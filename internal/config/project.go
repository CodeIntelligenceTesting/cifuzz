package config

import (
	_ "embed"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"code-intelligence.com/cifuzz/util/fileutil"
	"code-intelligence.com/cifuzz/util/stringutil"
)

const (
	BuildSystemAuto    string = "auto"
	BuildSystemCMake   string = "cmake"
	BuildSystemUnknown string = "unknown"
)

var buildSystemTypes = []string{BuildSystemAuto, BuildSystemCMake, BuildSystemUnknown}

type projectConfig struct {
	LastUpdated string
	BuildSystem string `yaml:"build_system"`
}

const projectConfigFile = "cifuzz.yaml"

//go:embed cifuzz.yaml.tmpl
var projectConfigTemplate string

// CreateProjectConfig creates a new project config in the given directory
func CreateProjectConfig(projectDir string) (configpath string, err error) {

	// try to open the target file, returns error if already exists
	configpath = filepath.Join(projectDir, projectConfigFile)
	f, err := os.OpenFile(configpath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return configpath, errors.WithStack(err)
		}
		return "", errors.WithStack(err)
	}

	// setup config struct with (default) values
	config := projectConfig{
		LastUpdated: time.Now().Format("2006-01-02"),
	}

	// parse the template and write it to config file
	t, err := template.New("project_config").Parse(projectConfigTemplate)
	if err != nil {
		return "", errors.WithStack(err)
	}
	if err = t.Execute(f, config); err != nil {
		return "", errors.WithStack(err)
	}

	return
}

func ReadProjectConfig(projectDir string) (*projectConfig, error) {
	configpath := filepath.Join(projectDir, projectConfigFile)

	bytes, err := ioutil.ReadFile(configpath)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	config := &projectConfig{}
	err = yaml.Unmarshal(bytes, config)
	if err != nil {
		return nil, errors.Wrapf(err, "Error parsing config file %s", configpath)
	}

	// Set defaults
	if config.BuildSystem == "" {
		config.BuildSystem = BuildSystemAuto
	}

	// Validate types
	if !stringutil.Contains(buildSystemTypes, config.BuildSystem) {
		return nil, errors.Errorf("Invalid build system \"%s\"", config.BuildSystem)
	}

	if config.BuildSystem == BuildSystemAuto {
		isCMakeProject, err := fileutil.Exists(filepath.Join(projectDir, "CMakeLists.txt"))
		if err != nil {
			return nil, err
		}
		if isCMakeProject {
			config.BuildSystem = BuildSystemCMake
		} else {
			config.BuildSystem = BuildSystemUnknown
		}
	}

	return config, nil
}
