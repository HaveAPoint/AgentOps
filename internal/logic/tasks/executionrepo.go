package tasks

import (
	"path/filepath"
	"strings"
)

func validateExecutionRepo(repoPath string, allowedRepoPaths []string) error {
	repoAbs, err := normalizeRepoPath(repoPath)
	if err != nil {
		return ErrRepoNotAllowed
	}

	for _, allowedRepoPath := range allowedRepoPaths {
		allowedAbs, err := normalizeRepoPath(allowedRepoPath)
		if err != nil {
			return ErrRepoNotAllowed
		}

		if repoAbs == allowedAbs {
			return nil
		}
	}

	return ErrRepoNotAllowed
}

func normalizeRepoPath(repoPath string) (string, error) {
	repoPath = strings.TrimSpace(repoPath)
	if repoPath == "" {
		return "", ErrRepoNotAllowed
	}

	abs, err := filepath.Abs(repoPath)
	if err != nil {
		return "", err
	}

	return filepath.Clean(abs), nil
}
