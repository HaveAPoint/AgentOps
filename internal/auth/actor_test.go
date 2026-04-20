package auth

import (
	"context"
	"errors"
	"testing"
)

func TestCurrentUserFromContextPrefersTypedUser(t *testing.T) {
	ctx := context.WithValue(context.Background(), ClaimUserID, "claim-user")
	ctx = WithCurrentUser(ctx, CurrentUser{
		ID:         "typed-user",
		Username:   "typed-user",
		SystemRole: SystemRoleAdmin,
	})

	user, err := CurrentUserFromContext(ctx)
	if err != nil {
		t.Fatalf("current user: %v", err)
	}
	if user.ID != "typed-user" || user.SystemRole != SystemRoleAdmin {
		t.Fatalf("unexpected user: %+v", user)
	}
}

func TestCurrentUserFromContextReadsJWTClaims(t *testing.T) {
	ctx := context.WithValue(context.Background(), ClaimUserID, "operator-1")
	ctx = context.WithValue(ctx, ClaimUsername, "operator")
	ctx = context.WithValue(ctx, ClaimSystemRole, SystemRoleOperator)

	user, err := CurrentUserFromContext(ctx)
	if err != nil {
		t.Fatalf("current user: %v", err)
	}
	if user.ID != "operator-1" || user.Username != "operator" || user.SystemRole != SystemRoleOperator {
		t.Fatalf("unexpected user: %+v", user)
	}
}

func TestCurrentUserFromContextRequiresUserIDAndRole(t *testing.T) {
	_, err := CurrentUserFromContext(context.Background())
	if !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("expected ErrUnauthenticated, got %v", err)
	}

	ctx := context.WithValue(context.Background(), ClaimUserID, "user-1")
	_, err = CurrentUserFromContext(ctx)
	if !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("expected ErrUnauthenticated, got %v", err)
	}
}
