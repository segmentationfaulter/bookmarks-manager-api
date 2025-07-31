package utils

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type UserId string

func DecodeRequestBody[T any](r *http.Request) (*T, error) {
	result := new(T)
	err := json.NewDecoder(r.Body).Decode(result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func bearerToken(r *http.Request) (string, bool) {
	const prefix = "Bearer "
	auth := r.Header.Get("Authorization")
	if auth == "" || !strings.HasPrefix(auth, prefix) {
		return "", false
	}
	return strings.TrimPrefix(auth, prefix), true
}

func SigningKey() []byte {
	keyStr := os.Getenv("JWT_SIGNING_KEY")
	if keyStr == "" {
		panic("JWT signing key not set")
	}
	key, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		panic("Decoding of JWT signing key failed")
	}
	return key
}

func IsAuthenticated(r *http.Request) (UserId, int, error) {
	tokenStr, ok := bearerToken(r)
	if !ok {
		return "", http.StatusUnauthorized, errors.New("Missing/malformed token")
	}

	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return SigningKey(), nil
	})
	if err != nil {
		return "", http.StatusUnauthorized, errors.New("Invalid token: " + err.Error())
	}

	userId, err := token.Claims.GetSubject()
	if err != nil {
		return "", http.StatusUnauthorized, errors.New("Invalid token: " + err.Error())
	}

	return UserId(userId), http.StatusOK, nil
}
