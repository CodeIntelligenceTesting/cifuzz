package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/otiai10/copy"
	"github.com/pkg/errors"

	"code-intelligence.com/cifuzz/pkg/log"
)

const nameCrashingInput = "crashing-input"
const nameJsonFile = "finding.json"
const nameFindingDir = ".cifuzz-findings"

type Finding struct {
	Name               string        `json:"name,omitempty"`
	Type               ErrorType     `json:"type,omitempty"`
	InputData          []byte        `json:"input_data,omitempty"`
	Logs               []string      `json:"logs,omitempty"`
	Details            string        `json:"details,omitempty"`
	HumanReadableInput string        `json:"human_readable_input,omitempty"`
	MoreDetails        *ErrorDetails `json:"more_details,omitempty"`
	Tag                uint64        `json:"tag,omitempty"`
	ShortDescription   string        `json:"short_description,omitempty"`
	InputFile          string
}

func (f *Finding) GetDetails() string {
	if f != nil {
		return f.Details
	}
	return ""
}

func (f *Finding) Save() error {
	findingDir := filepath.Join(nameFindingDir, f.Name)

	if err := os.MkdirAll(findingDir, 0755); err != nil {
		return errors.WithStack(err)
	}

	if f.InputFile != "" {
		if err := f.moveInputFile(findingDir); err != nil {
			return err
		}
	}

	if err := f.saveJson(findingDir); err != nil {
		return err
	}

	return nil
}

func (f *Finding) saveJson(findingDir string) error {
	bytes, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return errors.WithStack(err)
	}

	jsonPath := filepath.Join(findingDir, nameJsonFile)
	if err := os.WriteFile(jsonPath, bytes, 0644); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// move the input file to a new location and update
// the finding and the logs
func (f *Finding) moveInputFile(findingDir string) error {
	newPath := filepath.Join(findingDir, nameCrashingInput)

	// We don't use os.Rename to avoid errors when source and target
	// are not on the same mounted filesystem.
	if err := copy.Copy(f.InputFile, newPath); err != nil {
		return errors.WithStack(err)
	}
	if err := os.Remove(f.InputFile); err != nil {
		return errors.WithStack(err)
	}

	for i, line := range f.Logs {
		f.Logs[i] = strings.ReplaceAll(line, f.InputFile, newPath)
	}
	log.Debugf("moved input file from %s to %s", f.InputFile, newPath)
	f.InputFile = newPath
	return nil
}
