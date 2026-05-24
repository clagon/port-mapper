package auth

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
)

// GenerateToken returns a cryptographically-random token encoded as hex.
func GenerateToken(n int) (string, error) {
	if n <= 0 {
		n = 16
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// IsBearerToken reports whether the request value matches the expected bearer token.
func IsBearerToken(expected, provided string) bool {
	return strings.TrimSpace(expected) != "" && strings.TrimSpace(provided) == "Bearer "+strings.TrimSpace(expected)
}
