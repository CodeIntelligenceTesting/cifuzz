package dialog

import (
	"io"

	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"golang.org/x/exp/maps"

	"code-intelligence.com/cifuzz/pkg/cmdutils"
)

// Select offers the user a list of items (label:value) to select from and returns the value of the selected item
func Select(message string, items map[string]string, inReader io.Reader) (string, error) {
	prompt := promptui.Select{
		Label: message,
		Items: maps.Keys(items),
		Stdin: io.NopCloser(inReader),
	}
	_, result, err := prompt.Run()
	if err == promptui.ErrInterrupt {
		return "", cmdutils.WrapSilentError(errors.WithStack(err))
	}
	if err != nil {
		return "", errors.WithStack(err)
	}

	return items[result], nil
}
