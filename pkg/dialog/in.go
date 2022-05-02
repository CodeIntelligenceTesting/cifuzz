package dialog

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"golang.org/x/exp/maps"
)

// Select offers the user a list of items (label:value) to select from and returns the value of the selected item
func Select(message string, items map[string]string, inReader io.Reader) (string, error) {
	prompt := promptui.Select{
		Label: message,
		Items: maps.Keys(items),
		Stdin: io.NopCloser(inReader),
	}
	_, result, err := prompt.Run()
	if err != nil {
		return "", errors.WithStack(err)
	}

	return items[result], nil
}

// Input asks the user for entering a string
func Input(message string, defaultValue string, inReader io.Reader) (string, error) {
	reader := bufio.NewReader(inReader)
	if defaultValue == "" {
		InfoF("%s: ", message)
	} else {
		InfoF("%s [%s]: ", message, defaultValue)
	}

	input, err := reader.ReadString('\n')
	if err != nil {
		return "", errors.WithStack(err)
	}
	input = strings.Replace(input, "\n", "", -1)

	if input == "" {
		return defaultValue, nil
	}
	return input, nil

}
