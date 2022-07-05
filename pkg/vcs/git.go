package vcs

import (
	"os/exec"
	"strings"

	"github.com/pkg/errors"

	"code-intelligence.com/cifuzz/pkg/log"
)

// GitCommit returns the full SHA of the current commit if the working directory is contained in a Git repository.
func GitCommit() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	commit, err := cmd.Output()
	if err != nil {
		return "", errors.WithStack(err)
	}
	log.Debugf("Current Git commit: %s", string(commit))
	return strings.TrimSpace(string(commit)), nil
}

// GitBranch returns the name of the current branch if the working directory is contained in a Git repository.
func GitBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	commit, err := cmd.Output()
	if err != nil {
		return "", errors.WithStack(err)
	}
	log.Debugf("Current Git branch: %s", string(commit))
	return strings.TrimSpace(string(commit)), nil
}

// GitIsDirty returns true if and only if the current working directory is contained in a Git repository that has
// uncommitted changes and/or untracked files.
func GitIsDirty() bool {
	cmd := exec.Command("git", "status", "--porcelain")
	commit, err := cmd.CombinedOutput()
	if err != nil {
		log.Debugf("failed to run git status --porcelain: %+v", err)
	}
	return len(strings.TrimSpace(string(commit))) != 0
}
