package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/crypto/argon2"
)

const (
	argonTime    = uint32(3)
	argonMemory  = uint32(64 * 1024)
	argonThreads = uint8(2)
	argonKeyLen  = uint32(32)
	saltLength   = 16
)

var ErrWeakPassword = errors.New("password harus memiliki minimal 14 karakter dan tidak boleh sama dengan username")

func HashPassword(password string) (string, error) {
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("membuat salt password: %w", err)
	}
	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", argonMemory, argonTime, argonThreads, base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(hash)), nil
}

func VerifyPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return false
	}
	parameters := map[string]uint64{}
	for _, item := range strings.Split(parts[3], ",") {
		pair := strings.SplitN(item, "=", 2)
		if len(pair) != 2 {
			return false
		}
		value, err := strconv.ParseUint(pair[1], 10, 32)
		if err != nil {
			return false
		}
		parameters[pair[0]] = value
	}
	memory, okM := parameters["m"]
	timeCost, okT := parameters["t"]
	threads, okP := parameters["p"]
	if !okM || !okT || !okP || memory < 8*1024 || memory > 256*1024 || timeCost < 1 || timeCost > 10 || threads < 1 || threads > 8 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) < 16 || len(salt) > 64 {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(expected) < 16 || len(expected) > 64 {
		return false
	}
	actual := argon2.IDKey([]byte(password), salt, uint32(timeCost), uint32(memory), uint8(threads), uint32(len(expected)))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

func ValidateNewPassword(username, password string) error {
	if len(password) < 14 || len(password) > 1024 || strings.EqualFold(username, password) {
		return ErrWeakPassword
	}
	var letters, nonLetters bool
	for _, char := range password {
		if unicode.IsLetter(char) {
			letters = true
		} else {
			nonLetters = true
		}
	}
	if !letters || !nonLetters {
		return ErrWeakPassword
	}
	return nil
}
