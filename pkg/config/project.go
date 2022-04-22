package config

import (
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

type projectConfig struct {
	LastUpdated string
}

const projectConfigFile = "cifuzz.yaml"

const projectConfigTemplate = `## Configuration for a CI Fuzz project
## Generated on {{.LastUpdated}}
`

// CreateProjectConfig creates a new project config in the given directory
func CreateProjectConfig(path string, fs *afero.Afero) (configpath string, err error) {

	// try to open the target file, returns error if already exists
	configpath = filepath.Join(path, projectConfigFile)
	f, err := fs.OpenFile(configpath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0644)
	if err != nil {
		if os.IsExist(err) {
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
