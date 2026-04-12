package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argonTime    uint32 = 2
	argonMemory  uint32 = 64 * 1024
	argonThreads uint8  = 4
	argonKeyLen  uint32 = 32
	saltLen             = 16
)

func HashPassword(password string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}
	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf(
		"argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argonMemory,
		argonTime,
		argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

func VerifyPassword(encodedHash, password string) (bool, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 5 {
		return false, errors.New("invalid hash format")
	}
	if parts[0] != "argon2id" {
		return false, errors.New("invalid hash algorithm")
	}

	var version int
	if _, err := fmt.Sscanf(parts[1], "v=%d", &version); err != nil {
		return false, errors.New("invalid version format")
	}
	if version != argon2.Version {
		return false, errors.New("argon2 version mismatch")
	}

	memory, iterations, threads, err := parseParams(parts[2])
	if err != nil {
		return false, err
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return false, errors.New("invalid salt encoding")
	}
	decodedHash, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, errors.New("invalid hash encoding")
	}

	calculated := argon2.IDKey([]byte(password), salt, iterations, memory, threads, uint32(len(decodedHash)))
	matched := subtle.ConstantTimeCompare(decodedHash, calculated) == 1
	return matched, nil
}

func parseParams(encoded string) (memory, iterations uint32, threads uint8, err error) {
	parts := strings.Split(encoded, ",")
	if len(parts) != 3 {
		return 0, 0, 0, errors.New("invalid argon2 params")
	}
	for _, part := range parts {
		kv := strings.Split(part, "=")
		if len(kv) != 2 {
			return 0, 0, 0, errors.New("invalid argon2 param key/value")
		}
		switch kv[0] {
		case "m":
			value, convErr := strconv.ParseUint(kv[1], 10, 32)
			if convErr != nil {
				return 0, 0, 0, errors.New("invalid memory value")
			}
			memory = uint32(value)
		case "t":
			value, convErr := strconv.ParseUint(kv[1], 10, 32)
			if convErr != nil {
				return 0, 0, 0, errors.New("invalid time value")
			}
			iterations = uint32(value)
		case "p":
			value, convErr := strconv.ParseUint(kv[1], 10, 8)
			if convErr != nil {
				return 0, 0, 0, errors.New("invalid threads value")
			}
			threads = uint8(value)
		default:
			return 0, 0, 0, errors.New("unknown argon2 param")
		}
	}
	return memory, iterations, threads, nil
}
