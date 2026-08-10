package functions

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base32"

	"golang.org/x/crypto/argon2"
)

func MakeTokens() (string, error) {
	bytes := make([]byte, 15)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	sessionId := base32.StdEncoding.EncodeToString(bytes)
	return sessionId, nil
}

func MakeHash(password string) string {
	tmp := []byte(password)
	salt := []byte("testsalt")

	hash := argon2.IDKey(tmp, salt, 2, 19*1024, 1, 32)
	return base32.StdEncoding.EncodeToString(hash)
}

func ValidateHash(password string, savedHash string) bool {
	hash := MakeHash(password)
	return subtle.ConstantTimeCompare([]byte(hash), []byte(savedHash)) == 1
}
