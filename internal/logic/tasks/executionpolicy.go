package tasks

import (
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
