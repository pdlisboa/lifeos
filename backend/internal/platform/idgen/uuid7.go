// Package idgen gera identificadores UUIDv7 (02-modelo-de-dados.md §2):
// ordenáveis por tempo, para índices que não fragmentam como uuidv4.
package idgen

import (
	"crypto/rand"
	"fmt"
)

func NewUUIDv7() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("gerar aleatoriedade: %w", err)
	}

	ms := nowUnixMilli()
	b[0] = byte(ms >> 40)
	b[1] = byte(ms >> 32)
	b[2] = byte(ms >> 24)
	b[3] = byte(ms >> 16)
	b[4] = byte(ms >> 8)
	b[5] = byte(ms)

	b[6] = (b[6] & 0x0F) | 0x70 // versão 7
	b[8] = (b[8] & 0x3F) | 0x80 // variante RFC 4122

	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
