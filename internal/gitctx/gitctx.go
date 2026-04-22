package gitctx

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

var ErrNotGitRepo = errors.New("repoPath must point to a git repository")
var ErrNoCommitYet = errors.New("git repository has no commit yet")

type Context struct {
	Branch     string
	HeadCommit string
	Dirty      bool
}

func Read(repoPath string) (*Context, error) {
	if _, err := gitOutput(repoPath, "rev-parse", "--show-toplevel"); err != nil {
		return nil, ErrNotGitRepo
	}

	branch, err := gitOutput(repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return nil, ErrNoCommitYet
	}

	headCommit, err := gitOutput(repoPath, "rev-parse", "HEAD")
	if err != nil {
		return nil, ErrNoCommitYet
	}

	status, err := gitOutput(repoPath, "status", "--porcelain")
	if err != nil {
		return nil, err
	}

	return &Context{
		Branch:     branch,
		HeadCommit: headCommit,
		Dirty:      strings.TrimSpace(status) != "",
	}, nil
}

func ChangedFiles(repoPath string) ([]string, error) {
	// diff --name-only HEAD covers tracked changes.
	// ls-files --others --exclude-standard covers untracked files.
	// Together they describe the current changed file set.

	if _, err := gitOutput(repoPath, "rev-parse", "--show-toplevel"); err != nil {
		return nil, ErrNotGitRepo
	}

	tracked, err := gitOutput(repoPath, "diff", "--name-only", "HEAD")
	if err != nil {
		return nil, err
	}

	untracked, err := gitOutput(repoPath, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}

	return mergeGitPathLines(tracked, untracked), nil
}

func mergeGitPathLines(outputs ...string) []string {
	seen := make(map[string]struct{})
	files := make([]string, 0)

	for _, output := range outputs {
		for _, line := range strings.Split(output, "\n") {
			file := strings.TrimSpace(line)
			if file == "" {
				continue
			}

			if _, ok := seen[file]; ok {
				continue
			}

			seen[file] = struct{}{}
			files = append(files, file)
		}
	}

	return files
}

func gitOutput(repoPath string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", repoPath}, args...)...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s failed: %s", strings.Join(args, " "), msg)
	}

	return strings.TrimSpace(string(out)), nil
}
