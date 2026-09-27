package api

import (
	"net/http"
	"strings"
)

// ExtractBearer extracts the Bearer token from the "Authorization: Bearer <token>" header.
func ExtractBearer(r *http.Request) (string, bool) {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return "", false
	}
	token := strings.TrimPrefix(h, "Bearer ")
	token = strings.TrimSpace(token)
	if token == "" {
		return "", false
	}
	return token, true
}

// GetUserByToken looks up a user by their Bearer token.
// Returns the user and true if found, zero value and false otherwise.
func GetUserByToken(token string) (*User, bool) {
	u, ok := Users[token]
	if !ok {
		return nil, false
	}
	return &u, true
}
