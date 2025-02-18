package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMakeAndValidateJWT(t *testing.T) {
	secret := "mysecret"
	userID := uuid.New()

	token, err := MakeJWT(userID, secret, time.Minute)
	if err != nil {
		t.Fatalf("MakeJWT error: %v", err)
	}

	validateUserID, err := ValidateJWT(token, secret)
	if err != nil {
		t.Fatalf("ValidateJWT error: %v", err)
	}

	if validateUserID != userID {
		t.Fatalf("expected userID %s, got %s", userID, validateUserID)
	}
}

func TextExpiredJWT(t *testing.T) {
	secret := "mysecret"
	userID := uuid.New()

	token, err := MakeJWT(userID, secret, -time.Minute)
	if err != nil {
		t.Fatalf("MakeJWT error: %v", err)
	}

	_, err = ValidateJWT(token, secret)
	if err == nil {
		t.Fatalf("expected token to be expired, but validation passed")
	}
}

func TestWrongSecretJWT(t *testing.T) {
	secret := "mysecret"
	wrongSecret := "wrongsecret"
	userID := uuid.New()

	token, err := MakeJWT(userID, secret, time.Minute)
	if err != nil {
		t.Fatalf("MakeJWT error: %v", err)
	}

	_, err = ValidateJWT(token, wrongSecret)
	if err == nil {
		t.Fatalf("expected validation to fail with wrong secret, but it passed")
	}
}
