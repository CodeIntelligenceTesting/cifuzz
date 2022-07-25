package dialog

import (
	"sort"

	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"golang.org/x/exp/maps"
)

// Select offers the user a list of items (label:value) to select from and returns the value of the selected item
func Select(message string, items map[string]string) (string, error) {
	options := maps.Keys(items)
	sort.Strings(options)
	prompt := pterm.DefaultInteractiveSelect.WithOptions(options)
	prompt.DefaultText = message

	result, err := prompt.Show()
	if err != nil {
		return "", errors.WithStack(err)
	}

	return items[result], nil
}
