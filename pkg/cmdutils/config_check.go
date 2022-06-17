package cmdutils

import (
	"github.com/spf13/cobra"
)

func DisableConfigCheck(cmd *cobra.Command) {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}

	cmd.Annotations["skipConfigCheck"] = "true"
}

func NeedsConfig(cmd *cobra.Command) bool {
	switch cmd.Name() {
	case "help", cobra.ShellCompRequestCmd, cobra.ShellCompNoDescRequestCmd:
		return false
	}

	for c := cmd; c != nil; c = c.Parent() {
		if c.Annotations != nil && c.Annotations["skipConfigCheck"] == "true" {
			return false
		}
	}

	return true
}
