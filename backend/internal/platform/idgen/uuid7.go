// Package idgen gera identificadores UUIDv7 (02-modelo-de-dados.md §2):
// ordenáveis por tempo, para índices que não fragmentam como uuidv4.
package idgen

import "github.com/google/uuid"

func NewUUIDv7() (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}
