package vcs_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"code-intelligence.com/cifuzz/pkg/vcs"
	"code-intelligence.com/cifuzz/util/fileutil"
)

// Tests can't run in parallel as they change the current working directory.

func TestGitBranch(t *testing.T) {
	repo := createGitRepoWithCommits(t)
	defer os.RemoveAll(repo)
	err := os.Chdir(repo)
	require.NoError(t, err)

	branch, err := vcs.GitBranch()
	require.NoError(t, err)
	require.Equal(t, "main", branch)

	runGit(t, "", "checkout", "HEAD~")
	branch, err = vcs.GitBranch()
	require.NoError(t, err)
	require.Equal(t, "HEAD", branch)
}

func TestGitCommit(t *testing.T) {
	repo := createGitRepoWithCommits(t)
	defer os.RemoveAll(repo)
	err := os.Chdir(repo)
	require.NoError(t, err)

	commit1, err := vcs.GitCommit()
	require.NoError(t, err)
	// Verify that we obtain a full SHA-1 hash.
	require.Equalf(t, 40, len(commit1), "Expected full commit SHA, got %q", commit1)

	runGit(t, "", "checkout", "HEAD~")
	commit2, err := vcs.GitCommit()
	require.NoError(t, err)
	require.Equalf(t, 40, len(commit2), "Expected full commit SHA, got %q", commit2)

	require.NotEqual(t, commit1, commit2)
}

func TestGitIsDirty(t *testing.T) {
	repo := createGitRepoWithCommits(t)
	defer os.RemoveAll(repo)
	err := os.Chdir(repo)
	require.NoError(t, err)

	require.False(t, vcs.GitIsDirty())

	// Verify that modified files trigger a "dirty" state.
	err = os.WriteFile("empty_file", []byte("changed"), 0644)
	require.NoError(t, err)
	require.True(t, vcs.GitIsDirty())

	// Reset modifications.
	runGit(t, "", "checkout", "--", ".")
	require.False(t, vcs.GitIsDirty())

	// Verify that untracked files trigger a "dirty" state.
	err = fileutil.Touch("third_file")
	require.NoError(t, err)
	require.True(t, vcs.GitIsDirty())
}

func createGitRepoWithCommits(t *testing.T) string {
	t.Helper()

	repo, err := os.MkdirTemp("", "git-test-*")
	require.NoError(t, err)

	runGit(t, repo, "init")
	runGit(t, repo, "config", "user.email", "you@example.com")
	runGit(t, repo, "config", "user.name", "Your Name")

	// Ensure that the main branch is called "main" even with older Git versions.
	runGit(t, repo, "branch", "-M", "main")

	err = fileutil.Touch(filepath.Join(repo, "empty_file"))
	require.NoError(t, err)
	runGit(t, repo, "add", "empty_file")
	runGit(t, repo, "commit", "-m", "Initial commit")

	err = fileutil.Touch(filepath.Join(repo, "other_file"))
	require.NoError(t, err)
	runGit(t, repo, "add", "other_file")
	runGit(t, repo, "commit", "-m", "Second commit")

	return repo
}

func runGit(t *testing.T, repo string, args ...string) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repo
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	require.NoError(t, err)
}
