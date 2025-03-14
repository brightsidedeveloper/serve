package session

import (
	"testing"
	"time"
)

func TestGenerateToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "wow")

	token, err := GenerateToken("gamer420", time.Hour)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if token == "" {
		t.Errorf("Expected token to be generated")
	}
}

func TestVerifyToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "wow")

	token, err := GenerateToken("gamer420", time.Hour)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	claims, err := VerifyToken(token)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if claims == nil {
		t.Errorf("Expected claims to be non-nil")
	}

	if claims.Subject != "gamer420" {
		t.Errorf("Expected subject to be 'gamer420', got %v", claims.Subject)
	}

	if claims.ExpiresAt.Before(time.Now()) {
		t.Errorf("Expected expiration time to be in the future, got %v", claims.ExpiresAt)
	}

	durationUntilExpiration := time.Until(claims.ExpiresAt.Time)
	// We have to account for the time it takes to generate the token and when we verify it.
	if durationUntilExpiration < time.Hour-time.Second || durationUntilExpiration > time.Hour+time.Second {
		t.Errorf("Expected expiration to be approximately 1 hour from now, got %v", durationUntilExpiration)
	}
}
