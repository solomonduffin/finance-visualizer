// Package auth provides bcrypt password verification and JWT token helpers
// for the finance-visualizer authentication system.
package auth

import (
	"time"

	"github.com/go-chi/jwtauth/v5"
	"golang.org/x/crypto/bcrypt"
)

// tokenAuth is the package-level JWTAuth instance initialized by Init.
var tokenAuth *jwtauth.JWTAuth

// Init initializes the package-level JWTAuth instance with the given HMAC-HS256 secret.
// Must be called once at application startup before any token operations.
func Init(secret string) {
	tokenAuth = jwtauth.New("HS256", []byte(secret), nil)
}

// TokenAuth returns the initialized JWTAuth instance for use in router middleware setup.
// Returns nil if Init has not been called.
func TokenAuth() *jwtauth.JWTAuth {
	return tokenAuth
}

// VerifyPassword compares a bcrypt hash against a plaintext password.
// Returns nil on match, an error on mismatch or empty hash.
func VerifyPassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// HashPassword generates a bcrypt hash of the given plaintext password using cost 12.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CreateToken encodes a JWT with a 30-day expiry using the initialized tokenAuth.
// Returns the signed JWT string. Init must be called before CreateToken.
func CreateToken() (string, error) {
	claims := map[string]interface{}{
		"exp": time.Now().Add(30 * 24 * time.Hour).UTC().Unix(),
	}
	_, tokenString, err := tokenAuth.Encode(claims)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}
