package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type AuthUser struct {
	ID       int64
	Username string
}

type AuthResult struct {
	User  AuthUser
	Error string
}

type tokenHeader struct {
	Algorithm string `json:"alg"`
}

type tokenPayload struct {
	Subject  string          `json:"sub"`
	Username json.RawMessage `json:"username"`
	Expires  json.RawMessage `json:"exp"`
}

func authenticate(r *http.Request) (AuthUser, string) {
	authorization := r.Header.Get("Authorization")
	if !strings.HasPrefix(authorization, "Bearer ") {
		return AuthUser{}, "missing bearer token"
	}

	token := strings.TrimPrefix(authorization, "Bearer ")
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return AuthUser{}, "invalid or expired token"
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return AuthUser{}, "invalid or expired token"
	}
	var header tokenHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil || header.Algorithm != "HS256" {
		return AuthUser{}, "invalid or expired token"
	}

	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return AuthUser{}, "invalid or expired token"
	}

	mac := hmac.New(sha256.New, []byte(jwtSecret))
	_, _ = mac.Write([]byte(parts[0] + "." + parts[1]))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return AuthUser{}, "invalid or expired token"
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return AuthUser{}, "invalid or expired token"
	}
	var payload tokenPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return AuthUser{}, "invalid or expired token"
	}

	var expiration float64
	if len(payload.Expires) == 0 ||
		json.Unmarshal(payload.Expires, &expiration) != nil ||
		math.IsNaN(expiration) ||
		math.IsInf(expiration, 0) ||
		float64(time.Now().Unix()) >= expiration {
		return AuthUser{}, "invalid or expired token"
	}

	id, err := strconv.ParseInt(payload.Subject, 10, 64)
	if err != nil || id < 1 || strconv.FormatInt(id, 10) != payload.Subject {
		return AuthUser{}, "invalid token payload"
	}

	var username string
	if len(payload.Username) == 0 || json.Unmarshal(payload.Username, &username) != nil {
		return AuthUser{}, "invalid token payload"
	}

	return AuthUser{ID: id, Username: username}, ""
}

func requireAuth(w http.ResponseWriter, r *http.Request) (AuthUser, bool) {
	user, message := authenticate(r)
	if message != "" {
		writeError(w, http.StatusUnauthorized, message)
		return AuthUser{}, false
	}
	return user, true
}

