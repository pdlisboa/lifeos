// Package auth implementa sessão e hash de senha (argon2id), conforme
// 01-arquitetura.md §8.2: auth própria e simples, sem Keycloak/Authelia.
package auth

import "github.com/alexedwards/argon2id"

func HashPassword(password string) (string, error) {
	return argon2id.CreateHash(password, argon2id.DefaultParams)
}

func VerifyPassword(password, encoded string) (bool, error) {
	match, _, err := argon2id.CheckHash(password, encoded)
	return match, err
}
