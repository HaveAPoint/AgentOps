package auth

import (
	"errors"
	"testing"
)

func TestPasswordHashAndVerify(t *testing.T) {
	hash, err := HashPassword("secret")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	ok, err := VerifyPassword("secret", hash)
	if err != nil {
		t.Fatalf("verify password: %v", err)
	}
	if !ok {
		t.Fatal("expected password to verify")
	}

	ok, err = VerifyPassword("wrong", hash)
	if err != nil {
		t.Fatalf("verify wrong password: %v", err)
	}
	if ok {
		t.Fatal("expected wrong password to fail")
	}
}

func TestVerifyPasswordRejectsInvalidHash(t *testing.T) {
	ok, err := VerifyPassword("secret", "not-a-valid-hash")
	if !errors.Is(err, ErrInvalidPasswordHash) {
		t.Fatalf("expected ErrInvalidPasswordHash, got %v", err)
	}
	if ok {
		t.Fatal("expected invalid hash not to verify")
	}
}
