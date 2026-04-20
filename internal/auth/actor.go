package auth

import (
	"context"
	"errors"
	"fmt"
)

const (
	ClaimUserID     = "userId"
	ClaimUsername   = "username"
	ClaimSystemRole = "systemRole"

	SystemRoleAdmin    = "admin"
	SystemRoleReviewer = "reviewer"
	SystemRoleOperator = "operator"
	SystemRoleViewer   = "viewer"
)

var ErrUnauthenticated = errors.New("unauthenticated")

type currentUserContextKey struct{}

type CurrentUser struct {
	ID         string
	Username   string
	SystemRole string
}

func WithCurrentUser(ctx context.Context, user CurrentUser) context.Context {
	return context.WithValue(ctx, currentUserContextKey{}, user)
}

func CurrentUserFromContext(ctx context.Context) (CurrentUser, error) {
	if user, ok := ctx.Value(currentUserContextKey{}).(CurrentUser); ok {
		if user.ID != "" && user.SystemRole != "" {
			return user, nil
		}
	}

	userID, ok := stringFromContext(ctx, ClaimUserID)
	if !ok || userID == "" {
		return CurrentUser{}, ErrUnauthenticated
	}

	username, _ := stringFromContext(ctx, ClaimUsername)

	systemRole, ok := stringFromContext(ctx, ClaimSystemRole)
	if !ok || systemRole == "" {
		return CurrentUser{}, ErrUnauthenticated
	}

	return CurrentUser{
		ID:         userID,
		Username:   username,
		SystemRole: systemRole,
	}, nil
}

func IsSystemRole(role string) bool {
	switch role {
	case SystemRoleAdmin, SystemRoleReviewer, SystemRoleOperator, SystemRoleViewer:
		return true
	default:
		return false
	}
}

func stringFromContext(ctx context.Context, key string) (string, bool) {
	value := ctx.Value(key)
	if value == nil {
		return "", false
	}

	switch v := value.(type) {
	case string:
		return v, true
	case fmt.Stringer:
		return v.String(), true
	default:
		return "", false
	}
}
