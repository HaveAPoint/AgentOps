package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"crypto/pbkdf2"
)

const (
	passwordHashAlgorithm = "pbkdf2_sha256"
	passwordHashIter      = 210000
	passwordSaltBytes     = 16
	passwordKeyBytes      = 32
)

var ErrInvalidPasswordHash = errors.New("invalid password hash")

func HashPassword(password string) (string, error) {
	salt := make([]byte, passwordSaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	key, err := pbkdf2.Key(sha256.New, password, salt, passwordHashIter, passwordKeyBytes)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(
		"%s$%d$%s$%s",
		passwordHashAlgorithm,
		passwordHashIter,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

func VerifyPassword(password string, encodedHash string) (bool, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 4 {
		return false, ErrInvalidPasswordHash
	}

	if parts[0] != passwordHashAlgorithm {
		return false, ErrInvalidPasswordHash
	}

	iterations, err := strconv.Atoi(parts[1])
	if err != nil || iterations <= 0 {
		return false, ErrInvalidPasswordHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		return false, ErrInvalidPasswordHash
	}

	expectedKey, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil || len(expectedKey) == 0 {
		return false, ErrInvalidPasswordHash
	}

	actualKey, err := pbkdf2.Key(sha256.New, password, salt, iterations, len(expectedKey))
	if err != nil {
		return false, err
	}

	return subtle.ConstantTimeCompare(actualKey, expectedKey) == 1, nil
}
