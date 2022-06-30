package dialog

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

// InputFilename reads a filename from stdin, with tab-completion if
// available in the current shell
func InputFilename(reader io.Reader, message string, defaultValue string) (string, error) {
	// Print the message
	if defaultValue == "" {
		fmt.Printf("%s: \n", message)
	} else {
		fmt.Printf("%s [%s]: \n", message, defaultValue)
	}

	return readFilenameWithShellCompletion(reader, defaultValue)
}

func readline(reader io.Reader, defaultValue string) (string, error) {
	input, err := bufio.NewReader(reader).ReadString('\n')
	if err != nil {
		return "", errors.WithStack(err)
	}
	input = strings.Replace(input, "\n", "", -1)

	if input == "" {
		return defaultValue, nil
	}
	return input, nil
}

var shellCompletionArgs = map[string][]string{
	"bash": {"-c", "read -e && echo \"${REPLY}\""},
	"zsh":  {"-i", "-c", "vared -c REPLY"},
}

func readFilenameWithShellCompletion(reader io.Reader, defaultValue string) (string, error) {
	shellAbsPath, err := exec.LookPath(os.Getenv("SHELL"))
	if errors.Is(err, exec.ErrNotFound) {
		// Shell not found, fall back to reading without completion support.
		return readline(reader, defaultValue)
	}
	if err != nil {
		return "", errors.WithStack(err)
	}
	shellName := filepath.Base(shellAbsPath)
	completionArgs, supported := shellCompletionArgs[shellName]
	if !supported {
		// Shell not supported, fall back to reading without completion support.
		return readline(reader, defaultValue)
	}

	cmd := exec.Command(shellAbsPath, completionArgs...)
	cmd.Stderr = os.Stderr
	cmd.Stdin = reader
	out, err := cmd.Output()
	if _, ok := err.(*exec.ExitError); ok {
		// Fail silently in this case, which includes the user hitting Ctrl + C.
		return "", cmdutils.WrapSilentError(errors.WithStack(err))
	}
	if err != nil {
		return "", errors.WithStack(err)
	}
	input := strings.TrimSpace(string(out))

	if input == "" {
		return defaultValue, nil
	}
	return input, nil
}
