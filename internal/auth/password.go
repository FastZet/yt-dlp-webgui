package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2id parameters. These are intentionally conservative for a
// single-user personal tool where login attempts are rare.
const (
	argonTime    = 2
	argonMemory  = 64 * 1024 // 64 MB
	argonThreads = 4
	argonKeyLen  = 32
	argonSaltLen = 16
)

// HashPassword generates an Argon2id hash of the given plaintext password.
// The returned string is self-contained and includes the salt, so it can
// be stored directly in APP_PASSWORD_HASH.
//
// Format: $argon2id$v=19$m=65536,t=2,p=4$<salt_b64>$<hash_b64>
func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generating salt: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		argonTime,
		argonMemory,
		argonThreads,
		argonKeyLen,
	)

	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argonMemory,
		argonTime,
		argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)

	return encoded, nil
}

// VerifyPassword checks a plaintext password against a stored Argon2id hash.
// Returns nil on match, an error otherwise.
func VerifyPassword(password, encoded string) error {
	salt, storedHash, params, err := decodeHash(encoded)
	if err != nil {
		return fmt.Errorf("decoding stored hash: %w", err)
	}

	candidateHash := argon2.IDKey(
		[]byte(password),
		salt,
		params.time,
		params.memory,
		params.threads,
		params.keyLen,
	)

	// Constant-time comparison to prevent timing attacks.
	if subtle.ConstantTimeCompare(candidateHash, storedHash) != 1 {
		return errors.New("password does not match")
	}

	return nil
}

// argonParams holds decoded parameters from a stored hash string.
type argonParams struct {
	time    uint32
	memory  uint32
	threads uint8
	keyLen  uint32
}

// decodeHash parses an encoded Argon2id hash string into its components.
func decodeHash(encoded string) (salt, hash []byte, params *argonParams, err error) {
	parts := strings.Split(encoded, "$")
	// Expected format: ["", "argon2id", "v=19", "m=65536,t=2,p=4", "<salt>", "<hash>"]
	if len(parts) != 6 {
		return nil, nil, nil, errors.New("invalid hash format: expected 6 segments")
	}

	if parts[1] != "argon2id" {
		return nil, nil, nil, fmt.Errorf("unsupported algorithm: %s", parts[1])
	}

	var version int
	if _, err = fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return nil, nil, nil, fmt.Errorf("parsing version: %w", err)
	}
	if version != argon2.Version {
		return nil, nil, nil, fmt.Errorf("incompatible argon2 version: %d", version)
	}

	params = &argonParams{}
	if _, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d",
		&params.memory, &params.time, &params.threads); err != nil {
		return nil, nil, nil, fmt.Errorf("parsing parameters: %w", err)
	}

	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("decoding salt: %w", err)
	}

	hash, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("decoding hash: %w", err)
	}

	params.keyLen = uint32(len(hash))

	return salt, hash, params, nil
}
