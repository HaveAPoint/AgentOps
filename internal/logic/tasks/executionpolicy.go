package tasks

import (
	"fmt"
	"path"
	"strings"

	"agentops/internal/model"
)

func validateExecutionPolicy(task *model.Task, policy *model.TaskPolicy) error {
	if policy == nil {
		return ErrExecutionPolicyRequired
	}

	if err := validatePolicyPaths(policy); err != nil {
		return err
	}

	switch task.Mode {
	case TaskModeAnalyze:
		return nil
	case TaskModePatch:
		return validatePatchPolicy(policy)
	default:
		return ErrInvalidMode
	}
}

func validatePatchPolicy(policy *model.TaskPolicy) error {
	if !hasNonEmptyPath(policy.AllowedPaths) {
		return ErrAllowedPathRequiredForPatch
	}

	return nil
}

func validatePolicyPaths(policy *model.TaskPolicy) error {
	for _, p := range policy.AllowedPaths {
		if !isSafePolicyPath(p) {
			return ErrInvalidPolicyPath
		}
	}

	for _, p := range policy.DeniedPaths {
		if !isSafePolicyPath(p) {
			return ErrInvalidPolicyPath
		}
	}

	return nil
}

func hasNonEmptyPath(paths []string) bool {
	for _, p := range paths {
		if strings.TrimSpace(p) != "" {
			return true
		}
	}
	return false
}

func isSafePolicyPath(p string) bool {
	p = strings.TrimSpace(p)
	if p == "" {
		return false
	}

	if strings.HasPrefix(p, "/") || strings.Contains(p, "\\") {
		return false
	}

	clean := path.Clean(p)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return false
	}

	return true
}

func validateChangedFilesAllowed(mode string, policy *model.TaskPolicy, changedFiles []string) error {
	for _, file := range changedFiles {
		if matchesAnyPolicyPath(file, policy.DeniedPaths) {
			return fmt.Errorf("%w: %s", ErrChangedFileDenied, file)
		}

		if mode == TaskModePatch && !matchesAnyPolicyPath(file, policy.AllowedPaths) {
			return fmt.Errorf("%w: %s", ErrChangedFileNotAllowed, file)
		}
	}

	return nil
}

func matchesAnyPolicyPath(file string, policyPaths []string) bool {
	file = strings.TrimSpace(file)
	if file == "" {
		return false
	}

	for _, policyPath := range policyPaths {
		policyPath = strings.TrimSpace(policyPath)
		if policyPath == "" {
			continue
		}

		if file == policyPath || strings.HasPrefix(file, policyPath+"/") {
			return true
		}
	}

	return false
}
