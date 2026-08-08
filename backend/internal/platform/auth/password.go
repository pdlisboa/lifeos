// Package auth implementa sessão e hash de senha (argon2id), conforme
// 01-arquitetura.md §8.2: auth própria e simples, sem Keycloak/Authelia.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argonTime    uint32 = 1
	argonMemory  uint32 = 64 * 1024 // KiB
	argonThreads uint8  = 4
	argonKeyLen  uint32 = 32
	saltLen             = 16
)

func HashPassword(password string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("gerar salt: %w", err)
	}
	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

func VerifyPassword(password, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, fmt.Errorf("formato de hash inválido")
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, fmt.Errorf("versão inválida: %w", err)
	}
	var memory, iterations uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &threads); err != nil {
		return false, fmt.Errorf("parâmetros inválidos: %w", err)
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("salt inválido: %w", err)
	}
	wantHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("hash inválido: %w", err)
	}
	gotHash := argon2.IDKey([]byte(password), salt, iterations, memory, threads, uint32(len(wantHash)))
	return subtle.ConstantTimeCompare(gotHash, wantHash) == 1, nil
}
